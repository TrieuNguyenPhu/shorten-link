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
	if _, err := netip.ParseAddr(host); err != nil {
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
