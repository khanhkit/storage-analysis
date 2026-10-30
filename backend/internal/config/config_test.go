package config

import "testing"

func TestLoadAddressDefaultsToLoopback(t *testing.T) {
	t.Setenv("ADDRESS", "")
	t.Setenv("PORT", "8080")

	cfg := Load()
	if cfg.Address != "127.0.0.1:8080" {
		t.Fatalf("expected loopback address 127.0.0.1:8080, got %q", cfg.Address)
	}
}

func TestLoadDefaultScanWorkers(t *testing.T) {
	t.Setenv("SCAN_WORKERS", "")

	cfg := Load()
	expected := defaultScanWorkers()
	if cfg.ScanWorkers != expected {
		t.Fatalf("expected ScanWorkers %d, got %d", expected, cfg.ScanWorkers)
	}
}

func TestLoadScanWorkersOverride(t *testing.T) {
	t.Setenv("SCAN_WORKERS", "12")

	cfg := Load()
	if cfg.ScanWorkers != 12 {
		t.Fatalf("expected ScanWorkers 12, got %d", cfg.ScanWorkers)
	}
}

func TestCapDefaultScanWorkers(t *testing.T) {
	if capDefaultScanWorkers(1024) != 128 {
		t.Fatalf("expected cap 128")
	}

	if capDefaultScanWorkers(64) != 64 {
		t.Fatalf("expected cap to keep 64")
	}
}

func TestDefaultScanWorkersIs128(t *testing.T) {
	if defaultScanWorkers() != 128 {
		t.Fatalf("expected default scan workers 128, got %d", defaultScanWorkers())
	}
}

func TestEffectiveGOMAXPROCS(t *testing.T) {
	if EffectiveGOMAXPROCS(12) != 12 {
		t.Fatalf("expected gomaxprocs 12")
	}

	if EffectiveGOMAXPROCS(0) <= 0 {
		t.Fatalf("expected gomaxprocs to be positive")
	}
}
