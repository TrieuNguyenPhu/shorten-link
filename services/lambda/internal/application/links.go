package application

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
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

func ValidActor(actor string) bool {
	return len(actor) > 0 && len(actor) <= 128 && utf8.ValidString(actor) && !strings.ContainsFunc(actor, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
}

type CreateInput struct {
	URL       string     `json:"url"`
	Alias     string     `json:"alias,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type NullableTime struct {
	Set   bool
	Value *time.Time
}

func (n *NullableTime) UnmarshalJSON(data []byte) error {
	n.Set = true
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		n.Value = nil
		return nil
	}
	var t time.Time
	if err := json.Unmarshal(trimmed, &t); err != nil {
		return err
	}
	n.Value = &t
	return nil
}

type UpdateInput struct {
	Version   int64        `json:"version"`
	URL       *string      `json:"url,omitempty"`
	Enabled   *bool        `json:"enabled,omitempty"`
	ExpiresAt NullableTime `json:"expiresAt,omitempty"`
}

func (u *UpdateInput) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, nonNullKey := range []string{"version", "url", "enabled"} {
		if val, ok := raw[nonNullKey]; ok && bytes.Equal(bytes.TrimSpace(val), []byte("null")) {
			return fmt.Errorf("%s cannot be null", nonNullKey)
		}
	}
	type plain UpdateInput
	var target plain
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&target); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON values")
	}
	*u = UpdateInput(target)
	return nil
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
	if strings.Contains(u.Hostname(), ":") {
		if addr, err := netip.ParseAddr(u.Hostname()); err != nil || !addr.Is6() {
			return fmt.Errorf("%w: destination IPv6 address is invalid", domain.ErrInvalid)
		}
	} else {
		host := strings.TrimSuffix(u.Hostname(), ".")
		labels := strings.Split(host, ".")
		if slices.Contains(labels, "") {
			return fmt.Errorf("%w: destination hostname is invalid", domain.ErrInvalid)
		}
		last := labels[len(labels)-1]
		if isNumericLabel(last) {
			if len(labels) != 4 {
				return fmt.Errorf("%w: destination IPv4 address is invalid", domain.ErrInvalid)
			}
			if addr, err := netip.ParseAddr(host); err != nil || !addr.Is4() {
				return fmt.Errorf("%w: destination IPv4 address is invalid", domain.ErrInvalid)
			}
		}
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return fmt.Errorf("%w: destination port must be between 0 and 65535", domain.ErrInvalid)
		}
	}
	return nil
}

func isNumericLabel(label string) bool {
	if strings.Trim(label, "0123456789") == "" {
		return true
	}
	if strings.HasPrefix(label, "0x") || strings.HasPrefix(label, "0X") {
		return strings.Trim(label[2:], "0123456789abcdefABCDEF") == ""
	}
	return false
}

func (s *Service) Create(ctx context.Context, actor string, input CreateInput) (domain.Link, error) {
	if !ValidActor(actor) {
		return domain.Link{}, domain.ErrUnauthorized
	}
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
	link := domain.Link{
		OwnerID:   actor,
		URL:       input.URL,
		Enabled:   true,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: input.ExpiresAt,
	}
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

func (s *Service) Get(ctx context.Context, actor, code string) (domain.Link, error) {
	if !ValidActor(actor) {
		return domain.Link{}, domain.ErrUnauthorized
	}
	if !ValidCode(code) {
		return domain.Link{}, fmt.Errorf("%w: malformed short code", domain.ErrInvalid)
	}
	link, err := s.repo.Get(ctx, code)
	if err != nil {
		return domain.Link{}, err
	}
	if link.Deleted || link.OwnerID == "" || link.OwnerID != actor {
		return domain.Link{}, domain.ErrNotFound
	}
	if link.Version < 1 {
		link.Version = 1
	}
	return link, nil
}

func (s *Service) Resolve(ctx context.Context, code string) (domain.Link, error) {
	if !ValidCode(code) {
		return domain.Link{}, fmt.Errorf("%w: malformed short code", domain.ErrInvalid)
	}
	link, err := s.repo.Get(ctx, code)
	if err != nil {
		return domain.Link{}, err
	}
	if link.Status(s.now()) != "active" || link.URL == "" {
		return domain.Link{}, domain.ErrGone
	}
	return link, nil
}

func (s *Service) List(ctx context.Context, actor, cursor string) (domain.Page, error) {
	if !ValidActor(actor) {
		return domain.Page{}, domain.ErrUnauthorized
	}
	return s.repo.List(ctx, actor, cursor)
}

func (s *Service) Update(ctx context.Context, actor, code string, input UpdateInput) (domain.Link, error) {
	if !ValidActor(actor) {
		return domain.Link{}, domain.ErrUnauthorized
	}
	if !ValidCode(code) {
		return domain.Link{}, fmt.Errorf("%w: malformed short code", domain.ErrInvalid)
	}
	if input.Version < 1 {
		return domain.Link{}, fmt.Errorf("%w: version must be a positive integer", domain.ErrInvalid)
	}
	if input.URL == nil && input.Enabled == nil && !input.ExpiresAt.Set {
		return domain.Link{}, fmt.Errorf("%w: at least one field (url, enabled, or expiresAt) must be provided", domain.ErrInvalid)
	}
	if input.URL != nil {
		if err := validateURL(*input.URL); err != nil {
			return domain.Link{}, err
		}
	}
	now := s.now().UTC()
	var expiresAt *time.Time
	if input.ExpiresAt.Set && input.ExpiresAt.Value != nil {
		expiry := input.ExpiresAt.Value.UTC()
		if !expiry.After(now) || expiry.Year() > 9999 {
			return domain.Link{}, fmt.Errorf("%w: expiration must be in the future", domain.ErrInvalid)
		}
		expiresAt = &expiry
	}
	return s.repo.Update(ctx, actor, code, domain.LinkPatch{
		Version:      input.Version,
		URL:          input.URL,
		Enabled:      input.Enabled,
		ExpiresAtSet: input.ExpiresAt.Set,
		ExpiresAt:    expiresAt,
		UpdatedAt:    now,
	})
}

func (s *Service) Delete(ctx context.Context, actor, code string, version int64) error {
	if !ValidActor(actor) {
		return domain.ErrUnauthorized
	}
	if !ValidCode(code) {
		return fmt.Errorf("%w: malformed short code", domain.ErrInvalid)
	}
	if version < 0 {
		return fmt.Errorf("%w: version must not be negative", domain.ErrInvalid)
	}
	return s.repo.Delete(ctx, actor, code, version, s.now().UTC())
}
