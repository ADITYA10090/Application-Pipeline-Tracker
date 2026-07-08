// Package scraper contains the per-board fetchers that turn public job-board
// JSON APIs into normalized model.Job values.
//
// We deliberately target JSON APIs (Greenhouse / Lever public board endpoints)
// rather than scraping arbitrary company HTML: the JSON contracts are stable
// and documented, so a fetcher only breaks if a company migrates ATS entirely,
// not every time they restyle their careers page.
package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"trackfolio/scraper-go/internal/config"
	"trackfolio/scraper-go/internal/model"
)

// Fetcher retrieves and normalizes the open postings for a single board.
type Fetcher interface {
	Fetch(ctx context.Context) ([]model.Job, error)
}

// New returns the Fetcher for a source, or an error for an unknown kind.
func New(client *http.Client, cfg config.Config, src config.Source) (Fetcher, error) {
	switch src.Kind {
	case "greenhouse":
		return &greenhouseFetcher{client: client, base: cfg.GreenhouseBase, board: src.Board}, nil
	case "lever":
		return &leverFetcher{client: client, base: cfg.LeverBase, board: src.Board}, nil
	default:
		return nil, fmt.Errorf("unknown source kind %q", src.Kind)
	}
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "trackfolio-scraper/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- Greenhouse -----------------------------------------------------------

type greenhouseFetcher struct {
	client *http.Client
	base   string
	board  string
}

type ghResponse struct {
	Jobs []ghJob `json:"jobs"`
}

type ghJob struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	UpdatedAt   string `json:"updated_at"`
	AbsoluteURL string `json:"absolute_url"`
	Content     string `json:"content"` // HTML-escaped description
	Location    struct {
		Name string `json:"name"`
	} `json:"location"`
}

func (g *greenhouseFetcher) Fetch(ctx context.Context) ([]model.Job, error) {
	url := fmt.Sprintf("%s/v1/boards/%s/jobs?content=true", strings.TrimRight(g.base, "/"), g.board)
	var resp ghResponse
	if err := getJSON(ctx, g.client, url, &resp); err != nil {
		return nil, fmt.Errorf("greenhouse %s: %w", g.board, err)
	}
	jobs := make([]model.Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		html := unescapeHTML(j.Content)
		jobs = append(jobs, model.Job{
			Company:    g.board,
			Title:      j.Title,
			URL:        j.AbsoluteURL,
			Location:   j.Location.Name,
			PostedDate: parseTime(j.UpdatedAt),
			RawHTML:    html,
			RawText:    stripHTML(html),
		})
	}
	return jobs, nil
}

// --- Lever ----------------------------------------------------------------

type leverFetcher struct {
	client *http.Client
	base   string
	board  string
}

type leverJob struct {
	ID              string `json:"id"`
	Text            string `json:"text"` // title
	HostedURL       string `json:"hostedUrl"`
	CreatedAt       int64  `json:"createdAt"` // epoch millis
	Description     string `json:"description"`
	DescriptionPlain string `json:"descriptionPlain"`
	Categories      struct {
		Location string `json:"location"`
		Team     string `json:"team"`
	} `json:"categories"`
}

func (l *leverFetcher) Fetch(ctx context.Context) ([]model.Job, error) {
	url := fmt.Sprintf("%s/v0/postings/%s?mode=json", strings.TrimRight(l.base, "/"), l.board)
	var resp []leverJob
	if err := getJSON(ctx, l.client, url, &resp); err != nil {
		return nil, fmt.Errorf("lever %s: %w", l.board, err)
	}
	jobs := make([]model.Job, 0, len(resp))
	for _, j := range resp {
		var posted *time.Time
		if j.CreatedAt > 0 {
			t := time.UnixMilli(j.CreatedAt).UTC()
			posted = &t
		}
		text := j.DescriptionPlain
		if text == "" {
			text = stripHTML(j.Description)
		}
		jobs = append(jobs, model.Job{
			Company:    l.board,
			Title:      j.Text,
			URL:        j.HostedURL,
			Location:   j.Categories.Location,
			PostedDate: posted,
			RawHTML:    j.Description,
			RawText:    text,
		})
	}
	return jobs, nil
}

// --- helpers --------------------------------------------------------------

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			t = t.UTC()
			return &t
		}
	}
	return nil
}

var tagRe = regexp.MustCompile(`<[^>]*>`)
var wsRe = regexp.MustCompile(`\s+`)

// stripHTML produces a plain-text approximation of an HTML fragment. Good
// enough for feeding a TF-IDF matcher; not a full HTML parser.
func stripHTML(s string) string {
	s = unescapeHTML(s)
	s = tagRe.ReplaceAllString(s, " ")
	s = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">").Replace(s)
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

func unescapeHTML(s string) string {
	// Greenhouse double-escapes; normalize the common entities.
	return strings.NewReplacer(
		"&lt;", "<", "&gt;", ">", "&amp;", "&", "&quot;", "\"", "&#39;", "'",
	).Replace(s)
}
