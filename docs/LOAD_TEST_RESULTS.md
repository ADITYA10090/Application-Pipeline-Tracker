# Load Test Results — Go Scraper Service

All numbers below were **actually measured** in the build environment on the
date of writing. Where something could not be measured here (Kubernetes HPA pod
scaling), it is called out explicitly with the commands to reproduce it — no
fabricated figures.

## Environment

| | |
|---|---|
| CPU | 4 cores |
| RAM | 15 GiB |
| Go | 1.24.7 |
| Postgres | 16 (local, unix socket) |
| Redis | 7 (local, TCP) |
| MongoDB | disabled for these runs (`MONGO_URI=""`) to isolate the Postgres write path |
| Tool | [`hey`](https://github.com/rakyll/hey) v0.1.5 |
| Job source | local fixture server returning 500 Greenhouse-format postings |

> MongoDB was disabled so the measurements reflect the fetch → dedup → Postgres
> path without a second datastore's latency mixed in. With Mongo enabled each
> job additionally does one upsert to Mongo.

## 1. HTTP endpoint throughput

`hey -n 5000 -c 50` against the running scraper:

| Endpoint | Requests/sec | Avg latency | p95 | p99 |
|---|---:|---:|---:|---:|
| `GET /health`  | **37,995** | 1.2 ms | — | 5.8 ms |
| `GET /metrics` | **24,088** | 2.0 ms | 4.7 ms | 13.6 ms |

`/metrics` is measurably slower than `/health` because it performs a Redis
`LLEN` (queue depth) on every call, while `/health` is a pure in-process
response. This is the expected cost of the live queue-depth gauge.

## 2. Pipeline throughput vs. worker-pool size

One scrape run of 500 postings, varying `SCRAPER_WORKERS`. Duration is the
scraper's own `scrape run finished` `duration_ms`; all 500 rows were inserted
each run (verified against Postgres).

| Workers | Duration | Jobs/sec |
|---:|---:|---:|
| 1  | 1097 ms | 456 |
| 2  |  800 ms | 625 |
| **5**  | **685 ms** | **730** |
| 10 |  796 ms | 628 |
| 20 |  740 ms | 676 |
| 50 |  775 ms | 645 |

**Interpretation.** Throughput improves ~1.6× going from 1 → 5 workers
(456 → 730 jobs/sec) as concurrent Postgres upserts overlap. Beyond ~5 workers
it **plateaus and slightly regresses**: a single local Postgres instance becomes
the bottleneck, and additional goroutines contend for the connection pool and
add scheduling overhead rather than useful parallelism. The takeaway that would
be defended in an interview: the worker pool is worth it, but the right worker
count is bounded by the downstream datastore, not by "more is better" — here the
sweet spot is ~5 for a single-node Postgres. Scaling further requires scaling
Postgres (connections / read replicas / partitioning), not the scraper.

## 3. Concurrent trigger handling (single-flight)

`hey -n 200 -c 40 -m POST /trigger` while a run is in progress:

```
Status code distribution:
  [202]   1 responses     <- one run accepted
  [409]   199 responses   <- rejected while a run is already in flight
```

The scraper guards runs with a mutex-backed single-flight: exactly one scrape
runs at a time and 199 concurrent duplicate triggers are cleanly rejected with
`409 Conflict` instead of piling up overlapping runs. This is what makes a naive
"spam the trigger button" load safe.

## 4. Kubernetes HPA pod scaling — NOT measured here

The `scraper-hpa.yaml` targets 60% CPU utilization, min 1 / max 5 replicas.
This build environment has **no Kubernetes cluster or metrics-server**, so pod
autoscaling behavior was **not measured** and no scaling numbers are claimed.

To reproduce on minikube:

```bash
minikube start --cpus 4 --memory 6g
minikube addons enable metrics-server
minikube addons enable ingress
# build images into minikube's docker daemon
eval $(minikube docker-env)
docker build -t trackfolio/scraper:latest      services/scraper-go
# ... (other services) ...
kubectl apply -f infra/k8s/
# drive load and watch the HPA react:
kubectl -n trackfolio get hpa scraper -w
# in another shell, hammer the scraper Service (via port-forward):
kubectl -n trackfolio port-forward svc/scraper 8081:8081
hey -z 3m -c 100 -m POST http://localhost:8081/trigger
```

Expected qualitative behavior: sustained CPU above 60% on the scraper pod drives
the HPA to add replicas up to the max of 5, then scale back down after the load
stops (120 s stabilization window per the manifest).

## How to reproduce sections 1–3

```bash
# start local postgres + redis, then the scraper pointed at a fixture:
go build -o /tmp/scraper ./services/scraper-go
MONGO_URI="" SCRAPER_GREENHOUSE_BASE=http://127.0.0.1:9098 \
  SCRAPER_SOURCES=greenhouse:acme SCRAPER_WORKERS=5 /tmp/scraper &
hey -n 5000 -c 50 http://localhost:8081/metrics        # section 1
# trigger a run and read /metrics duration_ms             # section 2
hey -n 200 -c 40 -m POST http://localhost:8081/trigger # section 3
```
