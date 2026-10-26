// Package localconfig configures only host-run local tools, never production Lambda handlers.
package localconfig

import (
	"fmt"
	"net"
	"net/url"
	"os"
)

type Config struct{ Address, Origin string }

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
	c := Config{
		Address: env("API_ADDRESS", "127.0.0.1:8080"),
		Origin:  env("WEB_ORIGIN", "http://localhost:3000"),
	}
	host, _, err := net.SplitHostPort(c.Address)
	if err != nil || !loopback(host) {
		return c, fmt.Errorf("API_ADDRESS must bind to a loopback host and port")
	}
	if !localURL(c.Origin) {
		return c, fmt.Errorf("WEB_ORIGIN must be a loopback HTTP(S) origin")
	}
	return c, nil
}
