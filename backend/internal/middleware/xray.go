package middleware

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
)

func InstrumentAWSConfig(cfg *aws.Config) {
	// shortcut: X-Ray SDK is in maintenance; migrate to ADOT when tracing requirements expand.
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)
}
