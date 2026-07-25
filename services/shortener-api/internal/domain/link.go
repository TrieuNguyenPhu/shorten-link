package domain

import "time"

type LinkStatus string

const (
	LinkStatusActive   LinkStatus = "active"
	LinkStatusExpired  LinkStatus = "expired"
	LinkStatusDisabled LinkStatus = "disabled"
)

type Link struct {
	Code      string
	TargetURL string
	Enabled   bool
	CreatedAt time.Time
	ExpiresAt *time.Time
}

func (l Link) StatusAt(now time.Time) LinkStatus {
	if !l.Enabled {
		return LinkStatusDisabled
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return LinkStatusExpired
	}
	return LinkStatusActive
}

func (l Link) EnsureResolvableAt(now time.Time) error {
	switch l.StatusAt(now) {
	case LinkStatusDisabled:
		return ErrLinkDisabled
	case LinkStatusExpired:
		return ErrLinkExpired
	default:
		return nil
	}
}
