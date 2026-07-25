package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	StorageMemory   = "memory"
	StorageDynamoDB = "dynamodb"
)

type Config struct {
	Address            string
	PublicBaseURL      string
	StorageDriver      string
	LinksTable         string
	CORSAllowedOrigins []string
}

func Load(runningInLambda bool) (Config, error) {
	storageDriver := strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_DRIVER")))
	if storageDriver == "" {
		if runningInLambda {
			storageDriver = StorageDynamoDB
		} else {
			storageDriver = StorageMemory
		}
	}

	if storageDriver != StorageMemory && storageDriver != StorageDynamoDB {
		return Config{}, fmt.Errorf("unsupported STORAGE_DRIVER %q", storageDriver)
	}

	linksTable := strings.TrimSpace(os.Getenv("LINKS_TABLE_NAME"))
	if storageDriver == StorageDynamoDB && linksTable == "" {
		return Config{}, errors.New("LINKS_TABLE_NAME is required when STORAGE_DRIVER=dynamodb")
	}

	address := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if address == "" {
		port := strings.TrimSpace(os.Getenv("PORT"))
		if port == "" {
			port = "8080"
		}
		address = ":" + port
	}

	rawPublicBaseURL := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	if rawPublicBaseURL == "" {
		if runningInLambda {
			return Config{}, errors.New("PUBLIC_BASE_URL is required when running in Lambda")
		}
		rawPublicBaseURL = "http://localhost:8080"
	}
	publicBaseURL, err := normalizeOrigin(rawPublicBaseURL)
	if err != nil {
		return Config{}, errors.New("PUBLIC_BASE_URL must be an absolute http or https origin")
	}

	rawAllowedOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if rawAllowedOrigins == "" && runningInLambda {
		return Config{}, errors.New("CORS_ALLOWED_ORIGINS is required when running in Lambda")
	}
	allowedOrigins, err := parseAllowedOrigins(rawAllowedOrigins)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Address:            address,
		PublicBaseURL:      publicBaseURL,
		StorageDriver:      storageDriver,
		LinksTable:         linksTable,
		CORSAllowedOrigins: allowedOrigins,
	}, nil
}

func parseAllowedOrigins(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "http://localhost:3000"
	}

	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for _, entry := range strings.Split(raw, ",") {
		origin, err := normalizeOrigin(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid origin %q in CORS_ALLOWED_ORIGINS", entry)
		}
		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins, nil
}

func normalizeOrigin(raw string) (string, error) {
	origin := strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("origin must contain only an http or https scheme and host")
	}
	return origin, nil
}
