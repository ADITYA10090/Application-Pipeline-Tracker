# API Reference

## Node API Gateway (`:8080`)

Base path: `/api`. All request/response bodies are JSON.

### `GET /health`
`200 → { "status": "ok" }`

### `GET /api/applications`
List applications with joined job/company fields.

Query params (optional): `status` (one of the pipeline statuses), `company`
(exact company name).

```
GET /api/applications?status=interview
200 → [ { id, job_posting_id, current_status, match_score, title, url,
          location, company, mongo_jd_ref, applied_date, resume_version,
          created_at, updated_at }, ... ]
```

### `POST /api/applications`
Create an application. Also writes the initial `status_events` row.

```json
{ "job_posting_id": 1, "current_status": "applied",
  "match_score": 0.42, "resume_version": "v1" }
```
`201 → { ...application }` · `400` on validation error · `404` if
`job_posting_id` does not exist.

### `GET /api/applications/:id`
Single application with its full status history.
`200 → { ...application, status_events: [ { status, source, occurred_at } ] }`
· `404` if not found.

### `PATCH /api/applications/:id/status`
Update status and append a `status_events` record.

```json
{ "status": "interview", "source": "manual" }
```
`200 → { ...application }` · `400` on invalid status · `404` if not found.

### `GET /api/companies`
`200 → [ { id, name, website, career_page_url, industry, created_at,
           posting_count } ]`

### `GET /api/jobs`
Scraped postings not yet turned into applications.
`200 → [ { id, title, url, location, posted_date, scraped_at, company,
           mongo_jd_ref } ]`

### `POST /api/jobs/scrape-trigger`
Proxies to the Go scraper's `POST /trigger`.
`202 → { "status": "scrape started" }` · `502` if the scraper is unreachable.

### `POST /api/match-score`
Proxies to the Python match service after validation.

```json
{ "resume_text": "...", "job_description_text": "..." }
```
`200 → { score, matched_keywords, missing_keywords }` · `400` on validation
error · `502` if the match service is unreachable.

### `GET /api/dashboard/stats`
```
200 → {
  total_applications: number,
  by_status: { wishlist, applied, oa, interview, offer, rejected },
  response_rate: number,        // responded / applied
  avg_match_score: number,
  applications_over_time: [ { day: "YYYY-MM-DD", count } ]
}
```

---

## Go Scraper (`:8081`)

### `GET /health`
`200 → { "status": "ok" }`

### `POST /trigger`
Starts a scrape run in the background (non-blocking).
`202 → { "status": "scrape started" }`

### `GET /metrics`
```
200 → {
  jobs_fetched, jobs_inserted, duplicates, errors,
  queue_depth, last_run_unix, running, configured_srcs
}
```

---

## Python Match Service (`:8000`)

### `GET /health`
`200 → { "status": "ok" }`

### `POST /score`
```json
{ "resume_text": "...", "job_description_text": "..." }
```
`200 → { "score": 0.27, "matched_keywords": ["python","postgresql"],
         "missing_keywords": ["go","kubernetes"] }`
`422` on missing/empty fields (Pydantic validation).

**Scoring:** TF-IDF vectorization (1–2 grams, English stop words) + cosine
similarity in `[0, 1]`. Keyword lists come from a curated skill-alias
dictionary matched with word boundaries.
