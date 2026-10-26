// Package localconfig configures only host-run local tools, never production Lambda handlers.
package localconfig

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Config struct{ Address, Endpoint, Table, Origin string }

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func loopback(host string) bool {
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func localURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && loopback(u.Hostname()) && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" && !u.ForceQuery
}

func Load() (Config, error) {
	c := Config{Address: env("API_ADDRESS", "127.0.0.1:8080"), Endpoint: env("DYNAMODB_ENDPOINT", "http://127.0.0.1:8000"), Table: env("DYNAMODB_TABLE", "shortlinks-local"), Origin: env("WEB_ORIGIN", "http://localhost:3000")}
	host, _, err := net.SplitHostPort(c.Address)
	if err != nil || !loopback(host) {
		return c, fmt.Errorf("API_ADDRESS must bind to a loopback host and port")
	}
	if !localURL(c.Endpoint) {
		return c, fmt.Errorf("DYNAMODB_ENDPOINT must be a loopback HTTP(S) origin without path or credentials")
	}
	if !localURL(c.Origin) {
		return c, fmt.Errorf("WEB_ORIGIN must be a loopback HTTP(S) origin")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]{3,255}$`).MatchString(c.Table) {
		return c, fmt.Errorf("DYNAMODB_TABLE must be a valid table name (3–255 characters)")
	}
	return c, nil
}

func (c Config) Client() *dynamodb.Client {
	// Local credentials are intentionally isolated from profiles, IMDS, and environment credentials.
	return dynamodb.NewFromConfig(aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("local", "local", ""),
		RetryMaxAttempts: 2,
		HTTPClient:       &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return fmt.Errorf("DynamoDB Local must not redirect") }},
	}, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(c.Endpoint) })
}
