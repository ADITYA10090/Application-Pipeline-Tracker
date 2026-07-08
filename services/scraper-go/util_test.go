package main

import "testing"

func TestStripHTML(t *testing.T) {
	in := "<p>Senior <b>Go</b> Engineer</p>&nbsp;<ul><li>5+ years</li></ul>"
	got := stripHTML(in)
	want := "Senior Go Engineer 5+ years"
	if got != want {
		t.Fatalf("stripHTML: got %q want %q", got, want)
	}
}

func TestURLHashStableAndDistinct(t *testing.T) {
	a := urlHash("https://example.com/jobs/1")
	aTrim := urlHash("  https://example.com/jobs/1  ")
	b := urlHash("https://example.com/jobs/2")
	if a != aTrim {
		t.Errorf("urlHash should ignore surrounding whitespace")
	}
	if a == b {
		t.Errorf("distinct URLs must hash differently")
	}
	if len(a) != 64 {
		t.Errorf("expected sha256 hex length 64, got %d", len(a))
	}
}

func TestParseSources(t *testing.T) {
	got := parseSources("greenhouse|stripe, lever|netflix ,remoteok|, ,bad")
	if len(got) != 4 {
		t.Fatalf("expected 4 sources, got %d: %+v", len(got), got)
	}
	if got[0].Kind != "greenhouse" || got[0].Slug != "stripe" {
		t.Errorf("bad first source: %+v", got[0])
	}
	if got[2].Kind != "remoteok" || got[2].Slug != "" {
		t.Errorf("bad remoteok source: %+v", got[2])
	}
	if got[3].Kind != "bad" {
		t.Errorf("bad trailing source: %+v", got[3])
	}
}
