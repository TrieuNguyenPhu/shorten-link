package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
)

func TestModerationCheckPreservesInputAndRejectsKeywords(t *testing.T) {
	for _, tc := range []struct {
		target string
		safe   bool
	}{
		{"https://example.org/docs", true}, {"https://devil.com/medieval", true},
		{"https://example.org/MALWARE", false}, {"https://phishing.example.org", false},
		{"https://example.org/?q=evil", false}, {"http://127.0.0.1/admin", false},
		{"http://169.254.169.254/latest/meta-data/", false},
	} {
		t.Run(tc.target, func(t *testing.T) {
			out, err := CheckSafety(context.Background(), domain.ModerationInput{Code: "Abc1234", Target: tc.target})
			if err != nil || out.Code != "Abc1234" || out.Target != tc.target || out.Safe != tc.safe {
				t.Fatal(out, err)
			}
			b, err := json.Marshal(out)
			var decoded map[string]any
			if err != nil || json.Unmarshal(b, &decoded) != nil || len(decoded) != 3 || decoded["safe"] != tc.safe || decoded["target"] != tc.target {
				t.Fatal(string(b), err)
			}
		})
	}
	for _, in := range []domain.ModerationInput{{}, {Code: "Abc1234"}, {Target: "https://example.org"}, {Code: " ", Target: "https://example.org"}} {
		if _, err := CheckSafety(context.Background(), in); err == nil {
			t.Fatal("missing input accepted", in)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CheckSafety(ctx, domain.ModerationInput{Code: "Abc1234", Target: "https://example.org"}); err == nil {
		t.Fatal("cancelled task accepted")
	}
}
