package ports

import (
	"context"
	"shortlink/internal/domain"
)

type LinkRepository interface {
	Create(context.Context, domain.Link) error
	Get(context.Context, string) (domain.Link, error)
	List(context.Context, string) (domain.Page, error)
	SetEnabled(context.Context, string, bool) (domain.Link, error)
}
