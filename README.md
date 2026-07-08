# TrackFolio — Job Application Pipeline Platform

A self-hosted platform that ingests job postings, scores them against your
resume, tracks applications through a pipeline, and visualizes progress on a
dashboard. Built as **four independently deployable microservices** to
demonstrate Go, Python, Node.js, React, PostgreSQL, MongoDB, Redis, Docker, and
Kubernetes in one coherent system.

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
             ┌──────┴──────┐
             │  Go Scraper │
             │   Service   │
             └──┬───────┬──┘
         ┌──────▼──┐ ┌──▼─────┐
         │  Redis  │ │ MongoDB│
         └─────────┘ └────────┘
```

## Services

| Service | Path | Tech | Responsibility |
|---------|------|------|----------------|
| Scraper | `services/scraper-go` | Go | Fetch postings from job-board JSON APIs, dedup, persist to Postgres + Mongo via a worker pool over a Redis queue |
| Match | `services/match-service-py` | Python / FastAPI | Score a resume against a JD (TF-IDF cosine similarity) + keyword gap analysis |
| Gateway | `services/api-gateway-node` | Node / Express / TS | Own the application/status data model; proxy to match + scraper; dashboard stats |
| Frontend | `services/frontend-react` | React / Vite / TS | Kanban board, detail modal, stats charts |

**Why these choices** are explained in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md);
the full endpoint reference is in [`docs/API.md`](docs/API.md).

## Quick start (docker-compose)

```bash
docker compose -f infra/docker-compose.yml up --build
# frontend:      http://localhost:8088
# gateway:       http://localhost:8080/health
# scraper:       http://localhost:8081/metrics
# match service: http://localhost:8000/health
```

The Postgres schema (`db/schema.sql`) is applied automatically on first boot.
Trigger a scrape from the UI's "Trigger Scrape" button, or:

```bash
curl -X POST http://localhost:8080/api/jobs/scrape-trigger
curl http://localhost:8081/metrics
```

Configure which boards to scrape via `SCRAPE_SOURCES` (see the scraper service
in `infra/docker-compose.yml`), formatted `kind|slug` comma-separated where
`kind` is `greenhouse`, `lever`, or `remoteok`.

## Kubernetes (minikube)

```bash
minikube start --cpus=4 --memory=6g
minikube addons enable metrics-server ingress

# Build images into minikube's docker daemon:
eval $(minikube docker-env)
docker build -t trackfolio/scraper:latest       services/scraper-go
docker build -t trackfolio/match-service:latest services/match-service-py
docker build -t trackfolio/api-gateway:latest   services/api-gateway-node
docker build -t trackfolio/frontend:latest      services/frontend-react

# Secrets + apply:
cp infra/k8s/secrets.yaml.example infra/k8s/secrets.yaml   # edit the password
kubectl apply -f infra/k8s/

# Reach the app:
minikube tunnel          # in a separate terminal, for the ingress
kubectl -n trackfolio get ingress
# ...or port-forward directly:
kubectl -n trackfolio port-forward svc/frontend 8088:80
```

The Go scraper has a `HorizontalPodAutoscaler` (CPU 60%, 1–5 replicas). Drive
load with `infra/loadtest.sh` — see [`docs/LOAD_TEST_RESULTS.md`](docs/LOAD_TEST_RESULTS.md).

## Data model

PostgreSQL: `companies`, `job_postings`, `applications`, `status_events`,
`resume_versions` (see `db/schema.sql`). MongoDB: one `job_descriptions`
collection of raw JD text/HTML. Redis: `dedup:job:<url_hash>` (TTL'd) and
`queue:scrape_jobs` (durable list).

## Verification status

What was exercised for real while building this (natively, without containers):

- **Go scraper** — `go vet` + unit tests pass; the Redis dedup + durable-queue
  path (`SETNX` dedup, TTL expiry, FIFO enqueue/dequeue, depth) verified with an
  integration test against a **real `redis-server`**.
- **Python match service** — 7 pytest cases pass; `POST /score` and `/health`
  verified live via the FastAPI test client.
- **Node gateway** — built and run against a **real PostgreSQL 16**; the full
  API was curled end-to-end: create/list/detail applications, `PATCH` status
  (with `status_events` history), company listing, dashboard aggregation,
  filters, zod `400`s, and `404`s.
- **Gateway → match-service proxy** — `POST /api/match-score` returned a real
  TF-IDF score and keyword gaps end-to-end across both services.
- **Frontend** — `tsc && vite build` succeeds.

## Known limitations / not implemented

This section is deliberate — it's what makes the project honest and defensible.

- **IMAP email status parsing** — **not implemented.** It was a stretch goal;
  the core CRUD/dashboard was prioritized instead. No stub is shipped pretending
  otherwise.
- **Authentication** — **none.** MVP is single-user, local/personal use, so auth
  was cut as an explicit scope decision rather than half-built.
- **Live scrape against real job boards was not run in the build environment.**
  The environment's egress policy returned `403` for `boards-api.greenhouse.io`,
  `api.lever.co`, and `remoteok.com`. The source adapters and JSON parsing are
  unit-tested; the fetch code is standard `net/http` + `encoding/json`. Run it in
  an environment with open egress to see live ingestion.
- **`docker compose up` / minikube were not run in the build environment.** The
  egress policy also `403`s Docker Hub image layers, so container images could
  not be pulled or built here. The Dockerfiles, compose file, and k8s manifests
  are complete and YAML-validated, but the full containerized end-to-end and the
  HPA load test have not been executed — see `docs/LOAD_TEST_RESULTS.md`, which
  contains **no fabricated performance numbers**.
- **Mongo persistence path** was not run natively (no local `mongod`); it is
  exercised only via the compose/k8s path above. The Postgres persistence path
  and schema were verified natively.
- **Match scoring is TF-IDF**, not embeddings. Fast, deterministic, and easy to
  reason about; the trade-off (surface-token matching vs semantic matching) is
  documented in `services/match-service-py/app/scoring.py`.

## Repository layout

```
services/{scraper-go,match-service-py,api-gateway-node,frontend-react}
infra/{docker-compose.yml,loadtest.sh,k8s/*}
db/schema.sql
docs/{ARCHITECTURE.md,API.md,LOAD_TEST_RESULTS.md}
```
