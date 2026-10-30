package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"storageanalysis/internal/security"
)

type Config struct {
	Address         string
	PublicDir       string
	CacheFile       string
	DefaultDepth    int
	ScanWorkers     int
	CacheMaxEntries int
	CacheTTL        time.Duration
	CacheStrategy   string
	AllowedOrigins  []string
	RobocopyThreads int
	OpsTimeout      time.Duration
	OpsAuthToken    string
	OpsAllowedRoots []string
	AllowRemoteBind bool
	TLSCertFile     string
	TLSKeyFile      string
}

func Load() Config {
	port := getEnv("PORT", "8080")
	address := getEnv("ADDRESS", "127.0.0.1:"+port)

	return Config{
		Address:         address,
		PublicDir:       getEnv("PUBLIC_DIR", "./public"),
		CacheFile:       getEnv("CACHE_FILE", "./data/cache.json"),
		DefaultDepth:    getEnvInt("DEFAULT_DEPTH", 2),
		ScanWorkers:     getEnvInt("SCAN_WORKERS", defaultScanWorkers()),
		CacheMaxEntries: getEnvInt("CACHE_MAX_ENTRIES", 2000),
		CacheTTL:        getEnvDuration("CACHE_TTL_MIN", 0, time.Minute),
		CacheStrategy:   getEnv("CACHE_STRATEGY", "deep"),
		AllowedOrigins:  splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		RobocopyThreads: getEnvInt("ROBOCOPY_THREADS", 32),
		OpsTimeout:      getEnvDuration("OPS_TIMEOUT_MIN", 30, time.Minute),
		OpsAuthToken:    getEnv("OPS_AUTH_TOKEN", ""),
		OpsAllowedRoots: splitOptionalCSV(getEnv("OPS_ALLOWED_ROOTS", "")),
		AllowRemoteBind: strings.EqualFold(getEnv("ALLOW_REMOTE_BIND", "false"), "true"),
		TLSCertFile:     getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:      getEnv("TLS_KEY_FILE", ""),
	}
}

func defaultScanWorkers() int {
	return capDefaultScanWorkers(128)
}

func capDefaultScanWorkers(workers int) int {
	if workers > 128 {
		return 128
	}
	return workers
}

func EffectiveGOMAXPROCS(scanWorkers int) int {
	if scanWorkers <= 0 {
		return runtime.NumCPU()
	}
	return scanWorkers
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback int, unit time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(fallback) * unit
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return time.Duration(fallback) * unit
	}
	return time.Duration(parsed) * unit
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:5173"}
	}
	return out
}

func splitOptionalCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// Validate rejects all unsafe network bind configurations before a listener
// opens. Remote use requires an explicit opt-in, TLS and strong credentials.
func (cfg Config) Validate() error {
	host, _, err := net.SplitHostPort(cfg.Address)
	if err != nil {
		return fmt.Errorf("invalid ADDRESS: %w", err)
	}
	loopback := strings.EqualFold(host, "localhost")
	if ip := net.ParseIP(host); ip != nil {
		loopback = ip.IsLoopback()
	}
	if cfg.OpsAuthToken != "" && len(cfg.OpsAuthToken) < 32 {
		return errors.New("OPS_AUTH_TOKEN must be at least 32 characters")
	}
	if _, err := security.NewPaths(cfg.OpsAllowedRoots); err != nil {
		return err
	}
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return errors.New("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
	}
	if !loopback {
		if !cfg.AllowRemoteBind {
			return errors.New("non-loopback ADDRESS requires ALLOW_REMOTE_BIND=true and secure mode")
		}
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" || len(cfg.OpsAuthToken) < 32 || len(cfg.OpsAllowedRoots) == 0 {
			return errors.New("remote bind requires TLS_CERT_FILE, TLS_KEY_FILE, OPS_AUTH_TOKEN (32+ chars), and OPS_ALLOWED_ROOTS")
		}
	}
	return nil
}
