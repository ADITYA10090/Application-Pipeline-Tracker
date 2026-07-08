# Architecture

TrackFolio is four independently deployable services around three datastores.
Each service owns a distinct responsibility and a distinct technology, chosen to
fit the job rather than to pad a résumé.

```
                         ┌─────────────┐
                         │   React     │
                         │  Dashboard  │
                         └──────┬──────┘
                                │ HTTP (/api via ingress)
                         ┌──────▼──────┐
                         │   Node.js   │
                         │ API Gateway │
                         └──┬───────┬──┘
                    ┌───────┘       └────────┐
             ┌──────▼──────┐         ┌───────▼────────┐
             │  PostgreSQL │         │ Python FastAPI  │
             │ (structured │         │ Match Scoring   │
             │    data)    │         │    Service      │
             └─────────────┘         └────────────────┘
                    ▲
                    │
             ┌──────┴──────┐
             │  Go Scraper │
             │   Service   │
             └──┬───────┬──┘
                │       │
         ┌──────▼──┐ ┌──▼─────┐
         │  Redis  │ │ MongoDB│
         │ (dedup, │ │ (raw   │
         │  queue) │ │  JD)   │
         └─────────┘ └────────┘
```

## Data flow

1. **Scrape.** The frontend's "Trigger Scrape" button hits the gateway's
   `POST /api/jobs/scrape-trigger`, which proxies to the Go scraper's
   `POST /trigger`. The scraper fetches each configured job-board endpoint,
   deduplicates by URL hash in Redis, pushes new jobs onto a durable Redis
   list, and drains that list with a worker pool that writes structured fields
   to Postgres and raw JD text to Mongo.
2. **Score.** `POST /api/match-score` proxies resume + JD text to the Python
   service, which returns a TF-IDF cosine similarity plus matched/missing skill
   keywords.
3. **Track.** The gateway owns applications and their status history. Moving a
   card on the Kanban board issues `PATCH /api/applications/:id/status`, which
   updates the row and appends a `status_events` audit record.
4. **Visualize.** `GET /api/dashboard/stats` aggregates counts by status,
   response rate, average match score, and applications-over-time for the
   charts.

## Why each technology

| Service | Language | Why |
|---------|----------|-----|
| Scraper | **Go** | Concurrent I/O-bound fan-out over many job boards is exactly what goroutines + channels are for; compiles to a tiny static binary. |
| Match | **Python / FastAPI** | The ML/NLP ecosystem (scikit-learn) lives here; FastAPI gives typed request/response validation for free. |
| Gateway | **Node.js / TypeScript** | Central I/O-bound API glue; async proxying and JSON handling are Node's sweet spot. |
| Frontend | **React / Vite** | Component model fits a Kanban board + charts; Vite gives a fast build. |

| Datastore | Holds | Why |
|-----------|-------|-----|
| **PostgreSQL** | companies, job_postings, applications, status_events | Relational, foreign-keyed pipeline data with aggregate queries. |
| **MongoDB** | raw JD HTML/text | Bulky, schema-loose documents that don't belong in relational rows. |
| **Redis** | dedup cache + work queue | O(1) `SETNX` dedup with TTL and a durable list that survives scraper restarts. |

## Scraper internals (the defensible part)

- **Worker pool.** One dispatcher goroutine `BLPOP`s jobs off the Redis list and
  fans them out over a buffered channel; `N` worker goroutines (configurable via
  `-workers` / `SCRAPE_WORKERS`) each persist a job. Channels give in-process
  back-pressure; Redis gives cross-restart durability. Using both is deliberate,
  not redundant — the channel is the fan-out, the list is the queue of record.
- **Dedup.** `dedup:job:<sha256(url)>` is set with `SETNX` and a TTL. First write
  wins and is processed; repeats within the TTL are dropped before touching
  Postgres. Postgres's `UNIQUE(url)` + `ON CONFLICT DO NOTHING` is the
  second line of defense if the cache is cold.
- **Source fragility.** Sources are structured JSON APIs (Greenhouse, Lever,
  RemoteOK), not scraped HTML, so a company changing its careers-page markup
  does not break ingestion. A source returning an unexpected shape fails that
  one source (logged, error-counted) without aborting the run.

## Deployment

- **docker-compose** brings up all seven containers on one network for local dev.
- **Kubernetes** (minikube) runs the databases as StatefulSets with PVCs, the app
  services as Deployments, an nginx Ingress for routing, and an HPA on the
  scraper (CPU-target, 1–5 replicas) to demonstrate horizontal scaling.
