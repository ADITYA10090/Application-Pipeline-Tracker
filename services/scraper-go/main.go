package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	workers := flag.Int("workers", 0, "worker pool size (overrides SCRAPE_WORKERS)")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := LoadConfig()
	if *workers > 0 {
		cfg.Workers = *workers
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	store, err := NewStore(ctx, cfg)
	cancel()
	if err != nil {
		log.Error("failed to connect to stores", "err", err)
		os.Exit(1)
	}
	defer store.Close(context.Background())

	queue := NewQueue(cfg)
	if err := queue.Ping(context.Background()); err != nil {
		log.Error("failed to connect to redis", "err", err)
		os.Exit(1)
	}
	defer queue.Close()

	engine := NewEngine(cfg, store, queue, log)
	srv := NewServer(engine, queue, log)

	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: srv.Routes()}

	go func() {
		log.Info("scraper listening", "addr", cfg.HTTPAddr, "workers", cfg.Workers)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = httpServer.Shutdown(shutCtx)
}
