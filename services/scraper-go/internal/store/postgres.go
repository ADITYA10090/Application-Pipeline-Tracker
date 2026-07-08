package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"trackfolio/scraper-go/internal/model"
)

// Postgres persists structured job/company fields.
type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() { p.pool.Close() }

// UpsertCompany returns the id for a company name, creating it if absent.
func (p *Postgres) UpsertCompany(ctx context.Context, name string) (int, error) {
	var id int
	err := p.pool.QueryRow(ctx, `
		INSERT INTO companies (name) VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, name).Scan(&id)
	return id, err
}

// UpsertJob inserts a posting keyed by its unique URL. It returns inserted=true
// only when a brand-new row was created, so callers can count fresh jobs.
// mongoRef links to the raw JD document stored in Mongo.
func (p *Postgres) UpsertJob(ctx context.Context, j model.Job, companyID int, mongoRef string) (id int, inserted bool, err error) {
	err = p.pool.QueryRow(ctx, `
		INSERT INTO job_postings (company_id, title, url, location, posted_date, mongo_jd_ref)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (url) DO UPDATE
			SET title = EXCLUDED.title,
			    location = EXCLUDED.location,
			    posted_date = EXCLUDED.posted_date
		RETURNING id, (xmax = 0) AS inserted`,
		companyID, j.Title, j.URL, j.Location, j.PostedDate, mongoRef,
	).Scan(&id, &inserted)
	return id, inserted, err
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }
