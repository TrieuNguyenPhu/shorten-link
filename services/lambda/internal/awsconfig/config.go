// Package awsconfig bootstraps only deployed Lambda entrypoints.
package awsconfig

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"shortlink/internal/adapters/dynamostore"
	"shortlink/internal/adapters/lambdaapi"
	"shortlink/internal/application"
)

type Config struct {
	Region, Table      string
	key, secret, token string
}

func Load() (Config, error) {
	c := Config{Region: os.Getenv("AWS_REGION"), Table: os.Getenv("DYNAMODB_TABLE"), key: os.Getenv("AWS_ACCESS_KEY_ID"), secret: os.Getenv("AWS_SECRET_ACCESS_KEY"), token: os.Getenv("AWS_SESSION_TOKEN")}
	if !regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d+$`).MatchString(c.Region) || !regexp.MustCompile(`^[A-Za-z0-9_.-]{3,255}$`).MatchString(c.Table) {
		return c, fmt.Errorf("AWS_REGION and a valid DYNAMODB_TABLE are required")
	}
	if c.key == "" || c.secret == "" || c.token == "" {
		return c, fmt.Errorf("Lambda execution-role credentials are required")
	}
	return c, nil
}

func (c Config) Client() *dynamodb.Client {
	// Lambda injects temporary role credentials; do not consult local profiles, IMDS or endpoint overrides.
	return dynamodb.NewFromConfig(aws.Config{Region: c.Region, Credentials: credentials.NewStaticCredentialsProvider(c.key, c.secret, c.token), RetryMaxAttempts: 2, HTTPClient: &http.Client{Timeout: 5 * time.Second}})
}

func Handler() (*lambdaapi.Handler, error) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return lambdaapi.New(application.New(dynamostore.New(cfg.Client(), cfg.Table))), nil
}
