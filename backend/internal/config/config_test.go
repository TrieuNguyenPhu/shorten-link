package config

import "testing"

func clearConfig(t *testing.T) {
	t.Helper()
	for _, key := range []string{"TABLE_NAME", "EVENT_BUS", "WS_ENDPOINT", "PUBLIC_REDIRECT_BASE_URL"} {
		t.Setenv(key, "")
	}
}

func TestLoad(t *testing.T) {
	clearConfig(t)
	t.Setenv("TABLE_NAME", "url-shortener")
	t.Setenv("EVENT_BUS", "url-shortener-events")
	t.Setenv("WS_ENDPOINT", "https://ws.example.com/prod/")
	t.Setenv("PUBLIC_REDIRECT_BASE_URL", "https://api.example.com/")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TableName != "url-shortener" || cfg.EventBusName != "url-shortener-events" || cfg.WSEndpoint != "https://ws.example.com/prod" || cfg.PublicRedirectBaseURL != "https://api.example.com" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRequiresTable(t *testing.T) {
	clearConfig(t)
	if _, err := Load(); err == nil {
		t.Fatal("missing table accepted")
	}
	t.Setenv("TABLE_NAME", "  ")
	if _, err := Load(); err == nil {
		t.Fatal("blank table accepted")
	}
}

func TestLoadOptionalEndpoints(t *testing.T) {
	clearConfig(t)
	t.Setenv("TABLE_NAME", "url-shortener")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"PUBLIC_REDIRECT_BASE_URL", "WS_ENDPOINT"} {
		for _, value := range []string{"wss://example.com", "https:///path", "https://user:pass@example.com", "https://example.com?token=secret", "https://example.com/#part"} {
			t.Run(key+value, func(t *testing.T) {
				t.Setenv(key, value)
				if _, err := Load(); err == nil {
					t.Fatal("accepted invalid endpoint")
				}
			})
		}
	}
}
