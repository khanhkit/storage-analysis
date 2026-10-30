package duplicates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Group struct {
	Hash      string   `json:"hash"`
	SizeBytes int64    `json:"sizeBytes"`
	Paths     []string `json:"paths"`
}

type Result struct {
	Groups         []Group   `json:"groups"`
	FileCount      int       `json:"fileCount"`
	DuplicateFiles int       `json:"duplicateFiles"`
	ScannedAt      time.Time `json:"scannedAt"`
	Warnings       []string  `json:"warnings,omitempty"`
}

type Scanner struct {
	sem          chan struct{}
	filesScanned atomic.Int64
	dirsScanned  atomic.Int64
	hashedFiles  atomic.Int64
	scanning     atomic.Bool
	startedAt    atomic.Value
}

type Stats struct {
	Scanning     bool
	StartedAt    time.Time
	FilesScanned int64
	DirsScanned  int64
	HashedFiles  int64
}

func New(workers int) *Scanner {
	if workers <= 0 {
		workers = 4
	}

	scanner := &Scanner{
		sem: make(chan struct{}, workers),
	}
	scanner.startedAt.Store(time.Time{})
	return scanner
}

func (s *Scanner) Stats() Stats {
	startedAt, _ := s.startedAt.Load().(time.Time)
	return Stats{
		Scanning:     s.scanning.Load(),
		StartedAt:    startedAt,
		FilesScanned: s.filesScanned.Load(),
		DirsScanned:  s.dirsScanned.Load(),
		HashedFiles:  s.hashedFiles.Load(),
	}
}

func (s *Scanner) beginScan() {
	s.filesScanned.Store(0)
	s.dirsScanned.Store(0)
	s.hashedFiles.Store(0)
	s.startedAt.Store(time.Now())
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

func (s *Scanner) Scan(ctx context.Context, root string, minSize int64) (Result, error) {
	if root == "" {
		return Result{}, fmt.Errorf("path is required")
	}
	if minSize < 0 {
		minSize = 0
	}

	s.beginScan()
	defer s.endScan()

	sizeBuckets := make(map[int64][]string)
	warnings := make([]string, 0)
	fileCount := 0

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			warnings = append(warnings, err.Error())
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			warnings = append(warnings, infoErr.Error())
			return nil
		}
		if isReparsePoint(info) {
			return nil
		}
		if info.IsDir() {
			s.dirsScanned.Add(1)
			return nil
		}

		s.filesScanned.Add(1)
		if info.Size() < minSize {
			return nil
		}

		fileCount++
		sizeBuckets[info.Size()] = append(sizeBuckets[info.Size()], path)
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	hashBuckets := make(map[int64]map[string][]string)
	var hashMu sync.Mutex
	var wg sync.WaitGroup

	processHash := func(size int64, path string) {
		hash, hashErr := hashFile(path)
		if hashErr != nil {
			hashMu.Lock()
			warnings = append(warnings, hashErr.Error())
			hashMu.Unlock()
			return
		}

		s.hashedFiles.Add(1)
		hashMu.Lock()
		if _, ok := hashBuckets[size]; !ok {
			hashBuckets[size] = make(map[string][]string)
		}
		hashBuckets[size][hash] = append(hashBuckets[size][hash], path)
		hashMu.Unlock()
	}

	for size, paths := range sizeBuckets {
		if len(paths) < 2 {
			continue
		}
		for _, path := range paths {
			if ctx.Err() != nil {
				return Result{}, ctx.Err()
			}
			if s.tryAcquire() {
				wg.Add(1)
				go func(filePath string, fileSize int64) {
					defer wg.Done()
					defer s.release()
					processHash(fileSize, filePath)
				}(path, size)
				continue
			}
			processHash(size, path)
		}
	}

	wg.Wait()

	groups := make([]Group, 0)
	duplicateFiles := 0

	for size, bucket := range hashBuckets {
		for hash, list := range bucket {
			if len(list) < 2 {
				continue
			}
			sort.Strings(list)
			duplicateFiles += len(list)
			groups = append(groups, Group{
				Hash:      hash,
				SizeBytes: size,
				Paths:     list,
			})
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].SizeBytes == groups[j].SizeBytes {
			return groups[i].Hash < groups[j].Hash
		}
		return groups[i].SizeBytes > groups[j].SizeBytes
	})

	return Result{
		Groups:         groups,
		FileCount:      fileCount,
		DuplicateFiles: duplicateFiles,
		ScannedAt:      time.Now(),
		Warnings:       warnings,
	}, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
