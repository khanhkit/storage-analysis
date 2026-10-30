package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"storageanalysis/internal/cache"
	"storageanalysis/internal/model"
)

type ScanOptions struct {
	MaxDepth int
	UseCache bool
	CacheStrategy string
	ResetStats bool
}

type ScanResult struct {
	Node      model.Node
	FromCache bool
	ModTime   time.Time
	ScannedAt time.Time
	Warnings  []string
}

type Scanner struct {
	cache *cache.Cache
	sem   chan struct{}
	filesScanned atomic.Int64
	dirsScanned  atomic.Int64
	scanning     atomic.Bool
	startedAt    atomic.Value
}

func New(store *cache.Cache, workers int) *Scanner {
	if workers <= 0 {
		workers = 4
	}

	scanner := &Scanner{
		cache: store,
		sem:   make(chan struct{}, workers),
	}
	scanner.startedAt.Store(time.Time{})
	return scanner
}

type ScanStats struct {
	Scanning     bool
	StartedAt    time.Time
	FilesScanned int64
	DirsScanned  int64
}

func (s *Scanner) Stats() ScanStats {
	startedAt, _ := s.startedAt.Load().(time.Time)
	return ScanStats{
		Scanning:     s.scanning.Load(),
		StartedAt:    startedAt,
		FilesScanned: s.filesScanned.Load(),
		DirsScanned:  s.dirsScanned.Load(),
	}
}

func (s *Scanner) beginScan(reset bool) {
	if reset {
		s.filesScanned.Store(0)
		s.dirsScanned.Store(0)
		s.startedAt.Store(time.Now())
	}
	s.scanning.Store(true)
}

func (s *Scanner) endScan() {
	s.scanning.Store(false)
}

func (s *Scanner) tryAcquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Scanner) release() {
	<-s.sem
}

func (s *Scanner) Scan(ctx context.Context, path string, opts ScanOptions) (ScanResult, error) {
	if path == "" {
		return ScanResult{}, errors.New("path is required")
	}

	s.beginScan(opts.ResetStats)
	defer s.endScan()

	info, err := os.Stat(path)
	if err != nil {
		return ScanResult{}, err
	}

	if !info.IsDir() {
		s.filesScanned.Add(1)
		node := model.Node{
			Path:       path,
			Name:       info.Name(),
			Type:       model.NodeFile,
			SizeBytes:  info.Size(),
			ModTime:    info.ModTime(),
			ChildCount: 0,
		}

		return ScanResult{
			Node:      node,
			FromCache: false,
			ModTime:   info.ModTime(),
			ScannedAt: time.Now(),
		}, nil
	}

	if isReparsePoint(info) {
		s.dirsScanned.Add(1)
		node := model.Node{
			Path:       path,
			Name:       info.Name(),
			Type:       model.NodeDir,
			SizeBytes:  0,
			ModTime:    info.ModTime(),
			ChildCount: 0,
		}

		return ScanResult{
			Node:      node,
			FromCache: false,
			ModTime:   info.ModTime(),
			ScannedAt: time.Now(),
			Warnings:  []string{"skipped reparse point: " + path},
		}, nil
	}

	modTime := info.ModTime()
	if opts.UseCache {
		cached, ok := s.cache.Get(path)
		if ok {
			if useDeepCache(opts.CacheStrategy) && !cached.DeepModTime.IsZero() {
				currentDeep, err := s.computeDeepModTime(ctx, path)
				if err == nil && currentDeep.Equal(cached.DeepModTime) {
					return ScanResult{
						Node:      cached.Tree,
						FromCache: true,
						ModTime:   cached.ModTime,
						ScannedAt: cached.ScannedAt,
					}, nil
				}
			} else if cached.ModTime.Equal(modTime) {
				return ScanResult{
					Node:      cached.Tree,
					FromCache: true,
					ModTime:   cached.ModTime,
					ScannedAt: cached.ScannedAt,
				}, nil
			}
		}
	}

	node, deepModTime, warnings, err := s.scanDir(ctx, path, info, 0, opts)
	if err != nil {
		return ScanResult{}, err
	}

	entry := cache.Entry{
		Path:      path,
		ModTime:   modTime,
		DeepModTime: deepModTime,
		Tree:      node,
		ScannedAt: time.Now(),
	}

	s.cache.Set(entry)

	return ScanResult{
		Node:      node,
		FromCache: false,
		ModTime:   modTime,
		ScannedAt: entry.ScannedAt,
		Warnings:  warnings,
	}, nil
}

func (s *Scanner) scanDir(ctx context.Context, path string, info os.FileInfo, depth int, opts ScanOptions) (model.Node, time.Time, []string, error) {
	if err := ctx.Err(); err != nil {
		return model.Node{}, time.Time{}, nil, err
	}

	s.dirsScanned.Add(1)

	entries, err := os.ReadDir(path)
	if err != nil {
		node := model.Node{
			Path:       path,
			Name:       info.Name(),
			Type:       model.NodeDir,
			SizeBytes:  0,
			ModTime:    info.ModTime(),
			ChildCount: 0,
		}
		return node, info.ModTime(), []string{err.Error()}, nil
	}

	includeChildren := opts.MaxDepth < 0 || depth < opts.MaxDepth
	childNodes := make([]model.Node, 0, len(entries))
	warnings := make([]string, 0)
	deepModTime := info.ModTime()

	var total int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	processEntry := func(entry os.DirEntry, childPath string) {
		node, childDeepModTime, childWarnings, err := s.scanEntry(ctx, childPath, entry, depth+1, opts)
		if err != nil {
			mu.Lock()
			warnings = append(warnings, err.Error())
			mu.Unlock()
			return
		}

		mu.Lock()
		total += node.SizeBytes
		if childDeepModTime.After(deepModTime) {
			deepModTime = childDeepModTime
		}
		warnings = append(warnings, childWarnings...)
		if includeChildren {
			childNodes = append(childNodes, node)
		}
		mu.Unlock()
	}

	for _, entry := range entries {
		entry := entry
		childPath := filepath.Join(path, entry.Name())
		if s.tryAcquire() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer s.release()
				processEntry(entry, childPath)
			}()
			continue
		}
		processEntry(entry, childPath)
	}

	wg.Wait()

	node := model.Node{
		Path:       path,
		Name:       info.Name(),
		Type:       model.NodeDir,
		SizeBytes:  total,
		ModTime:    info.ModTime(),
		ChildCount: len(entries),
		Children:   childNodes,
	}

	return node, deepModTime, warnings, nil
}

func (s *Scanner) scanEntry(ctx context.Context, path string, entry os.DirEntry, depth int, opts ScanOptions) (model.Node, time.Time, []string, error) {
	if err := ctx.Err(); err != nil {
		return model.Node{}, time.Time{}, nil, err
	}

	if entry.Type()&os.ModeSymlink != 0 {
		info, err := entry.Info()
		if err != nil {
			return model.Node{}, time.Time{}, nil, err
		}

		node := model.Node{
			Path:       path,
			Name:       entry.Name(),
			Type:       model.NodeFile,
			SizeBytes:  0,
			ModTime:    info.ModTime(),
			ChildCount: 0,
		}

		return node, info.ModTime(), []string{"skipped symlink: " + path}, nil
	}

	info, err := entry.Info()
	if err != nil {
		return model.Node{}, time.Time{}, nil, err
	}

	if isReparsePoint(info) {
		if info.IsDir() {
			s.dirsScanned.Add(1)
		} else {
			s.filesScanned.Add(1)
		}
		node := model.Node{
			Path:       path,
			Name:       entry.Name(),
			Type:       model.NodeFile,
			SizeBytes:  0,
			ModTime:    info.ModTime(),
			ChildCount: 0,
		}

		return node, info.ModTime(), []string{"skipped reparse point: " + path}, nil
	}

	if info.IsDir() {
		return s.scanDir(ctx, path, info, depth, opts)
	}

	s.filesScanned.Add(1)

	node := model.Node{
		Path:       path,
		Name:       entry.Name(),
		Type:       model.NodeFile,
		SizeBytes:  info.Size(),
		ModTime:    info.ModTime(),
		ChildCount: 0,
	}

	return node, info.ModTime(), nil, nil
}

func useDeepCache(strategy string) bool {
	return strings.EqualFold(strategy, "deep")
}

func (s *Scanner) computeDeepModTime(ctx context.Context, root string) (time.Time, error) {
	info, err := os.Stat(root)
	if err != nil {
		return time.Time{}, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := cap(s.sem)
	if workerCount <= 0 {
		workerCount = 4
	}

	dirCh := make(chan string, workerCount*2)
	var tasks sync.WaitGroup
	var workers sync.WaitGroup
	var latestMu sync.Mutex
	latest := info.ModTime()

	var firstErr error
	var errOnce sync.Once
	recordErr := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	worker := func() {
		defer workers.Done()
		for path := range dirCh {
			if ctx.Err() != nil {
				tasks.Done()
				continue
			}

			entries, readErr := os.ReadDir(path)
			if readErr != nil {
				recordErr(readErr)
				tasks.Done()
				continue
			}

			for _, entry := range entries {
				if ctx.Err() != nil {
					break
				}
				if entry.Type()&os.ModeSymlink != 0 {
					continue
				}

				entryInfo, infoErr := entry.Info()
				if infoErr != nil {
					recordErr(infoErr)
					continue
				}

				latestMu.Lock()
				if entryInfo.ModTime().After(latest) {
					latest = entryInfo.ModTime()
				}
				latestMu.Unlock()

				if entryInfo.IsDir() && ctx.Err() == nil {
					childPath := filepath.Join(path, entry.Name())
					tasks.Add(1)
					dirCh <- childPath
				}
			}

			tasks.Done()
		}
	}

	workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	tasks.Add(1)
	dirCh <- root
	go func() {
		tasks.Wait()
		close(dirCh)
	}()

	workers.Wait()

	if firstErr != nil {
		return time.Time{}, firstErr
	}

	return latest, nil
}
