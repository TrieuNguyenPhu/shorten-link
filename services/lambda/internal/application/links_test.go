package application

import (
	"context"
	"encoding/json"
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

func (m *memoryRepository) List(_ context.Context, ownerID, _ string) (domain.Page, error) {
	if m.err != nil {
		return domain.Page{}, m.err
	}
	var owned []domain.Link
	for _, l := range m.links {
		if !l.Deleted && l.OwnerID == ownerID {
			owned = append(owned, l)
		}
	}
	return domain.Page{Links: owned}, nil
}

func (m *memoryRepository) Update(_ context.Context, ownerID, code string, patch domain.LinkPatch) (domain.Link, error) {
	if m.err != nil {
		return domain.Link{}, m.err
	}
	link, ok := m.links[code]
	if !ok || link.Deleted || link.OwnerID == "" || link.OwnerID != ownerID {
		return domain.Link{}, domain.ErrNotFound
	}
	if link.Version != patch.Version {
		return domain.Link{}, domain.ErrVersionConflict
	}
	if patch.URL != nil {
		link.URL = *patch.URL
	}
	if patch.Enabled != nil {
		link.Enabled = *patch.Enabled
	}
	if patch.ExpiresAtSet {
		link.ExpiresAt = patch.ExpiresAt
	}
	link.Version++
	link.UpdatedAt = patch.UpdatedAt
	m.links[code] = link
	return link, nil
}

func (m *memoryRepository) Delete(_ context.Context, ownerID, code string, version int64, deletedAt time.Time) error {
	if m.err != nil {
		return m.err
	}
	link, ok := m.links[code]
	if !ok || link.Deleted || link.OwnerID == "" || link.OwnerID != ownerID {
		return domain.ErrNotFound
	}
	if version > 0 && link.Version != version {
		return domain.ErrVersionConflict
	}
	m.links[code] = domain.Link{
		Code:      code,
		Deleted:   true,
		Version:   link.Version + 1,
		CreatedAt: link.CreatedAt,
		UpdatedAt: deletedAt,
	}
	return nil
}

var now = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

const (
	actorA = "user-a-sub"
	actorB = "user-b-sub"
)

func setup() (*Service, *memoryRepository) {
	repo := &memoryRepository{links: map[string]domain.Link{}}
	svc := New(repo)
	svc.now = func() time.Time { return now }
	return svc, repo
}

func boolPtr(v bool) *bool       { return &v }
func stringPtr(v string) *string { return &v }

func TestCreateValidation(t *testing.T) {
	for _, destination := range []string{"https://example.com/a?q=1#section", "http://localhost:3000", "https://example.com:8443"} {
		svc, _ := setup()
		link, err := svc.Create(context.Background(), actorA, CreateInput{URL: destination})
		if err != nil || link.URL != destination || link.OwnerID != actorA || link.Version != 1 || !link.Enabled || len(link.Code) != 8 || !link.CreatedAt.Equal(now) {
			t.Fatalf("destination %q: link=%+v err=%v", destination, link, err)
		}
	}
	for _, destination := range []string{"", "/relative", "//example.com", "ftp://example.com", "javascript:alert(1)", "https://", "https://user:pass@example.com", "https://example.com/\r\nHeader:x", " https://example.com", "https://example.com/a b", "https://example.com\\@evil.com", "http://example.com:bad"} {
		svc, _ := setup()
		_, err := svc.Create(context.Background(), actorA, CreateInput{URL: destination})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("accepted %q: %v", destination, err)
		}
	}
	svc, _ := setup()
	if _, err := svc.Create(context.Background(), "", CreateInput{URL: "https://example.com"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for empty actor, got %v", err)
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
		_, err := svc.Create(context.Background(), actorA, CreateInput{URL: destination})
		if !errors.Is(err, domain.ErrInvalid) || len(repo.links) != 0 {
			t.Errorf("accepted browser-invalid destination %q: %v", destination, err)
		}
	}
	for _, destination := range []string{"https://example.com:65535/", "http://[::1]:8080/", "https://example.com/a%20b", "http://127.0.0.1:8080/"} {
		svc, _ := setup()
		if _, err := svc.Create(context.Background(), actorA, CreateInput{URL: destination}); err != nil {
			t.Errorf("rejected valid destination %q: %v", destination, err)
		}
	}
}

func TestAliasAndExpirationValidation(t *testing.T) {
	for _, alias := range []string{"a", "ab", "has space", "a/b", "ééé", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"} {
		svc, _ := setup()
		_, err := svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com", Alias: alias})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("accepted alias %q: %v", alias, err)
		}
	}
	for _, expiry := range []time.Time{now.Add(-time.Second), now} {
		svc, _ := setup()
		_, err := svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com", ExpiresAt: &expiry})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("accepted past expiry: %v", err)
		}
	}
	svc, _ := setup()
	expiry := now.Add(time.Hour).In(time.FixedZone("offset", 7*3600))
	link, err := svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com", Alias: "my-link_1", ExpiresAt: &expiry})
	if err != nil || link.Code != "my-link_1" || link.ExpiresAt.Location() != time.UTC {
		t.Fatalf("link=%+v err=%v", link, err)
	}
}

func TestCollisionRetryAndExhaustion(t *testing.T) {
	svc, repo := setup()
	repo.links["existing"] = domain.Link{Code: "existing", OwnerID: actorB}
	calls := 0
	svc.generate = func() (string, error) {
		calls++
		if calls == 1 {
			return "existing", nil
		}
		return "fresh123", nil
	}
	link, err := svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com"})
	if err != nil || link.Code != "fresh123" || calls != 2 {
		t.Fatalf("retry: link=%+v calls=%d err=%v", link, calls, err)
	}
	calls = 0
	svc.generate = func() (string, error) { calls++; return "existing", nil }
	_, err = svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com"})
	if !errors.Is(err, domain.ErrUnavailable) || calls != 5 {
		t.Fatalf("exhaustion: calls=%d err=%v", calls, err)
	}
	calls = 0
	_, err = svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com", Alias: "existing"})
	if !errors.Is(err, domain.ErrConflict) || calls != 0 {
		t.Fatalf("alias collision: calls=%d err=%v", calls, err)
	}
}

func TestResolveUpdateAndDeleteLifecycle(t *testing.T) {
	svc, repo := setup()
	expiry := now.Add(time.Second)
	repo.links["test1234"] = domain.Link{Code: "test1234", OwnerID: actorA, URL: "https://example.com", Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now, ExpiresAt: &expiry}
	if _, err := svc.Resolve(context.Background(), "test1234"); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Update(context.Background(), actorA, "test1234", UpdateInput{Version: 1, Enabled: boolPtr(false)})
	if err != nil || updated.Enabled || updated.Version != 2 {
		t.Fatalf("disable: %+v %v", updated, err)
	}
	if _, err := svc.Resolve(context.Background(), "test1234"); !errors.Is(err, domain.ErrGone) {
		t.Fatalf("disabled: %v", err)
	}
	if _, err := svc.Get(context.Background(), actorA, "test1234"); err != nil {
		t.Fatalf("disabled metadata: %v", err)
	}
	updated, err = svc.Update(context.Background(), actorA, "test1234", UpdateInput{Version: 2, Enabled: boolPtr(true), URL: stringPtr("https://example.com/updated")})
	if err != nil || !updated.Enabled || updated.URL != "https://example.com/updated" || updated.Version != 3 {
		t.Fatalf("re-enable + edit: %+v %v", updated, err)
	}
	// Stale version conflict
	if _, err := svc.Update(context.Background(), actorA, "test1234", UpdateInput{Version: 2, Enabled: boolPtr(false)}); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
	// Clear expiration via NullableTime{Set: true, Value: nil}
	updated, err = svc.Update(context.Background(), actorA, "test1234", UpdateInput{Version: 3, ExpiresAt: NullableTime{Set: true, Value: nil}})
	if err != nil || updated.ExpiresAt != nil || updated.Version != 4 {
		t.Fatalf("clear expiry: %+v %v", updated, err)
	}
	// Set new expiration and test expiry boundary
	newExpiry := now.Add(5 * time.Second)
	updated, err = svc.Update(context.Background(), actorA, "test1234", UpdateInput{Version: 4, ExpiresAt: NullableTime{Set: true, Value: &newExpiry}})
	if err != nil || updated.ExpiresAt == nil || updated.Version != 5 {
		t.Fatalf("set expiry: %+v %v", updated, err)
	}
	svc.now = func() time.Time { return newExpiry }
	if _, err := svc.Resolve(context.Background(), "test1234"); !errors.Is(err, domain.ErrGone) {
		t.Fatalf("expiry boundary: %v", err)
	}
	// Stale version on Delete
	if err := svc.Delete(context.Background(), actorA, "test1234", 1); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict on delete with stale version, got %v", err)
	}
	// Valid Delete -> creates tombstone
	if err := svc.Delete(context.Background(), actorA, "test1234", 5); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Redirect after delete returns 410 Gone
	if _, err := svc.Resolve(context.Background(), "test1234"); !errors.Is(err, domain.ErrGone) {
		t.Fatalf("deleted redirect should return ErrGone, got %v", err)
	}
	// Metadata get/update/delete after delete returns 404 NotFound
	if _, err := svc.Get(context.Background(), actorA, "test1234"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted get should return ErrNotFound, got %v", err)
	}
	// Re-creating the deleted alias is rejected with ErrConflict
	if _, err := svc.Create(context.Background(), actorB, CreateInput{URL: "https://example.com/hijack", Alias: "test1234"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("deleted alias reuse must be rejected with ErrConflict, got %v", err)
	}
}

func TestActorIsolationAndOwnerlessRejection(t *testing.T) {
	svc, repo := setup()
	created, err := svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com/a", Alias: "owned-a"})
	if err != nil {
		t.Fatal(err)
	}
	// Legacy ownerless fixture must not be claimed or read by any user
	repo.links["legacy01"] = domain.Link{Code: "legacy01", OwnerID: "", URL: "https://example.com/legacy", Enabled: true, Version: 1, CreatedAt: now}

	// Actor B cannot Get, Update, or Delete Actor A's link (returns ErrNotFound, never ErrVersionConflict)
	if _, err := svc.Get(context.Background(), actorB, created.Code); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("actor B read actor A link: %v", err)
	}
	if _, err := svc.Update(context.Background(), actorB, created.Code, UpdateInput{Version: 1, Enabled: boolPtr(false)}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("actor B update actor A link: %v", err)
	}
	if _, err := svc.Update(context.Background(), actorB, created.Code, UpdateInput{Version: 99, Enabled: boolPtr(false)}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("actor B stale-version probe on actor A link must still return ErrNotFound: %v", err)
	}
	if err := svc.Delete(context.Background(), actorB, created.Code, 1); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("actor B delete actor A link: %v", err)
	}
	// Neither Actor A nor Actor B can access ownerless legacy link via management API
	for _, actor := range []string{actorA, actorB} {
		if _, err := svc.Get(context.Background(), actor, "legacy01"); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("actor %s accessed ownerless link: %v", actor, err)
		}
	}
	// List for Actor B returns 0 links; List for Actor A returns only "owned-a"
	pageB, err := svc.List(context.Background(), actorB, "")
	if err != nil || len(pageB.Links) != 0 {
		t.Fatalf("actor B list: %+v %v", pageB, err)
	}
	pageA, err := svc.List(context.Background(), actorA, "")
	if err != nil || len(pageA.Links) != 1 || pageA.Links[0].Code != "owned-a" {
		t.Fatalf("actor A list: %+v %v", pageA, err)
	}
}

func TestUpdateInputValidationAndJSON(t *testing.T) {
	svc, _ := setup()
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"null enabled", `{"version":1,"enabled":null}`},
		{"null url", `{"version":1,"url":null}`},
		{"null version", `{"version":null,"enabled":true}`},
		{"unknown field", `{"version":1,"enabled":true,"ownerId":"hijack"}`},
	} {
		var in UpdateInput
		if err := json.Unmarshal([]byte(tc.raw), &in); err == nil {
			t.Errorf("%s: expected JSON unmarshal error, got %+v", tc.name, in)
		}
	}
	if _, err := svc.Update(context.Background(), actorA, "valid123", UpdateInput{Version: 0, Enabled: boolPtr(true)}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid version error, got %v", err)
	}
	if _, err := svc.Update(context.Background(), actorA, "valid123", UpdateInput{Version: 1}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected empty patch error, got %v", err)
	}
}

func TestFailuresPropagate(t *testing.T) {
	svc, repo := setup()
	failure := errors.New("database down")
	repo.err = failure
	_, err := svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com"})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	repo.err = nil
	svc.generate = func() (string, error) { return "", failure }
	_, err = svc.Create(context.Background(), actorA, CreateInput{URL: "https://example.com"})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
