package middleware

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambdacontext"
)

func NewLogger(serviceName string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", serviceName)
}

func WithRequestID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		return logger.With("requestId", lc.AwsRequestID)
	}
	return logger
}
