package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"storageanalysis/internal/model"
)

type Entry struct {
	Path       string     `json:"path"`
	ModTime    time.Time  `json:"modTime"`
	DeepModTime time.Time `json:"deepModTime"`
	Tree       model.Node `json:"tree"`
	ScannedAt  time.Time  `json:"scannedAt"`
	LastAccess time.Time  `json:"lastAccess"`
}

type Options struct {
	MaxEntries int
	TTL        time.Duration
	FilePath   string
}

type Cache struct {
	mu         sync.RWMutex
	entries    map[string]Entry
	maxEntries int
	ttl        time.Duration
	filePath   string
	saveMu     sync.Mutex
	lastSave   time.Time
}

type diskState struct {
	Entries []Entry `json:"entries"`
}

func New(opts Options) *Cache {
	maxEntries := opts.MaxEntries
	if maxEntries <= 0 {
		maxEntries = 2000
	}

	return &Cache{
		entries:    make(map[string]Entry),
		maxEntries: maxEntries,
		ttl:        opts.TTL,
		filePath:   opts.FilePath,
	}
}

func (c *Cache) Get(path string) (Entry, bool) {
	if path == "" {
		return Entry{}, false
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[path]
	if !ok {
		return Entry{}, false
	}

	if c.isExpired(entry, now) {
		delete(c.entries, path)
		return Entry{}, false
	}

	entry.LastAccess = now
	c.entries[path] = entry
	return entry, true
}

func (c *Cache) Set(entry Entry) {
	if entry.Path == "" {
		return
	}

	now := time.Now()
	entry.LastAccess = now
	if entry.ScannedAt.IsZero() {
		entry.ScannedAt = now
	}

	c.mu.Lock()
	c.entries[entry.Path] = entry
	c.purgeExpiredLocked(now)
	c.pruneToMaxLocked()
	c.mu.Unlock()

	c.saveAsync()
}

func (c *Cache) LoadFromDisk() error {
	if c.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state diskState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]Entry, len(state.Entries))
	for _, entry := range state.Entries {
		if entry.Path != "" {
			c.entries[entry.Path] = entry
		}
	}
	return nil
}

func (c *Cache) SaveToDisk() error {
	if c.filePath == "" {
		return nil
	}

	entries := c.snapshot()
	state := diskState{Entries: entries}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.filePath), 0o755); err != nil {
		return err
	}

	tmp := c.filePath + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, c.filePath)
}

func (c *Cache) snapshot() []Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]Entry, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, entry)
	}

	return entries
}

func (c *Cache) isExpired(entry Entry, now time.Time) bool {
	if c.ttl <= 0 {
		return false
	}
	return now.Sub(entry.LastAccess) > c.ttl
}

func (c *Cache) purgeExpiredLocked(now time.Time) {
	if c.ttl <= 0 {
		return
	}

	for key, entry := range c.entries {
		if c.isExpired(entry, now) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) pruneToMaxLocked() {
	if len(c.entries) <= c.maxEntries {
		return
	}

	sorted := make([]Entry, 0, len(c.entries))
	for _, entry := range c.entries {
		sorted = append(sorted, entry)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastAccess.Before(sorted[j].LastAccess)
	})

	removeCount := len(c.entries) - c.maxEntries
	for i := 0; i < removeCount && i < len(sorted); i++ {
		delete(c.entries, sorted[i].Path)
	}
}

func (c *Cache) saveAsync() {
	if c.filePath == "" {
		return
	}

	go func() {
		c.saveMu.Lock()
		defer c.saveMu.Unlock()

		if time.Since(c.lastSave) < 5*time.Second {
			return
		}

		if err := c.SaveToDisk(); err == nil {
			c.lastSave = time.Now()
		}
	}()
}
