package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestProbeSignsOnlyManagementAndNeverFollowsRedirect(t *testing.T) {
	destinationCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/links":
			if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") || r.Header.Get("X-Amz-Security-Token") != "temporary-token" {
				t.Error("management request is not signed with the provided role")
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"code":"created"}`))
		case "/r/created":
			if r.Header.Get("Authorization") != "" {
				t.Error("public redirect was signed")
			}
			w.Header().Set("Location", "/destination")
			w.WriteHeader(302)
		case "/destination":
			destinationCalls++
		}
	}))
	defer server.Close()
	p := newProbe(aws.Credentials{AccessKeyID: "test-key", SecretAccessKey: "test-secret", SessionToken: "temporary-token"}, "ap-southeast-1")
	if _, err := p.check(context.Background(), "create", "POST", server.URL+"/api/links", `{"url":"https://example.com"}`, true, 201); err != nil {
		t.Fatal(err)
	}
	if _, err := p.check(context.Background(), "redirect", "GET", server.URL+"/r/created", "", false, 302); err != nil {
		t.Fatal(err)
	}
	if destinationCalls != 0 || len(p.results) != 2 {
		t.Fatal("probe followed a redirect or lost evidence")
	}
	if _, err := p.check(context.Background(), "wrong status", "GET", server.URL+"/r/created", "", false, 200); err == nil {
		t.Fatal("probe silently accepted a failed check")
	}
}

func TestSmokeTargetsExcludeUnavailableCloudFront(t *testing.T) {
	if targets := redirectTargets("https://api.example", ""); len(targets) != 1 || targets[0] != "https://api.example" {
		t.Fatalf("backend-only targets=%v", targets)
	}
	if targets := redirectTargets("https://api.example", "https://site.example"); len(targets) != 2 || targets[1] != "https://site.example" {
		t.Fatalf("full-stack targets=%v", targets)
	}
}

func TestRedirectProbeRejectsCacheableOrWrongDestinations(t *testing.T) {
	for _, tc := range []struct {
		cache, location string
		wantError       bool
	}{
		{"no-store", "https://example.com/phase-02-smoke?q=1#verified", false},
		{"max-age=60", "https://example.com/phase-02-smoke?q=1#verified", true},
		{"no-store", "https://example.com/wrong", true},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Cache-Control", tc.cache)
			w.Header().Set("Location", tc.location)
			w.WriteHeader(http.StatusFound)
		}))
		p := newProbe(aws.Credentials{}, "ap-southeast-2")
		err := p.redirect(context.Background(), "before expiry", server.URL, http.StatusFound)
		server.Close()
		if (err != nil) != tc.wantError {
			t.Fatalf("cache=%q location=%q: error=%v", tc.cache, tc.location, err)
		}
	}
}
