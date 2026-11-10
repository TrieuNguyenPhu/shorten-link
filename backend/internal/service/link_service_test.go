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
	ticket func(domain.RealtimeTicket) error
}

func (r repoStub) CreateLink(_ context.Context, l domain.Link) error { return r.create(l) }

func (r repoStub) GetLink(_ context.Context, c string) (domain.Link, error) { return r.get(c) }

func (r repoStub) ListUserLinks(_ context.Context, u string, limit int, cursor string) (domain.LinkPage, error) {
	return r.list(u, limit, cursor)
}

func (r repoStub) DeleteUserLink(_ context.Context, c, u string) error { return r.delete(c, u) }

func (r repoStub) PutRealtimeTicket(_ context.Context, ticket domain.RealtimeTicket) error {
	if r.ticket == nil {
		return errors.New("ticket store unavailable")
	}
	return r.ticket(ticket)
}

type publisherStub func(context.Context, domain.ClickEvent) error

func (p publisherStub) Publish(ctx context.Context, e domain.ClickEvent) error { return p(ctx, e) }

type moderationStarterStub func(context.Context, domain.ModerationInput) error

func (m moderationStarterStub) Start(ctx context.Context, in domain.ModerationInput) error {
	return m(ctx, in)
}

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

func TestCreateStartsModerationFromPendingState(t *testing.T) {
	started := false
	s := New(repoStub{create: func(link domain.Link) error {
		if link.Status != "pending" {
			t.Fatalf("created link status=%q", link.Status)
		}
		return nil
	}}, nil)
	s.SetModerationStarter(moderationStarterStub(func(_ context.Context, in domain.ModerationInput) error {
		started = true
		if in.Code == "" || in.Target != "https://example.org" {
			t.Fatalf("bad moderation input: %+v", in)
		}
		return nil
	}))
	link, err := s.Create(context.Background(), "alice", "https://example.org")
	if err != nil || link.Status != "pending" || !started {
		t.Fatalf("link=%+v err=%v started=%v", link, err, started)
	}
}

func TestCreateRejectsInvalidURLAndGeneratorError(t *testing.T) {
	s := New(repoStub{}, nil)
	for _, raw := range []string{"javascript:alert(1)", "http://127.0.0.1/admin", "http://localhost:8080/internal", "http://169.254.169.254/latest/meta-data/", "https://phishing.example.org/login"} {
		if _, err := s.Create(context.Background(), "alice", raw); !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("raw=%q err=%v", raw, err)
		}
	}
	s.generate = func() (string, error) { return "", errors.New("entropy unavailable") }
	if _, err := s.Create(context.Background(), "alice", "https://example.org"); err == nil {
		t.Fatal("expected entropy error")
	}
}

func TestResolveBlocksRejectedOrInvalidCodesAndReturnsTargetOnPublishError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		published := false
		s := New(repoStub{get: func(c string) (domain.Link, error) {
			return domain.Link{Code: c, Target: "https://example.org", Status: "active"}, nil
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
		if !published || (err != nil) != fail || target != "https://example.org" {
			t.Fatal(target, err)
		}
	}
	rejected := New(repoStub{get: func(c string) (domain.Link, error) {
		return domain.Link{Code: c, Target: "https://example.org", Status: "rejected"}, nil
	}}, nil)
	if _, err := rejected.Resolve(context.Background(), "Abc1234"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatal("rejected link resolved:", err)
	}
	for _, status := range []string{"pending", "disabled", ""} {
		blocked := New(repoStub{get: func(c string) (domain.Link, error) {
			return domain.Link{Code: c, Target: "https://example.org", Status: status}, nil
		}}, nil)
		if _, err := blocked.Resolve(context.Background(), "Abc1234"); !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("status %q resolved: %v", status, err)
		}
	}
	s := New(repoStub{get: func(string) (domain.Link, error) { t.Fatal("invalid code queried storage"); return domain.Link{}, nil }}, nil)
	if _, err := s.Resolve(context.Background(), "robots.txt"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestIssueRealtimeTicketRequiresOwnedLinkAndStoresShortLivedTicket(t *testing.T) {
	var stored domain.RealtimeTicket
	s := New(repoStub{
		get: func(string) (domain.Link, error) {
			return domain.Link{Code: "Abc1234", OwnerID: "alice", Target: "https://example.org"}, nil
		},
		ticket: func(ticket domain.RealtimeTicket) error { stored = ticket; return nil },
	}, nil)
	s.now = func() time.Time { return time.Unix(100, 0) }
	s.generateTicket = func() (string, error) { return "ticket-1", nil }
	ticket, err := s.IssueRealtimeTicket(context.Background(), "alice", "Abc1234")
	if err != nil || ticket != (domain.RealtimeTicket{Ticket: "ticket-1", OwnerID: "alice", Code: "Abc1234", TTL: 220}) || stored != ticket {
		t.Fatalf("ticket=%+v stored=%+v err=%v", ticket, stored, err)
	}
	if _, err := s.IssueRealtimeTicket(context.Background(), "bob", "Abc1234"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-owner ticket: %v", err)
	}
}
