// Package queue implements the Redis-backed dedup cache and work queue used by
// the scraper pipeline.
//
// Dedup: each processed job URL is recorded as a TTL'd key. On the next run a
// SET NX both tests "have we seen this URL recently?" and records it in one
// round trip, so re-runs skip the Postgres/Mongo writes for jobs already seen
// within the TTL window. Postgres UNIQUE(url) is still the durable source of
// truth; Redis is the fast front cache that keeps re-runs cheap.
//
// Queue: newly discovered job URLs are also RPUSH'd onto a Redis list so other
// consumers (e.g. the match service, future notifiers) can pull fresh postings.
package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// QueueKey is the Redis list of freshly discovered job URLs.
	QueueKey = "queue:scrape_jobs"
	// dedupPrefix namespaces the per-URL dedup keys: dedup:job:<sha256(url)>.
	dedupPrefix = "dedup:job:"
)

type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedis(addr string, ttlSeconds int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Redis{client: client, ttl: time.Duration(ttlSeconds) * time.Second}, nil
}

func (r *Redis) Close() error { return r.client.Close() }

func dedupKey(url string) string {
	sum := sha256.Sum256([]byte(url))
	return dedupPrefix + hex.EncodeToString(sum[:])
}

// Seen atomically records the URL and reports whether it was already present
// within the TTL window. Implemented with SET NX so the check-and-set is a
// single round trip and race-free across concurrent workers.
func (r *Redis) Seen(ctx context.Context, url string) (bool, error) {
	ok, err := r.client.SetNX(ctx, dedupKey(url), 1, r.ttl).Result()
	if err != nil {
		return false, err
	}
	// ok == true  => key was newly set => NOT seen before.
	// ok == false => key already existed => seen.
	return !ok, nil
}

// Enqueue pushes a newly discovered job URL onto the work queue.
func (r *Redis) Enqueue(ctx context.Context, url string) error {
	return r.client.RPush(ctx, QueueKey, url).Err()
}

// Depth returns the current length of the work queue (for /metrics).
func (r *Redis) Depth(ctx context.Context) (int64, error) {
	return r.client.LLen(ctx, QueueKey).Result()
}
