package events

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type StepFunctionsAPI interface {
	StartExecution(context.Context, *sfn.StartExecutionInput, ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
}

type ModerationStarter struct {
	client StepFunctionsAPI
	arn    string
}

func NewModerationStarter(client StepFunctionsAPI, arn string) *ModerationStarter {
	return &ModerationStarter{client: client, arn: strings.TrimSpace(arn)}
}

func (s *ModerationStarter) Start(ctx context.Context, in domain.ModerationInput) error {
	if s.client == nil || s.arn == "" {
		return errors.New("moderation state machine ARN is required")
	}
	input, err := json.Marshal(in)
	if err != nil {
		return err
	}
	_, err = s.client.StartExecution(ctx, &sfn.StartExecutionInput{StateMachineArn: aws.String(s.arn), Input: aws.String(string(input))})
	return err
}
