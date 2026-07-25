package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceGenerator struct {
	codes []string
	calls int
}

func (g *sequenceGenerator) Generate(context.Context) (string, error) {
	if g.calls >= len(g.codes) {
		return "", errors.New("generator sequence exhausted")
	}
	code := g.codes[g.calls]
	g.calls++
	return code, nil
}

type testRepository struct {
	links map[string]domain.Link
}

func newTestRepository() *testRepository {
	return &testRepository{links: make(map[string]domain.Link)}
}

func (r *testRepository) Create(ctx context.Context, link domain.Link) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.links[link.Code]; exists {
		return domain.ErrCodeAlreadyExists
	}
	r.links[link.Code] = link
	return nil
}

func (r *testRepository) GetByCode(ctx context.Context, code string) (domain.Link, error) {
	if err := ctx.Err(); err != nil {
		return domain.Link{}, err
	}
	link, exists := r.links[code]
	if !exists {
		return domain.Link{}, domain.ErrLinkNotFound
	}
	return link, nil
}

func TestCreateGeneratedLink(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	days := 30
	service := NewLinkService(
		newTestRepository(),
		&sequenceGenerator{codes: []string{"abc1234"}},
		fixedClock{now: now},
	)

	view, err := service.Create(context.Background(), CreateLinkInput{
		URL:           "https://example.com/articles/clean-architecture",
		ExpiresInDays: &days,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if view.Link.Code != "abc1234" {
		t.Fatalf("code = %q, want abc1234", view.Link.Code)
	}
	if view.Link.TargetURL != "https://example.com/articles/clean-architecture" {
		t.Fatalf("target URL = %q", view.Link.TargetURL)
	}
	if view.Status != domain.LinkStatusActive {
		t.Fatalf("status = %q, want active", view.Status)
	}
	wantExpiration := now.Add(30 * 24 * time.Hour)
	if view.Link.ExpiresAt == nil || !view.Link.ExpiresAt.Equal(wantExpiration) {
		t.Fatalf("expires at = %v, want %v", view.Link.ExpiresAt, wantExpiration)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	oneDay := 1
	zeroDays := 0
	tooManyDays := 366
	tests := []struct {
		name    string
		input   CreateLinkInput
		wantErr error
	}{
		{name: "relative URL", input: CreateLinkInput{URL: "/relative"}, wantErr: domain.ErrInvalidURL},
		{name: "unsupported URL scheme", input: CreateLinkInput{URL: "ftp://example.com/file"}, wantErr: domain.ErrInvalidURL},
		{name: "URL user info", input: CreateLinkInput{URL: "https://user:password@example.com"}, wantErr: domain.ErrInvalidURL},
		{name: "URL too long", input: CreateLinkInput{URL: "https://example.com/" + strings.Repeat("a", 2048)}, wantErr: domain.ErrInvalidURL},
		{name: "short alias", input: CreateLinkInput{URL: "https://example.com", CustomAlias: "abc"}, wantErr: domain.ErrInvalidAlias},
		{name: "uppercase alias", input: CreateLinkInput{URL: "https://example.com", CustomAlias: "My-Link"}, wantErr: domain.ErrInvalidAlias},
		{name: "reserved alias", input: CreateLinkInput{URL: "https://example.com", CustomAlias: "admin"}, wantErr: domain.ErrReservedAlias},
		{name: "zero expiration", input: CreateLinkInput{URL: "https://example.com", ExpiresInDays: &zeroDays}, wantErr: domain.ErrInvalidExpiration},
		{name: "excessive expiration", input: CreateLinkInput{URL: "https://example.com", ExpiresInDays: &tooManyDays}, wantErr: domain.ErrInvalidExpiration},
		{name: "valid expiration with bad URL", input: CreateLinkInput{URL: "not-a-url", ExpiresInDays: &oneDay}, wantErr: domain.ErrInvalidURL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewLinkService(
				newTestRepository(),
				&sequenceGenerator{codes: []string{"abc1234"}},
				fixedClock{now: now},
			)
			_, err := service.Create(context.Background(), test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestCreateRetriesGeneratedCodeCollision(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	repository := newTestRepository()
	if err := repository.Create(ctx, domain.Link{
		Code: "taken12", TargetURL: "https://existing.example", Enabled: true, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	generator := &sequenceGenerator{codes: []string{"taken12", "fresh12"}}
	service := NewLinkService(repository, generator, fixedClock{now: now})

	view, err := service.Create(ctx, CreateLinkInput{URL: "https://example.com/new"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if view.Link.Code != "fresh12" {
		t.Fatalf("code = %q, want fresh12", view.Link.Code)
	}
	if generator.calls != 2 {
		t.Fatalf("generator calls = %d, want 2", generator.calls)
	}
}

func TestCreateCustomAliasConflict(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	service := NewLinkService(
		newTestRepository(),
		&sequenceGenerator{codes: []string{"unused1"}},
		fixedClock{now: now},
	)
	input := CreateLinkInput{URL: "https://example.com", CustomAlias: "my-link"}
	if _, err := service.Create(ctx, input); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := service.Create(ctx, input); !errors.Is(err, domain.ErrCodeAlreadyExists) {
		t.Fatalf("second Create() error = %v, want conflict", err)
	}
}

func TestCreateReturnsExhaustedAfterCollisions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	repository := newTestRepository()
	if err := repository.Create(ctx, domain.Link{
		Code: "taken12", TargetURL: "https://existing.example", Enabled: true, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	service := NewLinkService(
		repository,
		&sequenceGenerator{codes: []string{"taken12", "taken12", "taken12", "taken12", "taken12"}},
		fixedClock{now: now},
	)

	_, err := service.Create(ctx, CreateLinkInput{URL: "https://example.com/new"})
	if !errors.Is(err, domain.ErrCodeGenerationExhausted) {
		t.Fatalf("Create() error = %v, want exhausted", err)
	}
}

func TestResolveMapsLinkState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)
	tests := []struct {
		name    string
		link    *domain.Link
		code    string
		wantErr error
	}{
		{name: "active", code: "active1", link: &domain.Link{Code: "active1", TargetURL: "https://example.com", Enabled: true, CreatedAt: now, ExpiresAt: &future}},
		{name: "expired", code: "expired", link: &domain.Link{Code: "expired", TargetURL: "https://example.com", Enabled: true, CreatedAt: now, ExpiresAt: &past}, wantErr: domain.ErrLinkExpired},
		{name: "disabled", code: "disabled", link: &domain.Link{Code: "disabled", TargetURL: "https://example.com", Enabled: false, CreatedAt: now}, wantErr: domain.ErrLinkDisabled},
		{name: "missing", code: "missing", wantErr: domain.ErrLinkNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newTestRepository()
			if test.link != nil {
				if err := repository.Create(ctx, *test.link); err != nil {
					t.Fatalf("seed repository: %v", err)
				}
			}
			service := NewLinkService(repository, &sequenceGenerator{}, fixedClock{now: now})
			link, err := service.Resolve(ctx, test.code)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Resolve() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && link.Code != test.code {
				t.Fatalf("Resolve() code = %q, want %q", link.Code, test.code)
			}
		})
	}
}
