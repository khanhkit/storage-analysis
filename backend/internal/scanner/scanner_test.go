package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"storageanalysis/internal/cache"
)

func TestScannerAggregatesSizes(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, filepath.Join(tempDir, "a.txt"), 1024)

	subDir := filepath.Join(tempDir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(subDir, "b.txt"), 2048)

	store := cache.New(cache.Options{MaxEntries: 10})
	scanner := New(store, 2)

	result, err := scanner.Scan(context.Background(), tempDir, ScanOptions{MaxDepth: 2, UseCache: true})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	expected := int64(3072)
	if result.Node.SizeBytes != expected {
		t.Fatalf("expected %d bytes, got %d", expected, result.Node.SizeBytes)
	}

	if result.Node.ChildCount != 2 {
		t.Fatalf("expected 2 children, got %d", result.Node.ChildCount)
	}
}

func TestScannerCacheHit(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, filepath.Join(tempDir, "a.txt"), 512)

	store := cache.New(cache.Options{MaxEntries: 10})
	scanner := New(store, 2)

	first, err := scanner.Scan(context.Background(), tempDir, ScanOptions{MaxDepth: 1, UseCache: true})
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if first.FromCache {
		t.Fatalf("first scan should not be from cache")
	}

	second, err := scanner.Scan(context.Background(), tempDir, ScanOptions{MaxDepth: 1, UseCache: true})
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if !second.FromCache {
		t.Fatalf("expected cache hit on second scan")
	}
}

func TestScannerStatsCounts(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(t, filepath.Join(tempDir, "a.txt"), 50)
	writeFile(t, filepath.Join(tempDir, "b.txt"), 60)

	store := cache.New(cache.Options{MaxEntries: 10})
	scanner := New(store, 2)

	_, err := scanner.Scan(context.Background(), tempDir, ScanOptions{MaxDepth: 1, UseCache: false, ResetStats: true})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	stats := scanner.Stats()
	if stats.FilesScanned != 2 {
		t.Fatalf("expected 2 files scanned, got %d", stats.FilesScanned)
	}
	if stats.DirsScanned <= 0 {
		t.Fatalf("expected at least 1 dir scanned")
	}
}

func TestScannerAvoidsDeadlockWithSingleWorker(t *testing.T) {
	tempDir := t.TempDir()

	firstDir := filepath.Join(tempDir, "first")
	if err := os.Mkdir(firstDir, 0o755); err != nil {
		t.Fatalf("mkdir first: %v", err)
	}
	writeFile(t, filepath.Join(firstDir, "a.txt"), 10)

	secondDir := filepath.Join(tempDir, "second")
	if err := os.Mkdir(secondDir, 0o755); err != nil {
		t.Fatalf("mkdir second: %v", err)
	}
	writeFile(t, filepath.Join(secondDir, "b.txt"), 20)

	store := cache.New(cache.Options{MaxEntries: 10})
	scanner := New(store, 1)

	done := make(chan error, 1)
	go func() {
		_, err := scanner.Scan(context.Background(), tempDir, ScanOptions{MaxDepth: 2, UseCache: false})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("scan: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("scan appears to be stuck")
	}
}

func writeFile(t *testing.T, path string, size int) {
	payload := make([]byte, size)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
