package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration, sourced from environment variables.
type Config struct {
	HTTPAddr string

	PostgresURL string
	MongoURI    string
	MongoDB     string
	RedisAddr   string

	// Sources is the list of job boards to scrape, e.g. "greenhouse:stripe".
	Sources []Source

	// Base URLs are overridable so the scraper can be pointed at a local
	// fixture server in tests / CI instead of the live public APIs.
	GreenhouseBase string
	LeverBase      string

	Workers int

	// DedupTTLSeconds controls how long a scraped job URL stays in the Redis
	// dedup set before it may be re-processed.
	DedupTTLSeconds int
}

// Source identifies one job board to scrape.
type Source struct {
	Kind  string // "greenhouse" | "lever"
	Board string // company/board slug
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// lookupOrDefault returns the env value if the variable is set (even to empty),
// otherwise the default. Lets callers explicitly disable a feature via KEY="".
func lookupOrDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
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

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	c := Config{
		HTTPAddr: getenv("SCRAPER_HTTP_ADDR", ":8081"),
		PostgresURL: getenv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/trackfolio?sslmode=disable"),
		// MongoURI: an explicitly-set empty value disables raw-JD storage
		// (LookupEnv distinguishes "set to empty" from "unset").
		MongoURI:        lookupOrDefault("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:         getenv("MONGO_DB", "trackfolio"),
		RedisAddr:       getenv("REDIS_ADDR", "localhost:6379"),
		GreenhouseBase:  getenv("SCRAPER_GREENHOUSE_BASE", "https://boards-api.greenhouse.io"),
		LeverBase:       getenv("SCRAPER_LEVER_BASE", "https://api.lever.co"),
		Workers:         getenvInt("SCRAPER_WORKERS", 10),
		DedupTTLSeconds: getenvInt("SCRAPER_DEDUP_TTL", 86400),
	}
	c.Sources = parseSources(getenv("SCRAPER_SOURCES",
		"greenhouse:stripe,greenhouse:airbnb,lever:netflix"))
	return c
}

// parseSources parses "greenhouse:stripe,lever:netflix" into []Source.
func parseSources(raw string) []Source {
	var out []Source
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		out = append(out, Source{Kind: strings.TrimSpace(kv[0]), Board: strings.TrimSpace(kv[1])})
	}
	return out
}
