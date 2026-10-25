package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalid     = errors.New("invalid input")
	ErrConflict    = errors.New("short code is already in use")
	ErrNotFound    = errors.New("link not found")
	ErrGone        = errors.New("link is disabled or expired")
	ErrUnavailable = errors.New("could not allocate a short code; try again")
)

type Link struct {
	Code      string     `json:"code" dynamodbav:"code"`
	URL       string     `json:"url" dynamodbav:"url"`
	Enabled   bool       `json:"enabled" dynamodbav:"enabled"`
	CreatedAt time.Time  `json:"createdAt" dynamodbav:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty" dynamodbav:"expiresAt,omitempty"`
}

func (l Link) Status(now time.Time) string {
	if !l.Enabled {
		return "disabled"
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return "expired"
	}
	return "active"
}

type Page struct {
	Links      []Link `json:"links"`
	NextCursor string `json:"nextCursor,omitempty"`
}
