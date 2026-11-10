package service

import (
	"context"
	"errors"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
)

func CheckSafety(ctx context.Context, in domain.ModerationInput) (domain.ModerationCheck, error) {
	if err := ctx.Err(); err != nil {
		return domain.ModerationCheck{}, err
	}
	if !domain.IsValidLinkCode(strings.TrimSpace(in.Code)) || strings.TrimSpace(in.Target) == "" {
		return domain.ModerationCheck{}, errors.New("valid code and target are required")
	}
	return domain.ModerationCheck{
		Code:   in.Code,
		Target: in.Target,
		Safe:   domain.IsSafeTargetURL(in.Target),
	}, nil
}
