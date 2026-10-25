package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"shortlink/internal/domain"
	"shortlink/internal/ports"
)

var codePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

func ValidCode(code string) bool { return codePattern.MatchString(code) }

type CreateInput struct {
	URL       string     `json:"url"`
	Alias     string     `json:"alias,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type Service struct {
	repo     ports.LinkRepository
	now      func() time.Time
	generate func() (string, error)
}

func New(repo ports.LinkRepository) *Service {
	return &Service{repo: repo, now: time.Now, generate: randomCode}
}

func randomCode() (string, error) {
	var bytes [6]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func validateURL(destination string) error {
	if len(destination) == 0 || len(destination) > 2048 || !utf8.ValidString(destination) || strings.ContainsAny(destination, `\`) || strings.ContainsFunc(destination, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return fmt.Errorf("%w: destination must be an HTTP(S) URL of at most 2048 bytes without whitespace", domain.ErrInvalid)
	}
	u, err := url.Parse(destination)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return fmt.Errorf("%w: destination must be an absolute HTTP(S) URL without credentials", domain.ErrInvalid)
	}
	if strings.ContainsAny(u.Hostname(), "<>^|%") {
		return fmt.Errorf("%w: destination hostname is invalid", domain.ErrInvalid)
	}
	host := strings.TrimSuffix(u.Hostname(), ".")
	if strings.Count(host, ".") == 3 && strings.Trim(host, "0123456789.") == "" {
		if _, err := netip.ParseAddr(host); err != nil {
			return fmt.Errorf("%w: destination IPv4 address is invalid", domain.ErrInvalid)
		}
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return fmt.Errorf("%w: destination port must be between 0 and 65535", domain.ErrInvalid)
		}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Link, error) {
	if err := validateURL(input.URL); err != nil {
		return domain.Link{}, err
	}
	if input.Alias != "" && !ValidCode(input.Alias) {
		return domain.Link{}, fmt.Errorf("%w: alias must contain 3–32 letters, digits, hyphens, or underscores", domain.ErrInvalid)
	}
	now := s.now().UTC()
	if input.ExpiresAt != nil {
		expiry := input.ExpiresAt.UTC()
		if !expiry.After(now) || expiry.Year() > 9999 {
			return domain.Link{}, fmt.Errorf("%w: expiration must be in the future", domain.ErrInvalid)
		}
		input.ExpiresAt = &expiry
	}
	link := domain.Link{URL: input.URL, Enabled: true, CreatedAt: now, ExpiresAt: input.ExpiresAt}
	for range 5 {
		link.Code = input.Alias
		if link.Code == "" {
			code, err := s.generate()
			if err != nil {
				return domain.Link{}, err
			}
			link.Code = code
		}
		err := s.repo.Create(ctx, link)
		if !errors.Is(err, domain.ErrConflict) || input.Alias != "" {
			return link, err
		}
	}
	return domain.Link{}, domain.ErrUnavailable
}

func (s *Service) Get(ctx context.Context, code string) (domain.Link, error) {
	if !ValidCode(code) {
		return domain.Link{}, fmt.Errorf("%w: malformed short code", domain.ErrInvalid)
	}
	return s.repo.Get(ctx, code)
}

func (s *Service) Resolve(ctx context.Context, code string) (domain.Link, error) {
	link, err := s.Get(ctx, code)
	if err != nil {
		return domain.Link{}, err
	}
	if link.Status(s.now()) != "active" {
		return domain.Link{}, domain.ErrGone
	}
	return link, nil
}

func (s *Service) List(ctx context.Context, cursor string) (domain.Page, error) {
	return s.repo.List(ctx, cursor)
}

func (s *Service) SetEnabled(ctx context.Context, code string, enabled bool) (domain.Link, error) {
	if !ValidCode(code) {
		return domain.Link{}, fmt.Errorf("%w: malformed short code", domain.ErrInvalid)
	}
	return s.repo.SetEnabled(ctx, code, enabled)
}
