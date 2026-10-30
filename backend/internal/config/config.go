package config

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
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
