package memstore

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"shortlink/internal/application"
	"shortlink/internal/domain"
)

const keyTime = "2006-01-02T15:04:05.000000000Z"

type Store struct {
	mu    sync.RWMutex
	items map[string]domain.Link
}

func New() *Store {
	return &Store{items: make(map[string]domain.Link)}
}

func (s *Store) Create(_ context.Context, link domain.Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[link.Code]; exists {
		return domain.ErrConflict
	}
	if link.Version < 1 {
		link.Version = 1
	}
	if link.UpdatedAt.IsZero() {
		link.UpdatedAt = link.CreatedAt
	}
	s.items[link.Code] = link
	return nil
}

func (s *Store) Get(_ context.Context, code string) (domain.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.items[code]
	if !ok {
		return domain.Link{}, domain.ErrNotFound
	}
	return link, nil
}

func (s *Store) Update(_ context.Context, ownerID, code string, patch domain.LinkPatch) (domain.Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.items[code]
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
	s.items[code] = link
	return link, nil
}

func (s *Store) Delete(_ context.Context, ownerID, code string, version int64, deletedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.items[code]
	if !ok || link.Deleted || link.OwnerID == "" || link.OwnerID != ownerID {
		return domain.ErrNotFound
	}
	if version > 0 && link.Version != version {
		return domain.ErrVersionConflict
	}
	s.items[code] = domain.Link{
		Code:      code,
		Deleted:   true,
		Version:   link.Version + 1,
		CreatedAt: link.CreatedAt,
		UpdatedAt: deletedAt,
	}
	return nil
}

type cursorData struct {
	OwnerID   string `json:"ownerId"`
	Code      string `json:"code"`
	CreatedAt string `json:"createdAt"`
}

func sortKey(l domain.Link) string {
	return l.CreatedAt.UTC().Format(keyTime) + "#" + l.Code
}

func (s *Store) List(_ context.Context, ownerID, cursor string) (domain.Page, error) {
	var startKey string
	if cursor != "" {
		var data cursorData
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if len(cursor) > 512 || err != nil || json.Unmarshal(decoded, &data) != nil || !application.ValidCode(data.Code) || data.OwnerID == "" || data.OwnerID != ownerID {
			return domain.Page{}, fmt.Errorf("%w: malformed cursor", domain.ErrInvalid)
		}
		if _, err := time.Parse(keyTime, data.CreatedAt); err != nil {
			return domain.Page{}, fmt.Errorf("%w: malformed cursor", domain.ErrInvalid)
		}
		startKey = data.CreatedAt + "#" + data.Code
	}
	s.mu.RLock()
	var owned []domain.Link
	for _, link := range s.items {
		if !link.Deleted && link.OwnerID == ownerID {
			owned = append(owned, link)
		}
	}
	s.mu.RUnlock()

	slices.SortFunc(owned, func(a, b domain.Link) int {
		return cmp.Compare(sortKey(b), sortKey(a))
	})

	if startKey != "" {
		idx := slices.IndexFunc(owned, func(l domain.Link) bool {
			return sortKey(l) < startKey
		})
		if idx == -1 {
			owned = nil
		} else {
			owned = owned[idx:]
		}
	}

	const limit = 25
	var nextCursor string
	if len(owned) > limit {
		last := owned[limit-1]
		owned = owned[:limit]
		encoded, err := json.Marshal(cursorData{
			OwnerID:   ownerID,
			Code:      last.Code,
			CreatedAt: last.CreatedAt.UTC().Format(keyTime),
		})
		if err != nil {
			return domain.Page{}, err
		}
		nextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	if owned == nil {
		owned = []domain.Link{}
	}
	return domain.Page{Links: owned, NextCursor: nextCursor}, nil
}
