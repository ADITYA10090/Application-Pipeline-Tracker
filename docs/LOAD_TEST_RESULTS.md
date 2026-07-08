# Load Test Results

## Status: NOT YET RUN — no fabricated numbers

The scraper HPA load test is designed to run against the Kubernetes
deployment on minikube. It was **not executed in the environment this
project was built in**, for two reasons, both of which are external
constraints rather than problems with the code:

1. **Container image pulls are blocked.** The build/CI environment's egress
   policy returns `403 Forbidden` for Docker Hub image layers
   (`production.cloudfront.docker.com`), so neither `docker compose up` nor a
   minikube cluster could be brought up here.
2. **Target job-board APIs are blocked.** The same egress policy returns `403`
   for `boards-api.greenhouse.io`, `api.lever.co`, and `remoteok.com`, so a
   real scrape run could not be exercised against live sources either.

Per the project's own rule — *"Real numbers only. If load testing isn't run,
don't state a throughput number"* — this file intentionally contains **no**
throughput/latency figures. Run the procedure below on a machine with
unrestricted egress and fill in the table.

## What *was* verified natively (no containers required)

| Component | How it was verified | Result |
|-----------|--------------------|--------|
| Scraper source parsing | Go unit tests (`go test`) | pass |
| Scraper Redis dedup + durable queue | Go integration test vs a real `redis-server` | pass (SETNX dedup, TTL expiry, FIFO queue, depth) |
| Match service scoring | 7 pytest cases + live FastAPI TestClient | pass |
| Gateway CRUD + dashboard | Live `node dist/index.js` vs a real PostgreSQL 16 | pass (create/list/patch/detail/stats/filters/validation/404) |
| Gateway → match-service proxy | Live end-to-end `POST /api/match-score` | pass (returned real score + keyword gaps) |
| Frontend | `tsc && vite build` | pass |

See the README "Verification status" section for the full breakdown.

## How to run the HPA load test (minikube)

Prerequisites: `minikube`, `kubectl`, and the metrics-server addon (the HPA
needs it to read CPU).

```bash
minikube start --cpus=4 --memory=6g
minikube addons enable metrics-server
minikube addons enable ingress

# Build the four images straight into minikube's docker daemon:
eval $(minikube docker-env)
docker build -t trackfolio/scraper:latest       services/scraper-go
docker build -t trackfolio/match-service:latest services/match-service-py
docker build -t trackfolio/api-gateway:latest   services/api-gateway-node
docker build -t trackfolio/frontend:latest      services/frontend-react

# Deploy:
cp infra/k8s/secrets.yaml.example infra/k8s/secrets.yaml   # edit the password
kubectl apply -f infra/k8s/

# In terminal A — watch the autoscaler react:
kubectl -n trackfolio get hpa scraper -w
kubectl -n trackfolio get pods -l app=scraper -w

# In terminal B — expose the scraper and drive load:
kubectl -n trackfolio port-forward svc/scraper 8081:8081 &
CONCURRENCY=50 REQUESTS=2000 TARGET=http://localhost:8081 ./infra/loadtest.sh
```

## Results table (fill in after running)

| Metric | Value |
|--------|-------|
| Requests | _e.g. 2000_ |
| Concurrency | _e.g. 50_ |
| Throughput (req/s) | _TBD_ |
| p50 / p95 latency of `POST /trigger` | _TBD_ |
| Scraper replicas before | 1 |
| Scraper replicas at peak | _TBD (≤5)_ |
| Time to scale up | _TBD_ |
| Time to scale back to 1 | _TBD_ |

> Note: `POST /trigger` returns `202 Accepted` immediately and runs the scrape
> in the background, so trigger-endpoint latency measures enqueue + dispatch
> cost, not the full scrape. CPU load (and therefore HPA scaling) comes from
> the concurrent scrape runs draining the queue and persisting jobs.
