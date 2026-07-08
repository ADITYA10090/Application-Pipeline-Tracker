package main

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics is the snapshot exposed on /metrics. Counters are atomic so the HTTP
// handler can read them while workers mutate them.
type Metrics struct {
	Fetched   atomic.Int64 // postings returned by sources
	Inserted  atomic.Int64 // new rows written to Postgres
	Duplicates atomic.Int64 // dropped by dedup or unique constraint
	Errors    atomic.Int64
	LastRunAt atomic.Int64 // unix seconds
}

// Engine owns the worker pool and the scrape lifecycle.
type Engine struct {
	cfg     Config
	store   *Store
	queue   *Queue
	log     *slog.Logger
	metrics *Metrics

	running atomic.Bool
}

func NewEngine(cfg Config, store *Store, queue *Queue, log *slog.Logger) *Engine {
	return &Engine{cfg: cfg, store: store, queue: queue, log: log, metrics: &Metrics{}}
}

// Run executes one full scrape: fetch every configured source, push new jobs
// onto the Redis queue, then drain the queue with a pool of `Workers`
// goroutines that persist each job. It is safe to call concurrently — a second
// call while one is in flight returns immediately.
func (e *Engine) Run(ctx context.Context) {
	if !e.running.CompareAndSwap(false, true) {
		e.log.Info("scrape already running, skipping trigger")
		return
	}
	defer e.running.Store(false)

	start := time.Now()
	e.metrics.LastRunAt.Store(start.Unix())
	e.log.Info("scrape run started", "sources", len(e.cfg.Sources), "workers", e.cfg.Workers)

	// --- Producer stage: fetch sources, dedup, enqueue.
	for _, src := range e.cfg.Sources {
		jobs, err := FetchSource(ctx, src)
		if err != nil {
			e.metrics.Errors.Add(1)
			e.log.Error("source fetch failed", "kind", src.Kind, "slug", src.Slug, "err", err)
			continue
		}
		e.log.Info("source fetched", "kind", src.Kind, "slug", src.Slug, "jobs", len(jobs))
		for _, j := range jobs {
			e.metrics.Fetched.Add(1)
			seen, err := e.queue.MarkSeen(ctx, j.URL)
			if err != nil {
				e.metrics.Errors.Add(1)
				e.log.Error("dedup check failed", "url", j.URL, "err", err)
				continue
			}
			if !seen {
				e.metrics.Duplicates.Add(1)
				continue
			}
			if err := e.queue.Enqueue(ctx, j); err != nil {
				e.metrics.Errors.Add(1)
				e.log.Error("enqueue failed", "url", j.URL, "err", err)
			}
		}
	}

	// --- Consumer stage: worker pool over channels drains the Redis queue.
	e.drain(ctx)
	e.log.Info("scrape run finished",
		"duration_ms", time.Since(start).Milliseconds(),
		"fetched", e.metrics.Fetched.Load(),
		"inserted", e.metrics.Inserted.Load(),
		"duplicates", e.metrics.Duplicates.Load(),
		"errors", e.metrics.Errors.Load())
}

// drain spins up the worker pool. A single dispatcher goroutine BLPOPs from
// Redis and fans jobs out over a buffered channel; N workers persist them.
func (e *Engine) drain(ctx context.Context) {
	jobCh := make(chan Job, e.cfg.Workers)
	var wg sync.WaitGroup

	for i := 0; i < e.cfg.Workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range jobCh {
				inserted, err := e.store.PersistJob(ctx, j)
				if err != nil {
					e.metrics.Errors.Add(1)
					e.log.Error("persist failed", "worker", id, "url", j.URL, "err", err)
					continue
				}
				if inserted {
					e.metrics.Inserted.Add(1)
				} else {
					e.metrics.Duplicates.Add(1)
				}
			}
		}(i)
	}

	// Dispatcher: pull until the queue is empty, then close the channel.
	for {
		j, err := e.queue.Dequeue(ctx, 1*time.Second)
		if err != nil {
			e.metrics.Errors.Add(1)
			e.log.Error("dequeue failed", "err", err)
			break
		}
		if j == nil { // queue drained
			break
		}
		jobCh <- *j
	}
	close(jobCh)
	wg.Wait()
}
