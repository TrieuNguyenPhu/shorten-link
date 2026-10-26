package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalid         = errors.New("invalid input")
	ErrUnauthorized    = errors.New("authentication required")
	ErrForbidden       = errors.New("insufficient scope")
	ErrConflict        = errors.New("short code is already in use")
	ErrVersionConflict = fmt.Errorf("%w: link was modified; refresh and try again", ErrConflict)
	ErrNotFound        = errors.New("link not found")
	ErrGone            = errors.New("link is disabled, expired, or deleted")
	ErrUnavailable     = errors.New("could not allocate a short code; try again")
)

type Link struct {
	Code      string
	OwnerID   string
	URL       string
	Enabled   bool
	Version   int64
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
}

func (l Link) Status(now time.Time) string {
	if l.Deleted {
		return "deleted"
	}
	if !l.Enabled {
		return "disabled"
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return "expired"
	}
	return "active"
}

type Page struct {
	Links      []Link
	NextCursor string
}

type LinkPatch struct {
	Version      int64
	URL          *string
	Enabled      *bool
	ExpiresAtSet bool
	ExpiresAt    *time.Time
	UpdatedAt    time.Time
}
