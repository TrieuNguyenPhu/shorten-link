package domain

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

func GenerateShortCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("short code length must be positive")
	}
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		code[i] = alphabet[n.Int64()]
	}
	return string(code), nil
}

func NormalizeURL(raw string) (string, error) {
	invalid := errors.New("URL must be an absolute http or https URL with a valid host")
	if strings.ContainsFunc(raw, unicode.IsControl) {
		return "", invalid
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Opaque != "" || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", invalid
	}
	host := u.Hostname()
	if _, err := netip.ParseAddr(strings.TrimSuffix(host, ".")); err != nil {
		// Malformed dotted IPv4 addresses must not fall back to DNS validation.
		if strings.Contains(host, ".") && strings.Trim(host, "0123456789.") == "" {
			return "", invalid
		}
		if len(host) > 253 {
			return "", invalid
		}
		for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return "", invalid
			}
			for _, c := range label {
				if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
					return "", invalid
				}
			}
		}
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", invalid
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return "", invalid
	}
	return u.String(), nil
}

const (
	ShortCodeLength    = 7
	MaxTargetURLLength = 2048
)

func IsValidShortCode(code string) bool {
	if len(code) != ShortCodeLength {
		return false
	}
	for i := range len(code) {
		c := code[i]
		if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
			return false
		}
	}
	return true
}

// ValidateTargetURL normalizes a user-supplied destination URL and blocks
// excessive lengths, localhost/local hostnames, and non-public IP literals.
func ValidateTargetURL(raw string) (string, error) {
	if len(strings.TrimSpace(raw)) > MaxTargetURLLength {
		return "", errors.New("URL exceeds maximum allowed length")
	}
	normalized, err := NormalizeURL(raw)
	if err != nil {
		return "", err
	}
	if len(normalized) > MaxTargetURLLength {
		return "", errors.New("URL exceeds maximum allowed length")
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return "", errors.New("URL host must be publicly routable")
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.Unmap()
		if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
			return "", errors.New("URL IP address must be publicly routable")
		}
	}
	return normalized, nil
}

// IsSafeTargetURL validates the target URL and checks token boundaries for
// known unsafe keywords without false positives on words like "medieval" or "devil".
func IsSafeTargetURL(raw string) bool {
	normalized, err := ValidateTargetURL(raw)
	if err != nil {
		return false
	}
	tokens := strings.FieldsFunc(strings.ToLower(normalized), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	for _, token := range tokens {
		switch token {
		case "malware", "phishing", "evil":
			return false
		}
	}
	return true
}
