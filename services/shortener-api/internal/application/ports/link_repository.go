package ports

import (
	"context"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

type LinkRepository interface {
	Create(ctx context.Context, link domain.Link) error
	GetByCode(ctx context.Context, code string) (domain.Link, error)
}
