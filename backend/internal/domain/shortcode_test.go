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
		{"https://127.0.0.1/", "https://127.0.0.1/"},
		{"https://127.0.0.1./", "https://127.0.0.1./"},
		{"https://123.example.com/", "https://123.example.com/"},
		{"https://example123.com/", "https://example123.com/"},
		{"https://127.0.0.999/", ""}, {"https://256.1.2.3/", ""},
		{"https://127.0.0.999./", ""}, {"https://1.2.3.4.5/", ""},
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

func TestValidateTargetURLBlocksPrivateHostsAndExcessiveLength(t *testing.T) {
	if got, err := ValidateTargetURL("https://example.com/docs?q=1"); err != nil || got != "https://example.com/docs?q=1" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := ValidateTargetURL("https://8.8.8.8/dns"); err != nil || got != "https://8.8.8.8/dns" {
		t.Fatalf("got %q, %v", got, err)
	}
	blocked := []string{
		"http://localhost:8080/",
		"https://sub.localhost/path",
		"http://internal.local/admin",
		"http://127.0.0.1/admin",
		"http://127.0.0.1./admin",
		"https://[::1]/",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/internal",
		"http://172.16.0.1/internal",
		"http://192.168.1.1/router",
		"http://0.0.0.0/",
		"https://[fe80::1]/",
		"https://[fd00::1]/",
		"https://example.com/" + strings.Repeat("a", MaxTargetURLLength),
	}
	for _, raw := range blocked {
		if got, err := ValidateTargetURL(raw); err == nil {
			t.Fatalf("accepted blocked target URL %q as %q", raw[:min(len(raw), 64)], got)
		}
	}
}

func TestIsValidShortCodeAndIsSafeTargetURL(t *testing.T) {
	if !IsValidShortCode("Abc1234") || IsValidShortCode("short") || IsValidShortCode("toolong8") || IsValidShortCode("bad-chr") {
		t.Fatal("invalid short code validation")
	}
	if !IsSafeTargetURL("https://devil.com/medieval-history") {
		t.Fatal("false positive on substring match")
	}
	for _, unsafe := range []string{"https://evil.example.org", "https://example.org/malware-kit", "https://phishing.example.org", "http://127.0.0.1/"} {
		if IsSafeTargetURL(unsafe) {
			t.Fatalf("accepted unsafe URL %q", unsafe)
		}
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
