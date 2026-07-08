// Package pipeline wires fetchers -> dedup -> stores using a goroutine worker
// pool fed by a channel. Concurrency is bounded by cfg.Workers.
//
// Why a hand-rolled worker pool instead of a queue library (asynq/machinery)?
// A scrape run is a bounded fan-out with a natural completion point ("this run
// is done"), so goroutines + a channel + a WaitGroup express it directly with
// no external broker in the hot path. Redis still backs cross-run dedup and an
// optional durable queue (see internal/queue), but the in-run concurrency does
// not need a job framework.
package pipeline

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"trackfolio/scraper-go/internal/config"
	"trackfolio/scraper-go/internal/model"
	"trackfolio/scraper-go/internal/scraper"
	"trackfolio/scraper-go/internal/store"
)

// Metrics is a snapshot of counters exposed on /metrics.
type Metrics struct {
	Runs         int64 `json:"runs"`
	JobsScraped  int64 `json:"jobs_scraped"`  // jobs seen from sources
	JobsInserted int64 `json:"jobs_inserted"` // new postings written to Postgres
	JobsDeduped  int64 `json:"jobs_deduped"`  // skipped via Redis dedup cache
	Errors       int64 `json:"errors"`
	QueueDepth   int64 `json:"queue_depth"`
	LastRunUnix  int64 `json:"last_run_unix"`
}

// Dedup is the subset of the Redis queue used to avoid re-processing jobs.
// It is an interface so the pipeline runs (via Postgres uniqueness alone) even
// when Redis is unavailable.
type Dedup interface {
	Seen(ctx context.Context, url string) (bool, error)
	Enqueue(ctx context.Context, url string) error
	Depth(ctx context.Context) (int64, error)
}

// RawStore persists raw JD text (MongoDB in production). Interface so the
// pipeline degrades gracefully — and stays unit-testable — without Mongo.
type RawStore interface {
	SaveJD(ctx context.Context, jd store.JobDescription) (string, error)
}

type Pipeline struct {
	cfg    config.Config
	pg     *store.Postgres
	mongo  RawStore // may be nil
	dedup  Dedup    // may be nil
	client *http.Client
	log    *slog.Logger

	runs         atomic.Int64
	jobsScraped  atomic.Int64
	jobsInserted atomic.Int64
	jobsDeduped  atomic.Int64
	errors       atomic.Int64
	lastRun      atomic.Int64

	mu      sync.Mutex
	running bool
}

func New(cfg config.Config, pg *store.Postgres, mongo RawStore, dedup Dedup, log *slog.Logger) *Pipeline {
	return &Pipeline{
		cfg:    cfg,
		pg:     pg,
		mongo:  mongo,
		dedup:  dedup,
		client: &http.Client{Timeout: 20 * time.Second},
		log:    log,
	}
}

// TriggerAsync starts a scrape run in the background if one is not already
// running, returning whether it was accepted.
func (p *Pipeline) TriggerAsync(ctx context.Context) bool {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return false
	}
	p.running = true
	p.mu.Unlock()

	go func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		p.RunOnce(runCtx)
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()
	return true
}

// RunOnce fetches every configured source and processes the results through a
// worker pool. It blocks until the run completes.
func (p *Pipeline) RunOnce(ctx context.Context) {
	start := time.Now()
	p.runs.Add(1)
	p.log.Info("scrape run started", "sources", len(p.cfg.Sources), "workers", p.cfg.Workers)

	jobsCh := make(chan model.Job, 256)

	// Worker pool: DB/Mongo writes are the slow part, so fan them out.
	var workers sync.WaitGroup
	n := p.cfg.Workers
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobsCh {
				p.process(ctx, job)
			}
		}()
	}

	// Fetch each source concurrently, feeding normalized jobs into the pool.
	var fetchers sync.WaitGroup
	for _, src := range p.cfg.Sources {
		fetchers.Add(1)
		go func(src config.Source) {
			defer fetchers.Done()
			f, err := scraper.New(p.client, p.cfg, src)
			if err != nil {
				p.errors.Add(1)
				p.log.Error("fetcher init failed", "source", src.Kind+":"+src.Board, "err", err)
				return
			}
			jobs, err := f.Fetch(ctx)
			if err != nil {
				p.errors.Add(1)
				p.log.Error("fetch failed", "source", src.Kind+":"+src.Board, "err", err)
				return
			}
			p.log.Info("fetched", "source", src.Kind+":"+src.Board, "jobs", len(jobs))
			for _, j := range jobs {
				select {
				case jobsCh <- j:
				case <-ctx.Done():
					return
				}
			}
		}(src)
	}

	fetchers.Wait()
	close(jobsCh)
	workers.Wait()

	p.lastRun.Store(time.Now().Unix())
	p.log.Info("scrape run finished",
		"duration_ms", time.Since(start).Milliseconds(),
		"scraped", p.jobsScraped.Load(),
		"inserted", p.jobsInserted.Load(),
		"deduped", p.jobsDeduped.Load(),
		"errors", p.errors.Load())
}

// process handles a single job: dedup check, Mongo raw write, Postgres upsert.
func (p *Pipeline) process(ctx context.Context, j model.Job) {
	p.jobsScraped.Add(1)
	if j.URL == "" || j.Title == "" {
		return
	}

	// Fast-path dedup: if Redis has seen this URL recently, skip the DB round
	// trips entirely. Postgres UNIQUE(url) remains the durable source of truth.
	if p.dedup != nil {
		seen, err := p.dedup.Seen(ctx, j.URL)
		if err != nil {
			p.log.Warn("dedup check failed, falling through to db", "err", err)
		} else if seen {
			p.jobsDeduped.Add(1)
			return
		}
	}

	var mongoRef string
	if p.mongo != nil {
		ref, err := p.mongo.SaveJD(ctx, store.JobDescription{
			JobURL:    j.URL,
			Company:   j.Company,
			Title:     j.Title,
			RawHTML:   j.RawHTML,
			RawText:   j.RawText,
			ScrapedAt: time.Now().UTC(),
		})
		if err != nil {
			p.errors.Add(1)
			p.log.Error("mongo write failed", "url", j.URL, "err", err)
			return
		}
		mongoRef = ref
	}

	companyID, err := p.pg.UpsertCompany(ctx, j.Company)
	if err != nil {
		p.errors.Add(1)
		p.log.Error("company upsert failed", "company", j.Company, "err", err)
		return
	}

	_, inserted, err := p.pg.UpsertJob(ctx, j, companyID, mongoRef)
	if err != nil {
		p.errors.Add(1)
		p.log.Error("job upsert failed", "url", j.URL, "err", err)
		return
	}
	if inserted {
		p.jobsInserted.Add(1)
	}

	if p.dedup != nil {
		if err := p.dedup.Enqueue(ctx, j.URL); err != nil {
			p.log.Warn("enqueue failed", "err", err)
		}
	}
}

// Snapshot returns the current metrics for the /metrics endpoint.
func (p *Pipeline) Snapshot(ctx context.Context) Metrics {
	var depth int64
	if p.dedup != nil {
		if d, err := p.dedup.Depth(ctx); err == nil {
			depth = d
		}
	}
	return Metrics{
		Runs:         p.runs.Load(),
		JobsScraped:  p.jobsScraped.Load(),
		JobsInserted: p.jobsInserted.Load(),
		JobsDeduped:  p.jobsDeduped.Load(),
		Errors:       p.errors.Load(),
		QueueDepth:   depth,
		LastRunUnix:  p.lastRun.Load(),
	}
}
