package duplicates

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDuplicatesFindsGroups(t *testing.T) {
	tempDir := t.TempDir()

	first := filepath.Join(tempDir, "first.bin")
	second := filepath.Join(tempDir, "second.bin")
	third := filepath.Join(tempDir, "third.bin")

	if err := os.WriteFile(first, []byte("same-content"), 0o644); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := os.WriteFile(second, []byte("same-content"), 0o644); err != nil {
		t.Fatalf("write second: %v", err)
	}
	if err := os.WriteFile(third, []byte("unique"), 0o644); err != nil {
		t.Fatalf("write third: %v", err)
	}

	scanner := New(2)
	result, err := scanner.Scan(context.Background(), tempDir, 1)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(result.Groups))
	}

	if len(result.Groups[0].Paths) != 2 {
		t.Fatalf("expected 2 duplicate files, got %d", len(result.Groups[0].Paths))
	}
}

func TestScanDuplicatesRespectsMinSize(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "small.bin")

	if err := os.WriteFile(filePath, []byte("tiny"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	scanner := New(2)
	result, err := scanner.Scan(context.Background(), tempDir, 1024)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if len(result.Groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(result.Groups))
	}
}

func TestDuplicateStatsCounts(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "alpha.bin")

	if err := os.WriteFile(filePath, []byte("alpha"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	scanner := New(2)
	_, err := scanner.Scan(context.Background(), tempDir, 1)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	stats := scanner.Stats()
	if stats.FilesScanned <= 0 {
		t.Fatalf("expected files scanned to be positive")
	}
	if stats.DirsScanned <= 0 {
		t.Fatalf("expected dirs scanned to be positive")
	}
}
