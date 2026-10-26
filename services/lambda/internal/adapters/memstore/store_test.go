package memstore

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"shortlink/internal/domain"
)

func TestMemStoreLifecycleAndIsolation(t *testing.T) {
	ctx := context.Background()
	store := New()
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)

	link := domain.Link{
		Code:      "mem12345",
		OwnerID:   "user-a",
		URL:       "https://example.com",
		Enabled:   true,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Create(ctx, link); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, link); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	disabled := false
	if _, err := store.Update(ctx, "user-b", "mem12345", domain.LinkPatch{Version: 1, Enabled: &disabled, UpdatedAt: now}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-owner update, got %v", err)
	}
	updated, err := store.Update(ctx, "user-a", "mem12345", domain.LinkPatch{Version: 1, Enabled: &disabled, UpdatedAt: now.Add(time.Second)})
	if err != nil || updated.Enabled || updated.Version != 2 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if _, err := store.Update(ctx, "user-a", "mem12345", domain.LinkPatch{Version: 1, Enabled: &disabled, UpdatedAt: now}); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
	if err := store.Delete(ctx, "user-a", "mem12345", 1, now); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict on delete, got %v", err)
	}
	if err := store.Delete(ctx, "user-a", "mem12345", 2, now); err != nil {
		t.Fatal(err)
	}
	tombstone, err := store.Get(ctx, "mem12345")
	if err != nil || !tombstone.Deleted || tombstone.URL != "" || tombstone.OwnerID != "" {
		t.Fatalf("tombstone: %+v %v", tombstone, err)
	}
	if err := store.Create(ctx, link); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict on tombstone reuse, got %v", err)
	}

	for i := range 27 {
		if err := store.Create(ctx, domain.Link{
			Code:      fmt.Sprintf("item%03d", i),
			OwnerID:   "user-a",
			URL:       "https://example.com",
			Enabled:   true,
			Version:   1,
			CreatedAt: now.Add(time.Duration(i+1) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	p1, err := store.List(ctx, "user-a", "")
	if err != nil || len(p1.Links) != 25 || p1.NextCursor == "" {
		t.Fatalf("page 1: %+v %v", p1, err)
	}
	if _, err := store.List(ctx, "user-b", p1.NextCursor); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid on cross-owner cursor, got %v", err)
	}
	p2, err := store.List(ctx, "user-a", p1.NextCursor)
	if err != nil || len(p2.Links) != 2 || p2.NextCursor != "" {
		t.Fatalf("page 2: %+v %v", p2, err)
	}
}
