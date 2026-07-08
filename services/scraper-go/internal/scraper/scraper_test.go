package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"trackfolio/scraper-go/internal/config"
)

const greenhousePayload = `{"jobs":[
  {"id":1,"title":"Senior Go Engineer","updated_at":"2026-06-01T10:00:00-04:00",
   "absolute_url":"https://boards.greenhouse.io/acme/jobs/1",
   "location":{"name":"Remote - US"},
   "content":"&lt;p&gt;Build &lt;strong&gt;distributed&lt;/strong&gt; systems in Go.&lt;/p&gt;"}
]}`

const leverPayload = `[
  {"id":"abc","text":"Backend Engineer","hostedUrl":"https://jobs.lever.co/acme/abc",
   "createdAt":1717243200000,"descriptionPlain":"Work on Python services.",
   "description":"<p>Work on Python services.</p>",
   "categories":{"location":"New York","team":"Platform"}}
]`

func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/boards/acme/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(greenhousePayload))
	})
	mux.HandleFunc("/v0/postings/acme", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(leverPayload))
	})
	return httptest.NewServer(mux)
}

func TestGreenhouseFetcher(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()

	cfg := config.Config{GreenhouseBase: srv.URL}
	f, err := New(srv.Client(), cfg, config.Source{Kind: "greenhouse", Board: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	j := jobs[0]
	if j.Title != "Senior Go Engineer" || j.Location != "Remote - US" {
		t.Errorf("unexpected fields: %+v", j)
	}
	if j.URL != "https://boards.greenhouse.io/acme/jobs/1" {
		t.Errorf("bad url: %s", j.URL)
	}
	// HTML entities must be decoded and tags stripped for the match service.
	if want := "Build distributed systems in Go."; j.RawText != want {
		t.Errorf("raw text = %q, want %q", j.RawText, want)
	}
	if j.PostedDate == nil || j.PostedDate.Year() != 2026 {
		t.Errorf("posted date not parsed: %v", j.PostedDate)
	}
}

func TestLeverFetcher(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()

	cfg := config.Config{LeverBase: srv.URL}
	f, err := New(srv.Client(), cfg, config.Source{Kind: "lever", Board: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := f.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	j := jobs[0]
	if j.Title != "Backend Engineer" || j.Location != "New York" {
		t.Errorf("unexpected fields: %+v", j)
	}
	if j.RawText != "Work on Python services." {
		t.Errorf("bad raw text: %q", j.RawText)
	}
	if j.PostedDate == nil {
		t.Error("posted date should be parsed from createdAt millis")
	}
}

func TestUnknownSource(t *testing.T) {
	_, err := New(http.DefaultClient, config.Config{}, config.Source{Kind: "workday", Board: "x"})
	if err == nil {
		t.Fatal("expected error for unknown source kind")
	}
}

func TestStripHTML(t *testing.T) {
	got := stripHTML("<p>Hello&nbsp;&amp; welcome to <b>Go</b></p>")
	if got != "Hello & welcome to Go" {
		t.Errorf("stripHTML = %q", got)
	}
}
