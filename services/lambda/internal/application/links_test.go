package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"shortlink/internal/domain"
)

type memoryRepository struct {
	links map[string]domain.Link
	err   error
}

func (m *memoryRepository) Create(_ context.Context, link domain.Link) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.links[link.Code]; ok {
		return domain.ErrConflict
	}
	m.links[link.Code] = link
	return nil
}
func (m *memoryRepository) Get(_ context.Context, code string) (domain.Link, error) {
	if m.err != nil {
		return domain.Link{}, m.err
	}
	link, ok := m.links[code]
	if !ok {
		return link, domain.ErrNotFound
	}
	return link, nil
}
func (m *memoryRepository) List(_ context.Context, _ string) (domain.Page, error) {
	return domain.Page{}, m.err
}
func (m *memoryRepository) SetEnabled(ctx context.Context, code string, enabled bool) (domain.Link, error) {
	link, err := m.Get(ctx, code)
	if err != nil {
		return link, err
	}
	link.Enabled = enabled
	m.links[code] = link
	return link, nil
}

var now = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

func setup() (*Service, *memoryRepository) {
	repo := &memoryRepository{links: map[string]domain.Link{}}
	svc := New(repo)
	svc.now = func() time.Time { return now }
	return svc, repo
}

func TestCreateValidation(t *testing.T) {
	for _, destination := range []string{"https://example.com/a?q=1#section", "http://localhost:3000", "https://example.com:8443"} {
		svc, _ := setup()
		link, err := svc.Create(context.Background(), CreateInput{URL: destination})
		if err != nil || link.URL != destination || !link.Enabled || len(link.Code) != 8 || !link.CreatedAt.Equal(now) {
			t.Fatalf("destination %q: link=%+v err=%v", destination, link, err)
		}
	}
	for _, destination := range []string{"", "/relative", "//example.com", "ftp://example.com", "javascript:alert(1)", "https://", "https://user:pass@example.com", "https://example.com/\r\nHeader:x", " https://example.com", "https://example.com/a b", "https://example.com\\@evil.com", "http://example.com:bad"} {
		svc, _ := setup()
		_, err := svc.Create(context.Background(), CreateInput{URL: destination})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("accepted %q: %v", destination, err)
		}
	}
}

func TestRejectBrowserInvalidDestination(t *testing.T) {
	for _, destination := range []string{
		"https://example.com:99999/",
		"https://exa<mple.com/",
		"https://exa>mple.com/",
		"https://exa^mple.com/",
		"https://exa|mple.com/",
		"https://256.256.256.256/",
		"https://256.256.256.256./",
		"https://4294967296/",
		"https://example.123/",
		"https://example.123./",
		"https://0x7f000001/",
		"https://example..com/",
		"https://.example.com/",
	} {
		svc, repo := setup()
		_, err := svc.Create(context.Background(), CreateInput{URL: destination})
		if !errors.Is(err, domain.ErrInvalid) || len(repo.links) != 0 {
			t.Errorf("accepted browser-invalid destination %q: %v", destination, err)
		}
	}
	for _, destination := range []string{"https://example.com:65535/", "http://[::1]:8080/", "https://example.com/a%20b", "http://127.0.0.1:8080/"} {
		svc, _ := setup()
		if _, err := svc.Create(context.Background(), CreateInput{URL: destination}); err != nil {
			t.Errorf("rejected valid destination %q: %v", destination, err)
		}
	}
}

func TestAliasAndExpirationValidation(t *testing.T) {
	for _, alias := range []string{"a", "ab", "has space", "a/b", "ééé", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		svc, _ := setup()
		_, err := svc.Create(context.Background(), CreateInput{URL: "https://example.com", Alias: alias})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("accepted alias %q: %v", alias, err)
		}
	}
	for _, expiry := range []time.Time{now.Add(-time.Second), now} {
		svc, _ := setup()
		_, err := svc.Create(context.Background(), CreateInput{URL: "https://example.com", ExpiresAt: &expiry})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("accepted past expiry: %v", err)
		}
	}
	svc, _ := setup()
	expiry := now.Add(time.Hour).In(time.FixedZone("offset", 7*3600))
	link, err := svc.Create(context.Background(), CreateInput{URL: "https://example.com", Alias: "my-link_1", ExpiresAt: &expiry})
	if err != nil || link.Code != "my-link_1" || link.ExpiresAt.Location() != time.UTC {
		t.Fatalf("link=%+v err=%v", link, err)
	}
}

func TestCollisionRetryAndExhaustion(t *testing.T) {
	svc, repo := setup()
	repo.links["existing"] = domain.Link{Code: "existing"}
	calls := 0
	svc.generate = func() (string, error) {
		calls++
		if calls == 1 {
			return "existing", nil
		}
		return "fresh123", nil
	}
	link, err := svc.Create(context.Background(), CreateInput{URL: "https://example.com"})
	if err != nil || link.Code != "fresh123" || calls != 2 {
		t.Fatalf("retry: link=%+v calls=%d err=%v", link, calls, err)
	}
	calls = 0
	svc.generate = func() (string, error) { calls++; return "existing", nil }
	_, err = svc.Create(context.Background(), CreateInput{URL: "https://example.com"})
	if !errors.Is(err, domain.ErrUnavailable) || calls != 5 {
		t.Fatalf("exhaustion: calls=%d err=%v", calls, err)
	}
	calls = 0
	_, err = svc.Create(context.Background(), CreateInput{URL: "https://example.com", Alias: "existing"})
	if !errors.Is(err, domain.ErrConflict) || calls != 0 {
		t.Fatalf("alias collision: calls=%d err=%v", calls, err)
	}
}

func TestResolveAndEnable(t *testing.T) {
	svc, repo := setup()
	expiry := now.Add(time.Second)
	repo.links["test1234"] = domain.Link{Code: "test1234", URL: "https://example.com", Enabled: true, ExpiresAt: &expiry}
	if _, err := svc.Resolve(context.Background(), "test1234"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetEnabled(context.Background(), "test1234", false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve(context.Background(), "test1234"); !errors.Is(err, domain.ErrGone) {
		t.Fatalf("disabled: %v", err)
	}
	if _, err := svc.Get(context.Background(), "test1234"); err != nil {
		t.Fatalf("disabled metadata: %v", err)
	}
	if _, err := svc.SetEnabled(context.Background(), "test1234", true); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return expiry }
	if _, err := svc.Resolve(context.Background(), "test1234"); !errors.Is(err, domain.ErrGone) {
		t.Fatalf("expiry boundary: %v", err)
	}
	if _, err := svc.Resolve(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := svc.Get(context.Background(), "../bad"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid code: %v", err)
	}
}

func TestFailuresPropagate(t *testing.T) {
	svc, repo := setup()
	failure := errors.New("database down")
	repo.err = failure
	_, err := svc.Create(context.Background(), CreateInput{URL: "https://example.com"})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	repo.err = nil
	svc.generate = func() (string, error) { return "", failure }
	_, err = svc.Create(context.Background(), CreateInput{URL: "https://example.com"})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
