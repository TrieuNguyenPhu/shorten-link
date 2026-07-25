package config

import (
	"reflect"
	"testing"
)

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"STORAGE_DRIVER",
		"LINKS_TABLE_NAME",
		"HTTP_ADDR",
		"PORT",
		"PUBLIC_BASE_URL",
		"CORS_ALLOWED_ORIGINS",
	} {
		t.Setenv(name, "")
	}
}

func TestLoadLocalDefaults(t *testing.T) {
	clearConfigEnvironment(t)

	got, err := Load(false)
	if err != nil {
		t.Fatalf("Load(false) error = %v", err)
	}
	if got.StorageDriver != StorageMemory || got.Address != ":8080" ||
		got.PublicBaseURL != "http://localhost:8080" ||
		!reflect.DeepEqual(got.CORSAllowedOrigins, []string{"http://localhost:3000"}) {
		t.Fatalf("Load(false) = %#v", got)
	}
}

func TestLoadLambdaRequiresTable(t *testing.T) {
	clearConfigEnvironment(t)

	if _, err := Load(true); err == nil {
		t.Fatal("Load(true) returned no error without LINKS_TABLE_NAME")
	}
}

func TestLoadLambdaRequiresPublicConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		publicBaseURL      string
		corsAllowedOrigins string
	}{
		{name: "public base URL"},
		{
			name:          "CORS allowed origins",
			publicBaseURL: "https://npt-shortenlink.dev",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearConfigEnvironment(t)
			t.Setenv("LINKS_TABLE_NAME", "links")
			t.Setenv("PUBLIC_BASE_URL", test.publicBaseURL)
			t.Setenv("CORS_ALLOWED_ORIGINS", test.corsAllowedOrigins)

			if _, err := Load(true); err == nil {
				t.Fatalf("Load(true) accepted missing %s", test.name)
			}
		})
	}
}

func TestLoadLambdaConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("LINKS_TABLE_NAME", "links")
	t.Setenv("PUBLIC_BASE_URL", "https://npt-shortenlink.dev/")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://npt-shortenlink.dev")

	got, err := Load(true)
	if err != nil {
		t.Fatalf("Load(true) error = %v", err)
	}
	if got.StorageDriver != StorageDynamoDB || got.LinksTable != "links" ||
		got.PublicBaseURL != "https://npt-shortenlink.dev" ||
		!reflect.DeepEqual(got.CORSAllowedOrigins, []string{"https://npt-shortenlink.dev"}) {
		t.Fatalf("Load(true) = %#v", got)
	}
}

func TestLoadDynamoDBConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("STORAGE_DRIVER", "dynamodb")
	t.Setenv("LINKS_TABLE_NAME", "links")
	t.Setenv("PUBLIC_BASE_URL", "https://npt-shortenlink.dev/")
	t.Setenv(
		"CORS_ALLOWED_ORIGINS",
		"https://npt-shortenlink.dev, https://npt-shortenlink.dev",
	)

	got, err := Load(false)
	if err != nil {
		t.Fatalf("Load(false) error = %v", err)
	}
	if got.LinksTable != "links" || got.PublicBaseURL != "https://npt-shortenlink.dev" ||
		!reflect.DeepEqual(got.CORSAllowedOrigins, []string{"https://npt-shortenlink.dev"}) {
		t.Fatalf("Load(false) = %#v", got)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "storage driver", key: "STORAGE_DRIVER", value: "filesystem"},
		{name: "public base URL", key: "PUBLIC_BASE_URL", value: "ftp://example.com"},
		{name: "public base URL path", key: "PUBLIC_BASE_URL", value: "https://example.com/links"},
		{name: "public base URL credentials", key: "PUBLIC_BASE_URL", value: "https://user:secret@example.com"},
		{name: "CORS origin path", key: "CORS_ALLOWED_ORIGINS", value: "https://example.com/path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearConfigEnvironment(t)
			t.Setenv(test.key, test.value)
			if _, err := Load(false); err == nil {
				t.Fatalf("Load(false) accepted %s=%q", test.key, test.value)
			}
		})
	}
}
