package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Job is the normalized shape produced by every source adapter, regardless of
// the upstream JSON schema.
type Job struct {
	Title      string    `json:"title"`
	Company    string    `json:"company"`
	URL        string    `json:"url"`
	Location   string    `json:"location"`
	PostedDate time.Time `json:"posted_date"`
	RawText    string    `json:"raw_text"`
	RawHTML    string    `json:"raw_html"`
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

// FetchSource dispatches to the right adapter based on the source kind and
// returns a normalized slice of jobs. A single failing source never aborts a
// whole run — the caller logs and continues.
func FetchSource(ctx context.Context, s Source) ([]Job, error) {
	switch s.Kind {
	case "greenhouse":
		return fetchGreenhouse(ctx, s.Slug)
	case "lever":
		return fetchLever(ctx, s.Slug)
	case "remoteok":
		return fetchRemoteOK(ctx)
	default:
		return nil, fmt.Errorf("unknown source kind %q", s.Kind)
	}
}

func getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// RemoteOK blocks the default Go user agent.
	req.Header.Set("User-Agent", "trackfolio-scraper/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- Greenhouse: https://boards-api.greenhouse.io/v1/boards/<token>/jobs?content=true

type ghResponse struct {
	Jobs []struct {
		Title    string `json:"title"`
		AbsURL   string `json:"absolute_url"`
		Content  string `json:"content"`
		UpdatedAt string `json:"updated_at"`
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
	} `json:"jobs"`
}

func fetchGreenhouse(ctx context.Context, token string) ([]Job, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", token)
	var r ghResponse
	if err := getJSON(ctx, url, &r); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(r.Jobs))
	for _, j := range r.Jobs {
		posted, _ := time.Parse(time.RFC3339, j.UpdatedAt)
		jobs = append(jobs, Job{
			Title:      j.Title,
			Company:    token,
			URL:        j.AbsURL,
			Location:   j.Location.Name,
			PostedDate: posted,
			RawHTML:    j.Content,
			RawText:    stripHTML(j.Content),
		})
	}
	return jobs, nil
}

// --- Lever: https://api.lever.co/v0/postings/<company>?mode=json

type leverPosting struct {
	Text            string `json:"text"`
	HostedURL       string `json:"hostedUrl"`
	DescriptionText string `json:"descriptionPlain"`
	Description     string `json:"description"`
	CreatedAt       int64  `json:"createdAt"`
	Categories      struct {
		Location string `json:"location"`
	} `json:"categories"`
}

func fetchLever(ctx context.Context, company string) ([]Job, error) {
	url := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", company)
	var postings []leverPosting
	if err := getJSON(ctx, url, &postings); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(postings))
	for _, p := range postings {
		jobs = append(jobs, Job{
			Title:      p.Text,
			Company:    company,
			URL:        p.HostedURL,
			Location:   p.Categories.Location,
			PostedDate: time.UnixMilli(p.CreatedAt),
			RawHTML:    p.Description,
			RawText:    p.DescriptionText,
		})
	}
	return jobs, nil
}

// --- RemoteOK: https://remoteok.com/api (first element is a legal notice)

type remoteOKItem struct {
	Position    string `json:"position"`
	Company     string `json:"company"`
	URL         string `json:"url"`
	Location    string `json:"location"`
	Date        string `json:"date"`
	Description string `json:"description"`
}

func fetchRemoteOK(ctx context.Context) ([]Job, error) {
	url := "https://remoteok.com/api"
	var raw []json.RawMessage
	if err := getJSON(ctx, url, &raw); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(raw))
	for _, item := range raw {
		var it remoteOKItem
		if err := json.Unmarshal(item, &it); err != nil {
			continue
		}
		if it.Position == "" || it.URL == "" { // skips the legal-notice element
			continue
		}
		posted, _ := time.Parse(time.RFC3339, it.Date)
		jobs = append(jobs, Job{
			Title:      it.Position,
			Company:    it.Company,
			URL:        it.URL,
			Location:   it.Location,
			PostedDate: posted,
			RawHTML:    it.Description,
			RawText:    stripHTML(it.Description),
		})
	}
	return jobs, nil
}
