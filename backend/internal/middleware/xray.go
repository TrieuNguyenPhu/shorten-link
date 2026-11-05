package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
)

func InstrumentAWSConfig(cfg *aws.Config) {
	// shortcut: X-Ray SDK is in maintenance; migrate to ADOT when tracing requirements expand.
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)
}

func LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	sdk, err := config.LoadDefaultConfig(ctx, config.WithRetryer(func() aws.Retryer {
		return retry.NewStandard(func(o *retry.StandardOptions) { o.MaxAttempts = 2; o.MaxBackoff = 100 * time.Millisecond })
	}))
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	InstrumentAWSConfig(&sdk)
	return sdk, nil
}
