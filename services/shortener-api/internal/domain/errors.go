package domain

import "errors"

var (
	ErrLinkNotFound            = errors.New("link not found")
	ErrLinkExpired             = errors.New("link expired")
	ErrLinkDisabled            = errors.New("link disabled")
	ErrCodeAlreadyExists       = errors.New("short code already exists")
	ErrCodeGenerationExhausted = errors.New("could not allocate a unique short code")
	ErrInvalidURL              = errors.New("invalid URL")
	ErrInvalidAlias            = errors.New("invalid custom alias")
	ErrReservedAlias           = errors.New("custom alias is reserved")
	ErrInvalidExpiration       = errors.New("expiration must be between 1 and 365 days")
)
