# API Reference

Base URL: the Node gateway (default `http://localhost:8080`). The frontend calls
these same-origin via the dev proxy / k8s ingress.

## Health

### `GET /health`
Returns `{ "status": "ok" }`; `503` with `{ "status": "degraded" }` if Postgres
is unreachable.

## Applications

### `GET /api/applications?status=&company=`
List applications joined to their posting + company, newest first. Optional
filters: `status` (one of `wishlist|applied|oa|interview|offer|rejected`),
`company` (exact name).

### `GET /api/applications/:id`
One application plus its `status_events` history array.

### `POST /api/applications`
Create an application for a scraped posting.
```json
{ "job_posting_id": 5, "current_status": "wishlist",
  "resume_version": "v1", "match_score": 0.42 }
```
`job_posting_id` is required. `409` if an application already exists for that
posting; `404` if the posting doesn't exist. Records an initial `status_events`
row in the same transaction.

### `PATCH /api/applications/:id/status`
```json
{ "status": "interview", "source": "manual" }
```
Updates `current_status` and appends a `status_events` row atomically. `400` on
an invalid status, `404` if the application is missing.

## Jobs

### `GET /api/jobs?applied=false`
List scraped postings with company name and `has_application`. `applied=false`
returns only postings not yet in the pipeline.

### `POST /api/jobs/scrape-trigger`
Proxies to the Go scraper's `POST /trigger`. Returns `202 { "status": "started" }`
or `{ "status": "already_running" }`.

## Match scoring

### `POST /api/match-score`
Proxies to the Python match service.
```json
{ "resume_text": "...", "job_description_text": "..." }
```
→
```json
{ "score": 0.31, "matched_keywords": ["go","postgres"],
  "missing_keywords": ["terraform"] }
```

## Companies

### `GET /api/companies`
Companies with a `posting_count`.

## Dashboard

### `GET /api/dashboard/stats`
```json
{
  "total_applications": 5,
  "counts_by_status": { "wishlist": 1, "applied": 1, "oa": 1,
                        "interview": 1, "offer": 1, "rejected": 0 },
  "submitted": 4,
  "response_rate": 0.75,
  "avg_match_score": 0.638,
  "applications_over_time": [ { "date": "2026-07-08", "count": 5 } ]
}
```
`response_rate` = of applications actually submitted (status ≠ `wishlist`), the
fraction that received any response (`oa`/`interview`/`offer`/`rejected`).

---

## Scraper service (internal, default `http://localhost:8081`)

| Endpoint | Description |
|---|---|
| `POST /trigger` | Start a scrape run (single-flight; `409` if already running). |
| `GET /health` | Liveness. |
| `GET /metrics` | JSON counters: `runs`, `jobs_scraped`, `jobs_inserted`, `jobs_deduped`, `errors`, `queue_depth`, `last_run_unix`. |

## Match service (internal, default `http://localhost:8082`)

| Endpoint | Description |
|---|---|
| `POST /score` | `{resume_text, job_description_text}` → `{score, matched_keywords, missing_keywords}`. |
| `GET /health` | Liveness. |
