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

	publicBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if publicBaseURL == "" {
		publicBaseURL = "http://localhost:8080"
	}
	parsedBaseURL, err := url.Parse(publicBaseURL)
	if err != nil || parsedBaseURL.Host == "" || (parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https") {
		return Config{}, errors.New("PUBLIC_BASE_URL must be an absolute http or https URL")
	}

	allowedOrigins, err := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
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
		origin := strings.TrimRight(strings.TrimSpace(entry), "/")
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
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
