package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	queueKey     = "queue:scrape_jobs"
	dedupKeyFmt  = "dedup:job:%s"
)

// Queue is the Redis-backed work queue plus the dedup cache.
//
//   - dedup:job:<url_hash>  a TTL'd string used as a "seen recently" marker so
//     the same posting fetched twice within the TTL window is dropped before it
//     ever hits Postgres.
//   - queue:scrape_jobs     a durable list of serialized Jobs awaiting persist.
//     Producers RPUSH, the worker pool BLPOPs. Surviving a scraper restart is
//     the reason this is Redis and not just an in-process channel.
type Queue struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewQueue(cfg Config) *Queue {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
	})
	return &Queue{rdb: rdb, ttl: time.Duration(cfg.DedupTTL) * time.Second}
}

func (q *Queue) Ping(ctx context.Context) error {
	return q.rdb.Ping(ctx).Err()
}

func (q *Queue) Close() error { return q.rdb.Close() }

// MarkSeen atomically records the URL hash and reports whether it was newly set
// (true = first time seen, should be processed; false = duplicate).
func (q *Queue) MarkSeen(ctx context.Context, url string) (bool, error) {
	key := "dedup:job:" + urlHash(url)
	ok, err := q.rdb.SetNX(ctx, key, "1", q.ttl).Result()
	return ok, err
}

func (q *Queue) Enqueue(ctx context.Context, j Job) error {
	b, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return q.rdb.RPush(ctx, queueKey, b).Err()
}

// Dequeue blocks up to timeout for the next job. Returns (nil, nil) on timeout.
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*Job, error) {
	res, err := q.rdb.BLPop(ctx, timeout, queueKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var j Job
	if err := json.Unmarshal([]byte(res[1]), &j); err != nil {
		return nil, err
	}
	return &j, nil
}

func (q *Queue) Depth(ctx context.Context) (int64, error) {
	return q.rdb.LLen(ctx, queueKey).Result()
}
