package domain

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateShortCode(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		code, err := GenerateShortCode(7)
		if err != nil || len(code) != 7 || strings.Trim(code, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz") != "" {
			t.Fatalf("invalid code %q: %v", code, err)
		}
		if seen[code] {
			t.Fatalf("repeated code %q", code)
		}
		seen[code] = true
	}
	for _, length := range []int{-1, 0} {
		if _, err := GenerateShortCode(length); err == nil {
			t.Fatalf("accepted length %d", length)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"https://example.com/a?q=b#part", "https://example.com/a?q=b#part"},
		{"  HTTP://example.com/path  ", "http://example.com/path"},
		{"http://localhost:8080/", "http://localhost:8080/"},
		{"https://[::1]/", "https://[::1]/"},
		{"", ""}, {"javascript:alert(1)", ""}, {"ftp://example.com", ""},
		{"https:///path", ""}, {"example.com", ""}, {"//example.com", ""},
		{"https://example.com:99999", ""}, {"https://example.com:bad", ""},
		{"https://example.com\r\nX-Test: true", ""}, {"https://exa mple.com", ""},
		{"https://[garbage]/", ""}, {"https://-bad.example/", ""},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := NormalizeURL(tc.raw)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("accepted %q as %q", tc.raw, got)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestFormatCreatedAt(t *testing.T) {
	zone := time.FixedZone("UTC+7", 7*60*60)
	first := time.Date(2026, 10, 27, 15, 0, 0, 0, zone)
	got := FormatCreatedAt(first)
	if got != "2026-10-27T08:00:00.000Z" {
		t.Fatal(got)
	}
	next := FormatCreatedAt(first.Add(100 * time.Millisecond))
	if next != "2026-10-27T08:00:00.100Z" || got >= next {
		t.Fatalf("bad sort order: %q, %q", got, next)
	}
	if got := FormatCreatedAt(first.Add(123456789 * time.Nanosecond)); got != "2026-10-27T08:00:00.123Z" {
		t.Fatal(got)
	}
}
