package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/application/ports"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

const defaultCollisionAttempts = 5

var (
	aliasPattern    = regexp.MustCompile(`^[a-z0-9-]{4,32}$`)
	reservedAliases = map[string]struct{}{
		"admin":     {},
		"api":       {},
		"app":       {},
		"assets":    {},
		"dashboard": {},
		"docs":      {},
		"healthz":   {},
		"link":      {},
		"login":     {},
		"logout":    {},
		"metrics":   {},
		"privacy":   {},
		"signup":    {},
		"static":    {},
		"terms":     {},
		"www":       {},
	}
)

type CreateLinkInput struct {
	URL           string
	CustomAlias   string
	ExpiresInDays *int
}

type LinkView struct {
	Link   domain.Link
	Status domain.LinkStatus
}

type LinkService struct {
	repository        ports.LinkRepository
	codeGenerator     ports.CodeGenerator
	clock             ports.Clock
	collisionAttempts int
}

func NewLinkService(
	repository ports.LinkRepository,
	codeGenerator ports.CodeGenerator,
	clock ports.Clock,
) *LinkService {
	return &LinkService{
		repository:        repository,
		codeGenerator:     codeGenerator,
		clock:             clock,
		collisionAttempts: defaultCollisionAttempts,
	}
}

func (s *LinkService) Create(ctx context.Context, input CreateLinkInput) (LinkView, error) {
	targetURL, err := normalizeURL(input.URL)
	if err != nil {
		return LinkView{}, err
	}

	var expiresAt *time.Time
	createdAt := s.clock.Now().UTC()
	if input.ExpiresInDays != nil {
		if *input.ExpiresInDays < 1 || *input.ExpiresInDays > 365 {
			return LinkView{}, domain.ErrInvalidExpiration
		}
		expiration := createdAt.Add(time.Duration(*input.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &expiration
	}

	if input.CustomAlias != "" {
		if err := validateAlias(input.CustomAlias); err != nil {
			return LinkView{}, err
		}
		link := newLink(input.CustomAlias, targetURL, createdAt, expiresAt)
		if err := s.repository.Create(ctx, link); err != nil {
			return LinkView{}, fmt.Errorf("create custom link: %w", err)
		}
		return LinkView{Link: link, Status: domain.LinkStatusActive}, nil
	}

	for attempt := 0; attempt < s.collisionAttempts; attempt++ {
		code, err := s.codeGenerator.Generate(ctx)
		if err != nil {
			return LinkView{}, fmt.Errorf("generate short code: %w", err)
		}
		if err := validateGeneratedCode(code); err != nil {
			return LinkView{}, fmt.Errorf("generated invalid short code: %w", err)
		}

		link := newLink(code, targetURL, createdAt, expiresAt)
		err = s.repository.Create(ctx, link)
		if err == nil {
			return LinkView{Link: link, Status: domain.LinkStatusActive}, nil
		}
		if errors.Is(err, domain.ErrCodeAlreadyExists) {
			continue
		}
		return LinkView{}, fmt.Errorf("create generated link: %w", err)
	}

	return LinkView{}, domain.ErrCodeGenerationExhausted
}

func newLink(code, targetURL string, createdAt time.Time, expiresAt *time.Time) domain.Link {
	return domain.Link{
		Code:      code,
		TargetURL: targetURL,
		Enabled:   true,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
}

func normalizeURL(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	if len(normalized) == 0 || len(normalized) > 2048 {
		return "", domain.ErrInvalidURL
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return "", domain.ErrInvalidURL
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", domain.ErrInvalidURL
	}
	return parsed.String(), nil
}

func validateAlias(alias string) error {
	if !aliasPattern.MatchString(alias) {
		return domain.ErrInvalidAlias
	}
	if _, reserved := reservedAliases[alias]; reserved {
		return domain.ErrReservedAlias
	}
	return nil
}

func validateGeneratedCode(code string) error {
	if !aliasPattern.MatchString(code) {
		return domain.ErrInvalidAlias
	}
	return nil
}
