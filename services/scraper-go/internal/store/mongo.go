package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Mongo persists raw JD text/HTML that the match service consumes later.
type Mongo struct {
	client *mongo.Client
	coll   *mongo.Collection
}

// JobDescription is one document in the job_descriptions collection.
type JobDescription struct {
	JobURL    string    `bson:"job_url"`
	Company   string    `bson:"company"`
	Title     string    `bson:"title"`
	RawHTML   string    `bson:"raw_html"`
	RawText   string    `bson:"raw_text"`
	ScrapedAt time.Time `bson:"scraped_at"`
}

func NewMongo(ctx context.Context, uri, db string) (*Mongo, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return &Mongo{client: client, coll: client.Database(db).Collection("job_descriptions")}, nil
}

func (m *Mongo) Close(ctx context.Context) { _ = m.client.Disconnect(ctx) }

// SaveJD upserts the raw description keyed by job URL and returns the document's
// ObjectId hex, which is stored back on the Postgres row as mongo_jd_ref.
func (m *Mongo) SaveJD(ctx context.Context, jd JobDescription) (string, error) {
	filter := map[string]any{"job_url": jd.JobURL}
	update := map[string]any{"$set": jd}
	opts := options.Update().SetUpsert(true)
	res, err := m.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return "", err
	}
	if res.UpsertedID != nil {
		if oid, ok := res.UpsertedID.(primitive.ObjectID); ok {
			return oid.Hex(), nil
		}
	}
	// Already existed: fetch its id.
	var doc struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := m.coll.FindOne(ctx, filter).Decode(&doc); err == nil {
		return doc.ID.Hex(), nil
	}
	return "", nil
}
