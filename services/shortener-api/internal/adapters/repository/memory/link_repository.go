package memory

import (
	"context"
	"sync"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

type LinkRepository struct {
	mu    sync.RWMutex
	links map[string]domain.Link
}

func NewLinkRepository() *LinkRepository {
	return &LinkRepository{links: make(map[string]domain.Link)}
}

func (r *LinkRepository) Create(ctx context.Context, link domain.Link) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.links[link.Code]; exists {
		return domain.ErrCodeAlreadyExists
	}
	r.links[link.Code] = cloneLink(link)
	return nil
}

func (r *LinkRepository) GetByCode(ctx context.Context, code string) (domain.Link, error) {
	if err := ctx.Err(); err != nil {
		return domain.Link{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	link, exists := r.links[code]
	if !exists {
		return domain.Link{}, domain.ErrLinkNotFound
	}
	return cloneLink(link), nil
}

func cloneLink(link domain.Link) domain.Link {
	copy := link
	if link.ExpiresAt != nil {
		expiration := *link.ExpiresAt
		copy.ExpiresAt = &expiration
	}
	return copy
}
