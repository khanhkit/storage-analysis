package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"storageanalysis/internal/cache"
	"storageanalysis/internal/config"
	"storageanalysis/internal/duplicates"
	"storageanalysis/internal/ops"
	"storageanalysis/internal/scanner"
)

type scanResponse struct {
	Success bool     `json:"success"`
	Data    ScanData `json:"data"`
}

type duplicatesResponse struct {
	Success bool         `json:"success"`
	Data    DuplicateData `json:"data"`
}

func TestHandleScan(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "alpha.txt")
	if err := os.WriteFile(filePath, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg := config.Config{DefaultDepth: 1}
	store := cache.New(cache.Options{MaxEntries: 10})
	scannerClient := scanner.New(store, 2)
	dupScanner := duplicates.New(2)
	opsClient := ops.NewRobocopy(ops.Options{Threads: 1})

	server := NewServer(cfg, scannerClient, dupScanner, opsClient)
	listener := httptest.NewServer(server.Router())
	defer listener.Close()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, listener.URL+"/api/scan?path="+tempDir+"&depth=1", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer response.Body.Close()

	var payload scanResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success")
	}

	if payload.Data.Node.Path != tempDir {
		t.Fatalf("unexpected path")
	}
}

func TestHandleDuplicates(t *testing.T) {
	tempDir := t.TempDir()
	first := filepath.Join(tempDir, "first.bin")
	second := filepath.Join(tempDir, "second.bin")

	if err := os.WriteFile(first, []byte("same"), 0o644); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := os.WriteFile(second, []byte("same"), 0o644); err != nil {
		t.Fatalf("write second: %v", err)
	}

	cfg := config.Config{DefaultDepth: 1}
	store := cache.New(cache.Options{MaxEntries: 10})
	scannerClient := scanner.New(store, 2)
	dupScanner := duplicates.New(2)
	opsClient := ops.NewRobocopy(ops.Options{Threads: 1})

	server := NewServer(cfg, scannerClient, dupScanner, opsClient)
	listener := httptest.NewServer(server.Router())
	defer listener.Close()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, listener.URL+"/api/duplicates?path="+tempDir+"&minSizeBytes=1", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer response.Body.Close()

	var payload duplicatesResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success")
	}

	if len(payload.Data.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(payload.Data.Groups))
	}
	if len(payload.Data.Groups[0].Paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(payload.Data.Groups[0].Paths))
	}
}
