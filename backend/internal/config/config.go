package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
)

type Config struct {
	TableName             string
	EventBusName          string
	WSEndpoint            string
	PublicRedirectBaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		TableName:    strings.TrimSpace(os.Getenv("TABLE_NAME")),
		EventBusName: strings.TrimSpace(os.Getenv("EVENT_BUS")),
	}
	if cfg.TableName == "" {
		return nil, fmt.Errorf("TABLE_NAME is required")
	}
	for key, dest := range map[string]*string{"WS_ENDPOINT": &cfg.WSEndpoint, "PUBLIC_REDIRECT_BASE_URL": &cfg.PublicRedirectBaseURL} {
		raw := os.Getenv(key)
		if raw == "" {
			continue
		}
		normalized, err := domain.NormalizeURL(raw)
		if err != nil {
			return nil, fmt.Errorf("%s must be a valid HTTP endpoint", key)
		}
		u, _ := url.Parse(normalized)
		if u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return nil, fmt.Errorf("%s must not include credentials, query or fragment", key)
		}
		*dest = strings.TrimRight(normalized, "/")
	}
	return cfg, nil
}
