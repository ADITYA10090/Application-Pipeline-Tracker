package model

import "time"

// Job is the normalized representation of a scraped posting, independent of
// which board it came from.
type Job struct {
	Company    string
	Title      string
	URL        string
	Location   string
	PostedDate *time.Time
	RawHTML    string
	RawText    string
}
