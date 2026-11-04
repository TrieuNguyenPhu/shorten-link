package service

import (
	"context"
	"errors"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
)

// CheckSafety preserves the reference substring policy; it does not fetch URLs.
func CheckSafety(ctx context.Context, in domain.ModerationInput) (domain.ModerationCheck, error) {
	if err := ctx.Err(); err != nil {
		return domain.ModerationCheck{}, err
	}
	if strings.TrimSpace(in.Code) == "" || strings.TrimSpace(in.Target) == "" {
		return domain.ModerationCheck{}, errors.New("code and target are required")
	}
	out := domain.ModerationCheck{Code: in.Code, Target: in.Target, Safe: true}
	lower := strings.ToLower(in.Target)
	for _, keyword := range []string{"malware", "phishing", "evil"} {
		if strings.Contains(lower, keyword) {
			out.Safe = false
			break
		}
	}
	return out, nil
}
