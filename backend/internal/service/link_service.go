package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
)

var (
	ErrPublish       = errors.New("click publication failed")
	ErrInvalidURL    = errors.New("invalid URL")
	ErrCodeExhausted = errors.New("could not allocate a unique short code")
)

type ClickPublisher interface {
	Publish(context.Context, domain.ClickEvent) error
}

type LinkService struct {
	repo      repository.LinkRepository
	publisher ClickPublisher
	generate  func() (string, error)
	now       func() time.Time
}

func New(repo repository.LinkRepository, publisher ClickPublisher) *LinkService {
	return &LinkService{repo: repo, publisher: publisher, generate: func() (string, error) { return domain.GenerateShortCode(7) }, now: time.Now}
}

func (s *LinkService) Create(ctx context.Context, owner, rawURL string) (domain.Link, error) {
	target, err := domain.NormalizeURL(rawURL)
	if err != nil {
		return domain.Link{}, ErrInvalidURL
	}
	link := domain.Link{OwnerID: owner, Target: target, CreatedAt: domain.FormatCreatedAt(s.now())}
	for range 5 {
		link.Code, err = s.generate()
		if err != nil {
			return domain.Link{}, err
		}
		err = s.repo.CreateLink(ctx, link)
		if err == nil {
			return link, nil
		}
		if !errors.Is(err, repository.ErrCollision) {
			return domain.Link{}, err
		}
	}
	return domain.Link{}, ErrCodeExhausted
}

func (s *LinkService) Resolve(ctx context.Context, code string) (string, error) {
	link, err := s.repo.GetLink(ctx, code)
	if err != nil {
		return "", err
	}
	if s.publisher == nil {
		return "", errors.New("click publisher is required")
	}
	if err := s.publisher.Publish(ctx, domain.ClickEvent{Code: code, At: domain.FormatCreatedAt(s.now())}); err != nil {
		return "", fmt.Errorf("%w: %w", ErrPublish, err)
	}
	return link.Target, nil
}

func (s *LinkService) List(ctx context.Context, owner string, limit int, cursor string) (domain.LinkPage, error) {
	page, err := s.repo.ListUserLinks(ctx, owner, limit, cursor)
	if err != nil {
		return domain.LinkPage{}, err
	}
	// Leave room for JSON escaping in the Lambda response envelope (at most 2x),
	// cursor fields and headers, below the synchronous 6 MiB invocation limit.
	const pageBudget = 5 * 1024 * 1024 / 2
	size := 4096
	for i, link := range page.Links {
		encoded, err := json.Marshal(link)
		if err != nil {
			return domain.LinkPage{}, err
		}
		size += len(encoded) + 1
		if size > pageBudget {
			if i == 0 {
				return domain.LinkPage{}, errors.New("stored link exceeds response budget")
			}
			page.NextCursor, err = repository.CursorForLink(owner, page.Links[i-1])
			if err != nil {
				return domain.LinkPage{}, err
			}
			page.Links = page.Links[:i]
			break
		}
	}
	return page, nil
}

func (s *LinkService) Delete(ctx context.Context, code, owner string) error {
	return s.repo.DeleteUserLink(ctx, code, owner)
}
