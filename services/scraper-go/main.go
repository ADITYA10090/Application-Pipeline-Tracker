package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trackfolio/scraper-go/internal/config"
	"trackfolio/scraper-go/internal/pipeline"
	"trackfolio/scraper-go/internal/queue"
	"trackfolio/scraper-go/internal/server"
	"trackfolio/scraper-go/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Postgres is required.
	pgCtx, pgCancel := context.WithTimeout(ctx, 10*time.Second)
	pg, err := store.NewPostgres(pgCtx, cfg.PostgresURL)
	pgCancel()
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	// Mongo is best-effort: if it is unreachable (or MONGO_URI is empty) the
	// scraper still records structured fields in Postgres, it just skips the
	// raw-JD copy.
	var raw pipeline.RawStore
	if cfg.MongoURI == "" {
		log.Warn("MONGO_URI empty, raw JD storage disabled")
	} else {
		mgoCtx, mgoCancel := context.WithTimeout(ctx, 10*time.Second)
		mongo, err := store.NewMongo(mgoCtx, cfg.MongoURI, cfg.MongoDB)
		mgoCancel()
		if err != nil {
			log.Warn("mongo unavailable, raw JD storage disabled", "err", err)
		} else {
			raw = mongo
			defer mongo.Close(context.Background())
		}
	}

	// Redis backs the dedup cache and work queue; best-effort like Mongo so a
	// missing Redis degrades to Postgres-only dedup instead of failing.
	var dedup pipeline.Dedup
	if r, err := queue.NewRedis(cfg.RedisAddr, cfg.DedupTTLSeconds); err != nil {
		log.Warn("redis unavailable, dedup cache + queue disabled", "err", err)
	} else {
		dedup = r
		defer r.Close()
	}

	pl := pipeline.New(cfg, pg, raw, dedup, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.New(pl, log).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("scraper listening", "addr", cfg.HTTPAddr, "sources", len(cfg.Sources))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}
