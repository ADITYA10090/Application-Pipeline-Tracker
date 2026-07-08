package main

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration, sourced exclusively from environment
// variables so the same binary runs unchanged in docker-compose and k8s.
type Config struct {
	HTTPAddr    string
	Workers     int
	PostgresURL string
	MongoURI    string
	MongoDB     string
	RedisAddr   string
	RedisPass   string
	DedupTTL    int // seconds
	// Sources describe the job-board endpoints to scrape. Each is "kind|slug"
	// where kind is greenhouse|lever|remoteok. Configured via SCRAPE_SOURCES,
	// comma separated, e.g. "greenhouse|stripe,lever|netflix,remoteok|".
	Sources []Source
}

type Source struct {
	Kind string
	Slug string // company board token (greenhouse/lever); empty for remoteok
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// LoadConfig reads configuration from the environment with sensible defaults
// that match the docker-compose service names.
func LoadConfig() Config {
	c := Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8081"),
		Workers:     getenvInt("SCRAPE_WORKERS", 10),
		PostgresURL: getenv("POSTGRES_URL", "postgres://trackfolio:trackfolio@postgres:5432/trackfolio?sslmode=disable"),
		MongoURI:    getenv("MONGO_URI", "mongodb://mongo:27017"),
		MongoDB:     getenv("MONGO_DB", "trackfolio"),
		RedisAddr:   getenv("REDIS_ADDR", "redis:6379"),
		RedisPass:   getenv("REDIS_PASSWORD", ""),
		DedupTTL:    getenvInt("DEDUP_TTL_SECONDS", 86400),
	}
	c.Sources = parseSources(getenv("SCRAPE_SOURCES", "greenhouse|stripe,greenhouse|gitlab,lever|netflix,remoteok|"))
	return c
}

func parseSources(raw string) []Source {
	var out []Source
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		bits := strings.SplitN(part, "|", 2)
		s := Source{Kind: strings.TrimSpace(bits[0])}
		if len(bits) == 2 {
			s.Slug = strings.TrimSpace(bits[1])
		}
		if s.Kind == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
