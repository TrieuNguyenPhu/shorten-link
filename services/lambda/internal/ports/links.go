package ports

import (
	"context"
	"time"

	"shortlink/internal/domain"
)

type LinkRepository interface {
	Create(ctx context.Context, link domain.Link) error
	Get(ctx context.Context, code string) (domain.Link, error)
	List(ctx context.Context, ownerID, cursor string) (domain.Page, error)
	Update(ctx context.Context, ownerID, code string, patch domain.LinkPatch) (domain.Link, error)
	Delete(ctx context.Context, ownerID, code string, version int64, deletedAt time.Time) error
}
