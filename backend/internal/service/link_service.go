package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	return &LinkService{repo: repo, publisher: publisher, generate: func() (string, error) { return domain.GenerateShortCode(domain.ShortCodeLength) }, now: time.Now}
}

func (s *LinkService) Create(ctx context.Context, owner, rawURL string) (domain.Link, error) {
	target, err := domain.ValidateTargetURL(rawURL)
	if err != nil || !domain.IsSafeTargetURL(target) {
		return domain.Link{}, ErrInvalidURL
	}
	link := domain.Link{OwnerID: owner, Target: target, CreatedAt: domain.FormatCreatedAt(s.now()), Status: "active"}
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
	if !domain.IsValidShortCode(code) {
		return "", repository.ErrNotFound
	}
	link, err := s.repo.GetLink(ctx, code)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(link.Status, "rejected") || link.Target == "" {
		return "", repository.ErrNotFound
	}
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, domain.ClickEvent{Code: code, At: domain.FormatCreatedAt(s.now())}); err != nil {
			return link.Target, fmt.Errorf("%w: %w", ErrPublish, err)
		}
	}
	return link.Target, nil
}

func (s *LinkService) List(ctx context.Context, owner string, limit int, cursor string) (domain.LinkPage, error) {
	return s.repo.ListUserLinks(ctx, owner, limit, cursor)
}

func (s *LinkService) Delete(ctx context.Context, code, owner string) error {
	if !domain.IsValidShortCode(code) {
		return repository.ErrNotFound
	}
	return s.repo.DeleteUserLink(ctx, code, owner)
}
