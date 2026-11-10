package events

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type sfnStub struct{ input *sfn.StartExecutionInput }

func (s *sfnStub) StartExecution(_ context.Context, in *sfn.StartExecutionInput, _ ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	s.input = in
	return &sfn.StartExecutionOutput{}, nil
}

func TestModerationStarterStartsStateMachineWithLinkInput(t *testing.T) {
	stub := &sfnStub{}
	starter := NewModerationStarter(stub, "arn:aws:states:ap-southeast-1:111111111111:stateMachine:moderation")
	if err := starter.Start(context.Background(), domain.ModerationInput{Code: "Abc1234", Target: "https://example.org"}); err != nil {
		t.Fatal(err)
	}
	if aws.ToString(stub.input.StateMachineArn) == "" {
		t.Fatal("missing state machine ARN")
	}
	var got domain.ModerationInput
	if err := json.Unmarshal([]byte(aws.ToString(stub.input.Input)), &got); err != nil || got.Code != "Abc1234" || got.Target != "https://example.org" {
		t.Fatalf("input=%q got=%+v err=%v", aws.ToString(stub.input.Input), got, err)
	}
}
