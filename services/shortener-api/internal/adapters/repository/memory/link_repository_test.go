package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

func TestLinkRepositoryCreateGetAndConflict(t *testing.T) {
	repository := NewLinkRepository()
	expiration := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	link := domain.Link{
		Code:      "abc1234",
		TargetURL: "https://example.com",
		Enabled:   true,
		CreatedAt: time.Date(2026, time.July, 26, 0, 0, 0, 0, time.UTC),
		ExpiresAt: &expiration,
	}

	if err := repository.Create(context.Background(), link); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := repository.Create(context.Background(), link); !errors.Is(err, domain.ErrCodeAlreadyExists) {
		t.Fatalf("second Create() error = %v, want ErrCodeAlreadyExists", err)
	}

	got, err := repository.GetByCode(context.Background(), link.Code)
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.Code != link.Code || got.TargetURL != link.TargetURL || got.ExpiresAt == link.ExpiresAt {
		t.Fatalf("GetByCode() returned an invalid or aliased copy: %#v", got)
	}
	*got.ExpiresAt = got.ExpiresAt.Add(time.Hour)
	again, err := repository.GetByCode(context.Background(), link.Code)
	if err != nil {
		t.Fatalf("second GetByCode() error = %v", err)
	}
	if !again.ExpiresAt.Equal(expiration) {
		t.Fatal("mutating a returned link changed repository state")
	}
}

func TestLinkRepositoryHonorsContextAndMissingCode(t *testing.T) {
	repository := NewLinkRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repository.Create(ctx, domain.Link{Code: "abc1234"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if _, err := repository.GetByCode(ctx, "abc1234"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByCode() error = %v, want context.Canceled", err)
	}
	if _, err := repository.GetByCode(context.Background(), "missing"); !errors.Is(err, domain.ErrLinkNotFound) {
		t.Fatalf("missing GetByCode() error = %v, want ErrLinkNotFound", err)
	}
}
