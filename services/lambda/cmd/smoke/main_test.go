package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeAttachesBearerTokenOnlyOnManagementAndNeverFollowsRedirect(t *testing.T) {
	destinationCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/links":
			if r.Header.Get("Authorization") != "Bearer test-access-token" {
				t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"code":"created","version":1}`))
		case "/r/created":
			if r.Header.Get("Authorization") != "" {
				t.Error("public redirect included Authorization header")
			}
			w.Header().Set("Location", "/destination")
			w.WriteHeader(302)
		case "/destination":
			destinationCalls++
		}
	}))
	defer server.Close()
	p := newProbe("ap-southeast-2")
	if _, err := p.check(context.Background(), "create", "POST", server.URL+"/api/links", `{"url":"https://example.com"}`, "test-access-token", 201); err != nil {
		t.Fatal(err)
	}
	if _, err := p.check(context.Background(), "redirect", "GET", server.URL+"/r/created", "", "", 302); err != nil {
		t.Fatal(err)
	}
	if destinationCalls != 0 || len(p.results) != 2 {
		t.Fatal("probe followed a redirect or lost evidence")
	}
	if _, err := p.check(context.Background(), "wrong status", "GET", server.URL+"/r/created", "", "", 200); err == nil {
		t.Fatal("probe silently accepted a failed check")
	}
}

func TestAuthenticatePKCEExchangesCodeWithS256Verifier(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/authorize":
			if r.URL.Query().Get("code_challenge_method") != "S256" || r.URL.Query().Get("code_challenge") == "" {
				t.Error("missing PKCE S256 challenge")
			}
			http.SetCookie(w, &http.Cookie{Name: "XSRF-TOKEN", Value: "csrf-123", Path: "/"})
			http.Redirect(w, r, serverURL+"/login?"+r.URL.RawQuery, http.StatusFound)
		case "/login":
			if r.Method == "GET" {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<form method="POST"><input type="hidden" name="_csrf" value="csrf-123"/></form>`))
				return
			}
			_ = r.ParseForm()
			if r.FormValue("_csrf") != "csrf-123" || r.FormValue("username") != "alice@example.com" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			redirectURI := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, redirectURI+"?code=auth-code-xyz&state="+state, http.StatusFound)
		case "/oauth2/token":
			_ = r.ParseForm()
			if r.FormValue("grant_type") != "authorization_code" || r.FormValue("code") != "auth-code-xyz" || len(r.FormValue("code_verifier")) < 43 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"acc-123","id_token":"id-123","refresh_token":"ref-123"}`))
		}
	}))
	defer server.Close()
	serverURL = server.URL

	tokens, err := authenticatePKCE(
		context.Background(),
		server.URL,
		"client-123",
		"https://npt-shortenlink.dev/auth/callback/",
		"alice@example.com",
		"Password123!",
		"openid email shortly-api/links.read shortly-api/links.write",
	)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "acc-123" || tokens.IDToken != "id-123" || tokens.RefreshToken != "ref-123" {
		t.Fatalf("unexpected tokens: %+v", tokens)
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
		p := newProbe("ap-southeast-2")
		err := p.redirect(context.Background(), "before expiry", server.URL, http.StatusFound)
		server.Close()
		if (err != nil) != tc.wantError {
			t.Fatalf("cache=%q location=%q: error=%v", tc.cache, tc.location, err)
		}
	}
}
