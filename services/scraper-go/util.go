package main

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var tagRE = regexp.MustCompile(`<[^>]*>`)
var wsRE = regexp.MustCompile(`\s+`)

// stripHTML produces a plain-text approximation of an HTML fragment. It is not
// a full parser — good enough to give the match service readable JD text.
func stripHTML(s string) string {
	s = tagRE.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = wsRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// urlHash is the dedup key: a job is "the same" if its canonical URL matches.
func urlHash(url string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(url)))
	return hex.EncodeToString(sum[:])
}
