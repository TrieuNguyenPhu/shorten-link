package domain

import "time"

type Link struct {
	Code         string `json:"code" dynamodbav:"code"`
	Target       string `json:"target" dynamodbav:"target"`
	OwnerID      string `json:"-" dynamodbav:"ownerId"`
	CreatedAt    string `json:"createdAt" dynamodbav:"createdAt"`
	Clicks       int64  `json:"clicks" dynamodbav:"clicks"`
	Status       string `json:"status,omitempty" dynamodbav:"status,omitempty"`
	StatusReason string `json:"statusReason,omitempty" dynamodbav:"statusReason,omitempty"`
}

type ClickEvent struct {
	Code    string `json:"code"`
	At      string `json:"at"`
	ClickID string `json:"clickId"`
}

type WSConnection struct {
	ConnectionID string
	OwnerID      string
	Code         string `json:"code" dynamodbav:"code"`
	TTL          int64  `json:"ttl" dynamodbav:"ttl"`
}

type RealtimeTicket struct {
	Ticket  string
	OwnerID string
	Code    string
	TTL     int64
}

type ModerationCheck struct {
	Code   string `json:"code"`
	Target string `json:"target"`
	Safe   bool   `json:"safe"`
}

type ModerationInput struct {
	Code   string `json:"code"`
	Target string `json:"target"`
}

func FormatCreatedAt(at time.Time) string {
	// Fixed milliseconds preserve chronological ordering in GSI1 sort keys.
	return at.UTC().Format("2006-01-02T15:04:05.000Z")
}

// LinkPage is an owner-scoped page. An empty cursor marks the end of the query.
type LinkPage struct {
	Links      []Link
	NextCursor string
}
