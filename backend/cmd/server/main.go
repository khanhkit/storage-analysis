package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"storageanalysis/internal/api"
	"storageanalysis/internal/cache"
	"storageanalysis/internal/config"
	"storageanalysis/internal/duplicates"
	"storageanalysis/internal/ops"
	"storageanalysis/internal/scanner"
)

func main() {
	cfg := config.Load()

	cacheStore := cache.New(cache.Options{
		MaxEntries: cfg.CacheMaxEntries,
		TTL:        cfg.CacheTTL,
		FilePath:   cfg.CacheFile,
	})
	if err := cacheStore.LoadFromDisk(); err != nil {
		log.Printf("cache load failed: %v", err)
	}

	runtime.GOMAXPROCS(config.EffectiveGOMAXPROCS(cfg.ScanWorkers))

	scannerClient := scanner.New(cacheStore, cfg.ScanWorkers)
	dupScanner := duplicates.New(cfg.ScanWorkers)
	opsClient := ops.NewRobocopy(ops.Options{
		Threads: cfg.RobocopyThreads,
		Timeout: cfg.OpsTimeout,
	})

	server := api.NewServer(cfg, scannerClient, dupScanner, opsClient)
	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: server.Router(),
	}

	go func() {
		log.Printf("storage analysis server listening on %s", cfg.Address)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if err := cacheStore.SaveToDisk(); err != nil {
		log.Printf("cache save failed: %v", err)
	}
}
