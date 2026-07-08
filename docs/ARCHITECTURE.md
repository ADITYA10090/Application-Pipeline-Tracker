# Architecture

TrackFolio is four independently deployable services over three datastores. Each
service owns a distinct job and communicates over HTTP; the databases are each
chosen for the shape of data they hold.

```
                         ┌─────────────┐
                         │   React     │
                         │  Dashboard  │
                         └──────┬──────┘
                                │ HTTP (/, /api via ingress or nginx proxy)
                         ┌──────▼──────┐
                         │   Node.js   │
                         │ API Gateway │
                         └──┬───────┬──┘
                     proxy  │       │  proxy
              ┌─────────────┘       └────────────┐
        ┌─────▼──────┐                    ┌───────▼────────┐
        │ PostgreSQL │                    │ Python FastAPI  │
        │ structured │                    │ Match Scoring   │
        │    data    │                    │  (TF-IDF/cosine)│
        └─────▲──────┘                    └─────────────────┘
              │ structured writes
        ┌─────┴───────┐
        │  Go Scraper │──── HTTPS ──▶ Greenhouse / Lever public JSON APIs
        │   Service   │
        └──┬───────┬──┘
           │       │
     ┌─────▼──┐ ┌──▼─────┐
     │  Redis │ │ MongoDB│
     │ dedup  │ │ raw JD │
     │ +queue │ │  text  │
     └────────┘ └────────┘
```

## Why these datastore choices

- **PostgreSQL — structured, relational, queried.** Companies, postings,
  applications, and the `status_events` audit trail are relational with foreign
  keys and are queried with filters/aggregations (dashboard stats). A relational
  DB with constraints (`UNIQUE(url)`, status `CHECK`s) is the right fit and also
  gives us idempotent scraping for free via the unique URL constraint.
- **MongoDB — raw, schemaless documents.** Raw JD HTML/text varies wildly in
  shape and size and is only ever fetched whole by document. A document store
  avoids bloating relational rows with large blobs.
- **Redis — ephemeral, fast, TTL'd.** The dedup cache (`SET NX` with TTL) and
  the work queue (`RPUSH`/`LLEN`) are hot-path, disposable state. Redis's
  atomic ops make the check-and-set race-free across concurrent workers.

## Why these languages

| Service | Language | Why |
|---|---|---|
| Scraper | **Go** | Concurrency: a goroutine worker pool + channels expresses bounded fan-out scraping directly, and a static binary ships on `scratch`. |
| Match | **Python** | The ML/NLP ecosystem (scikit-learn TF-IDF + cosine) is native and idiomatic here. |
| Gateway | **Node.js/TS** | I/O-bound API aggregation and proxying; async fits, and TypeScript + zod give end-to-end typing to the React client. |
| Frontend | **React/TS** | Component model suits a Kanban + dashboard; `@dnd-kit` for drag-and-drop, `recharts` for charts. |

## Data flow

1. **Scrape.** `POST /trigger` (directly or via the gateway) runs the Go worker
   pool: each configured board is fetched concurrently, normalized to a common
   `Job`, deduped against Redis, then written to Postgres (structured) and Mongo
   (raw). Postgres `UNIQUE(url)` is the durable dedup; Redis is the fast cache.
2. **Curate.** The frontend lists un-applied postings (`GET /api/jobs`) and adds
   them to the pipeline as `applications` (default status `wishlist`).
3. **Track.** Dragging a card between Kanban columns issues
   `PATCH /api/applications/:id/status`, which updates the row and appends a
   `status_events` record in one transaction.
4. **Score.** The detail modal posts resume + JD text to `POST /api/match-score`,
   which the gateway proxies to the Python service (TF-IDF cosine similarity +
   skill keyword-gap).
5. **Visualize.** `GET /api/dashboard/stats` aggregates counts by status,
   response rate, and average match score for the charts.

## Concurrency model (scraper)

A hand-rolled worker pool (`internal/pipeline`) rather than a job framework:
a scrape run is a bounded fan-out with a natural completion point, so
`goroutines + channel + WaitGroup` model it directly. Fetches fan out per
source; normalized jobs fan into a buffered channel drained by `SCRAPER_WORKERS`
goroutines doing the DB writes (the slow part). Runs are single-flight
(mutex-guarded) so concurrent triggers return `409` instead of overlapping.
See `docs/LOAD_TEST_RESULTS.md` for how throughput scales with worker count.
