-- TrackFolio PostgreSQL schema
-- Structured data for companies, job postings, applications, and status history.

CREATE TABLE IF NOT EXISTS companies (
    id           SERIAL PRIMARY KEY,
    name         TEXT NOT NULL,
    website      TEXT,
    career_page_url TEXT,
    industry     TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS job_postings (
    id           SERIAL PRIMARY KEY,
    company_id   INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    url          TEXT NOT NULL UNIQUE,
    location     TEXT,
    posted_date  DATE,
    scraped_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    mongo_jd_ref TEXT,               -- ObjectId (hex) of the raw JD doc in Mongo
    status       TEXT NOT NULL DEFAULT 'open'   -- open | closed
);

CREATE INDEX IF NOT EXISTS idx_job_postings_company ON job_postings(company_id);

-- Pipeline status vocabulary shared with the frontend Kanban columns.
CREATE TABLE IF NOT EXISTS applications (
    id             SERIAL PRIMARY KEY,
    job_posting_id INTEGER NOT NULL REFERENCES job_postings(id) ON DELETE CASCADE,
    resume_version TEXT,
    applied_date   DATE,
    current_status TEXT NOT NULL DEFAULT 'wishlist'
        CHECK (current_status IN ('wishlist','applied','oa','interview','offer','rejected')),
    match_score    REAL,             -- 0..1 cosine similarity from the match service
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (job_posting_id)
);

CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(current_status);

CREATE TABLE IF NOT EXISTS status_events (
    id             SERIAL PRIMARY KEY,
    application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    status         TEXT NOT NULL,
    source         TEXT NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual','email_parsed')),
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_status_events_app ON status_events(application_id);

CREATE TABLE IF NOT EXISTS resume_versions (
    id                 SERIAL PRIMARY KEY,
    filename           TEXT NOT NULL,
    content_hash       TEXT NOT NULL,
    tailored_for_job_id INTEGER REFERENCES job_postings(id) ON DELETE SET NULL,
    uploaded_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
