package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store wraps the two persistence backends. Postgres holds structured fields;
// Mongo holds the bulky raw JD text/HTML keyed back to the postgres row.
type Store struct {
	pg    *pgxpool.Pool
	mongo *mongo.Client
	jdCol *mongo.Collection
}

func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	pg, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		return nil, err
	}
	if err := pg.Ping(ctx); err != nil {
		return nil, err
	}

	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}
	if err := mc.Ping(ctx, nil); err != nil {
		return nil, err
	}
	col := mc.Database(cfg.MongoDB).Collection("job_descriptions")

	return &Store{pg: pg, mongo: mc, jdCol: col}, nil
}

func (s *Store) Close(ctx context.Context) {
	if s.pg != nil {
		s.pg.Close()
	}
	if s.mongo != nil {
		_ = s.mongo.Disconnect(ctx)
	}
}

// upsertCompany returns the company id, creating the row on first sight.
func (s *Store) upsertCompany(ctx context.Context, name string) (int, error) {
	if name == "" {
		name = "unknown"
	}
	var id int
	err := s.pg.QueryRow(ctx,
		`INSERT INTO companies (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`, name).Scan(&id)
	return id, err
}

// PersistJob writes the raw JD to Mongo, then the structured row to Postgres,
// linking the two via mongo_jd_ref. Returns true if a new posting was inserted
// (false if the URL already existed).
func (s *Store) PersistJob(ctx context.Context, j Job) (bool, error) {
	companyID, err := s.upsertCompany(ctx, j.Company)
	if err != nil {
		return false, err
	}

	res, err := s.jdCol.InsertOne(ctx, bson.M{
		"job_url":    j.URL,
		"raw_html":   j.RawHTML,
		"raw_text":   j.RawText,
		"scraped_at": time.Now().UTC(),
	})
	if err != nil {
		return false, err
	}
	mongoRef := ""
	if oid, ok := res.InsertedID.(interface{ Hex() string }); ok {
		mongoRef = oid.Hex()
	}

	var posted any
	if !j.PostedDate.IsZero() {
		posted = j.PostedDate
	}

	tag, err := s.pg.Exec(ctx,
		`INSERT INTO job_postings (company_id, title, url, location, posted_date, mongo_jd_ref)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (url) DO NOTHING`,
		companyID, j.Title, j.URL, j.Location, posted, mongoRef)
	if err != nil {
		return false, err
	}
	inserted := tag.RowsAffected() > 0
	// If the posting already existed, the JD doc we just inserted is redundant.
	if !inserted {
		_, _ = s.jdCol.DeleteOne(ctx, bson.M{"_id": res.InsertedID})
	}
	return inserted, nil
}
