package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
)

type repoStub struct {
	create func(domain.Link) error
	get    func(string) (domain.Link, error)
	list   func(string, int, string) (domain.LinkPage, error)
	delete func(string, string) error
}

func (r repoStub) CreateLink(_ context.Context, l domain.Link) error { return r.create(l) }

func (r repoStub) GetLink(_ context.Context, c string) (domain.Link, error) { return r.get(c) }

func (r repoStub) ListUserLinks(_ context.Context, u string, limit int, cursor string) (domain.LinkPage, error) {
	return r.list(u, limit, cursor)
}

func (r repoStub) DeleteUserLink(_ context.Context, c, u string) error { return r.delete(c, u) }

type publisherStub func(context.Context, domain.ClickEvent) error

func (p publisherStub) Publish(ctx context.Context, e domain.ClickEvent) error { return p(ctx, e) }

func TestCreateRetriesOnlyCollisionsFiveTotalWrites(t *testing.T) {
	for _, tc := range []struct {
		name       string
		collisions int
		failure    error
		wantCalls  int
		wantErr    bool
	}{{"success", 0, nil, 1, false}, {"last attempt", 4, nil, 5, false}, {"exhausted", 5, nil, 5, true}, {"storage error", 0, errors.New("unavailable"), 1, true}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			generated := 0
			s := New(repoStub{create: func(l domain.Link) error {
				calls++
				if l.OwnerID != "alice" || l.Target != "https://example.org" || l.CreatedAt != "2026-11-03T17:00:00.001Z" {
					t.Fatal(l)
				}
				if tc.failure != nil {
					return tc.failure
				}
				if calls <= tc.collisions {
					return repository.ErrCollision
				}
				return nil
			}}, nil)
			s.now = func() time.Time { return time.Date(2026, 11, 4, 0, 0, 0, 1000000, time.FixedZone("UTC+7", 7*3600)) }
			s.generate = func() (string, error) { generated++; return "Abc1234", nil }
			link, err := s.Create(context.Background(), "alice", "https://example.org")
			if (err != nil) != tc.wantErr || calls != tc.wantCalls || generated != calls {
				t.Fatalf("%+v %v calls=%d generated=%d", link, err, calls, generated)
			}
		})
	}
}

func TestCreateRejectsInvalidURLAndGeneratorError(t *testing.T) {
	s := New(repoStub{}, nil)
	if _, err := s.Create(context.Background(), "alice", "javascript:alert(1)"); !errors.Is(err, ErrInvalidURL) {
		t.Fatal(err)
	}
	s.generate = func() (string, error) { return "", errors.New("entropy unavailable") }
	if _, err := s.Create(context.Background(), "alice", "https://example.org"); err == nil {
		t.Fatal("expected entropy error")
	}
}

func TestResolveRequiresPublishAcknowledgement(t *testing.T) {
	for _, fail := range []bool{false, true} {
		published := false
		s := New(repoStub{get: func(c string) (domain.Link, error) {
			return domain.Link{Code: c, Target: "https://example.org", Status: "REJECTED"}, nil
		}}, publisherStub(func(_ context.Context, e domain.ClickEvent) error {
			published = true
			if e.Code != "Abc1234" || e.At == "" {
				t.Fatal(e)
			}
			if fail {
				return errors.New("denied")
			}
			return nil
		}))
		target, err := s.Resolve(context.Background(), "Abc1234")
		if !published || (err != nil) != fail || (!fail && target != "https://example.org") {
			t.Fatal(target, err)
		}
	}
	s := New(repoStub{get: func(string) (domain.Link, error) { return domain.Link{}, repository.ErrNotFound }}, nil)
	if _, err := s.Resolve(context.Background(), "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatal(err)
	}
}
