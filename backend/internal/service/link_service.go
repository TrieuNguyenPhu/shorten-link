package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
)

var (
	ErrPublish                = errors.New("click publication failed")
	ErrInvalidURL             = errors.New("invalid URL")
	ErrInvalidAlias           = errors.New("invalid alias")
	ErrInvalidExpiry          = errors.New("invalid expiry")
	ErrAliasTaken             = errors.New("alias already exists")
	ErrCodeExhausted          = errors.New("could not allocate a unique short code")
	ErrTicketStoreUnavailable = errors.New("realtime ticket store unavailable")
	ErrModerationStart        = errors.New("moderation workflow could not be started")
)

type ClickPublisher interface {
	Publish(context.Context, domain.ClickEvent) error
}

type RealtimeTicketStore interface {
	PutRealtimeTicket(context.Context, domain.RealtimeTicket) error
}

type ModerationStarter interface {
	Start(context.Context, domain.ModerationInput) error
}

type LinkService struct {
	repo            repository.LinkRepository
	publisher       ClickPublisher
	generate        func() (string, error)
	generateTicket  func() (string, error)
	generateClickID func() (string, error)
	moderator       ModerationStarter
	now             func() time.Time
}

func (s *LinkService) SetModerationStarter(starter ModerationStarter) { s.moderator = starter }

func New(repo repository.LinkRepository, publisher ClickPublisher) *LinkService {
	return &LinkService{repo: repo, publisher: publisher, generate: func() (string, error) { return domain.GenerateShortCode(domain.ShortCodeLength) }, generateTicket: func() (string, error) {
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		return base64.RawURLEncoding.EncodeToString(buf), nil
	}, generateClickID: func() (string, error) {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		return base64.RawURLEncoding.EncodeToString(buf), nil
	}, now: time.Now}
}

func (s *LinkService) IssueRealtimeTicket(ctx context.Context, owner, code string) (domain.RealtimeTicket, error) {
	if !domain.IsValidLinkCode(code) {
		return domain.RealtimeTicket{}, repository.ErrNotFound
	}
	if _, ok := s.repo.(RealtimeTicketStore); !ok {
		return domain.RealtimeTicket{}, ErrTicketStoreUnavailable
	}
	link, err := s.repo.GetLink(ctx, code)
	if err != nil || link.OwnerID != owner {
		return domain.RealtimeTicket{}, repository.ErrNotFound
	}
	return s.issueRealtimeTicket(ctx, owner, link)
}

func (s *LinkService) IssueRealtimeTicketForLink(ctx context.Context, owner string, link domain.Link) (domain.RealtimeTicket, error) {
	if link.OwnerID != owner || !domain.IsValidLinkCode(link.Code) {
		return domain.RealtimeTicket{}, repository.ErrNotFound
	}
	if _, ok := s.repo.(RealtimeTicketStore); !ok {
		return domain.RealtimeTicket{}, ErrTicketStoreUnavailable
	}
	return s.issueRealtimeTicket(ctx, owner, link)
}

func (s *LinkService) issueRealtimeTicket(ctx context.Context, owner string, link domain.Link) (domain.RealtimeTicket, error) {
	token, err := s.generateTicket()
	if err != nil {
		return domain.RealtimeTicket{}, err
	}
	store, ok := s.repo.(RealtimeTicketStore)
	if !ok {
		return domain.RealtimeTicket{}, ErrTicketStoreUnavailable
	}
	ticket := domain.RealtimeTicket{Ticket: token, OwnerID: owner, Code: link.Code, TTL: s.now().Unix() + 120}
	if err := store.PutRealtimeTicket(ctx, ticket); err != nil {
		return domain.RealtimeTicket{}, err
	}
	return ticket, nil
}

func (s *LinkService) Create(ctx context.Context, owner, rawURL, rawAlias string, expiryDays *int) (domain.Link, error) {
	alias, aliasErr := domain.ValidateAlias(rawAlias)
	if aliasErr != nil {
		return domain.Link{}, ErrInvalidAlias
	}
	if expiryDays != nil {
		if err := domain.ValidateExpiryDays(*expiryDays); err != nil {
			return domain.Link{}, ErrInvalidExpiry
		}
	}
	target, err := domain.ValidateTargetURL(rawURL)
	if err != nil || !domain.IsSafeTargetURL(target) {
		return domain.Link{}, ErrInvalidURL
	}
	createdAt := s.now()
	link := domain.Link{OwnerID: owner, Target: target, CreatedAt: domain.FormatCreatedAt(createdAt), Status: "pending"}
	if expiryDays != nil {
		link.ExpiresAt = createdAt.Add(time.Duration(*expiryDays) * 24 * time.Hour).Unix()
	}
	if alias != "" {
		link.Code = alias
		err = s.repo.CreateLink(ctx, link)
		if err == nil {
			return s.startModeration(ctx, link)
		}
		if errors.Is(err, repository.ErrCollision) {
			return domain.Link{}, ErrAliasTaken
		}
		return domain.Link{}, err
	}
	for range 5 {
		link.Code, err = s.generate()
		if err != nil {
			return domain.Link{}, err
		}
		err = s.repo.CreateLink(ctx, link)
		if err == nil {
			return s.startModeration(ctx, link)
		}
		if !errors.Is(err, repository.ErrCollision) {
			return domain.Link{}, err
		}
	}
	return domain.Link{}, ErrCodeExhausted
}

func (s *LinkService) startModeration(ctx context.Context, link domain.Link) (domain.Link, error) {
	if s.moderator != nil {
		if startErr := s.moderator.Start(ctx, domain.ModerationInput{Code: link.Code, Target: link.Target}); startErr != nil {
			return link, fmt.Errorf("%w: %v", ErrModerationStart, startErr)
		}
	}
	return link, nil
}

func (s *LinkService) Resolve(ctx context.Context, code string) (string, error) {
	if !domain.IsValidLinkCode(code) {
		return "", repository.ErrNotFound
	}
	link, err := s.repo.GetLink(ctx, code)
	if err != nil {
		return "", err
	}
	if link.ExpiresAt > 0 && s.now().Unix() >= link.ExpiresAt {
		return "", repository.ErrNotFound
	}
	if !strings.EqualFold(link.Status, "active") || link.Target == "" {
		return "", repository.ErrNotFound
	}
	if s.publisher != nil {
		clickID, idErr := s.generateClickID()
		if idErr != nil {
			return link.Target, fmt.Errorf("%w: %v", ErrPublish, idErr)
		}
		if err := s.publisher.Publish(ctx, domain.ClickEvent{Code: code, At: domain.FormatCreatedAt(s.now()), ClickID: clickID}); err != nil {
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
