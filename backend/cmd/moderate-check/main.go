package main

import (
	"context"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/lambda"
)

var logger = middleware.NewLogger("moderate-check")

func handle(ctx context.Context, in domain.ModerationInput) (domain.ModerationCheck, error) {
	out, err := service.CheckSafety(ctx, in)
	log := middleware.WithRequestID(ctx, logger)
	if err != nil {
		log.ErrorContext(ctx, "moderation check failed", "error", err)
	} else {
		log.InfoContext(ctx, "moderation checked", "code", out.Code, "safe", out.Safe)
	}
	return out, err
}

func main() { lambda.Start(handle) }
