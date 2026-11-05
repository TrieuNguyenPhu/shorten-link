package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
)

func TestHandlerDoesNotRequireAWSConfiguration(t *testing.T) {
	t.Setenv("TABLE_NAME", "")
	for _, target := range []string{"https://example.org", "https://example.org/evil"} {
		out, err := handle(context.Background(), domain.ModerationInput{Code: "Abc1234", Target: target})
		if err != nil || out.Code != "Abc1234" || out.Target != target || out.Safe != (target == "https://example.org") {
			t.Fatal(out, err)
		}
	}
}

// The ASL is an independent manual workflow. These contract checks prevent
// accidentally switching to a Lambda envelope or dropping the failure branch.
// They do not execute Step Functions; execution history requires AWS E2E.
func TestModerationStateMachineContract(t *testing.T) {
	data, err := os.ReadFile("../../../statemachine/moderation.asl.json")
	if err != nil {
		t.Fatal(err)
	}
	var machine struct {
		StartAt string
		States  map[string]struct {
			Type, Resource, Next, Default, Error string
			End                                  bool
			Retry                                []struct {
				ErrorEquals                  []string
				IntervalSeconds, MaxAttempts int
				BackoffRate                  float64
			}
			Catch []struct {
				ErrorEquals []string
				Next        string
			}
			Choices []struct {
				Variable      string
				BooleanEquals bool
				Next          string
			}
			Parameters map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(data, &machine); err != nil {
		t.Fatal(err)
	}
	check := machine.States["CheckSafety"]
	if machine.StartAt != "CheckSafety" || check.Type != "Task" || check.Resource != "${ModerateCheckArn}" || check.Next != "IsSafe" || len(check.Retry) != 1 || len(check.Catch) != 1 {
		t.Fatal(machine)
	}
	retry := check.Retry[0]
	if !reflect.DeepEqual(retry.ErrorEquals, []string{"Lambda.ServiceException", "Lambda.AWSLambdaException", "Lambda.SdkClientException", "Lambda.TooManyRequestsException"}) || retry.IntervalSeconds != 1 || retry.MaxAttempts != 3 || retry.BackoffRate != 2 || check.Catch[0].Next != "MarkError" || !reflect.DeepEqual(check.Catch[0].ErrorEquals, []string{"States.ALL"}) {
		t.Fatal(check)
	}
	choice := machine.States["IsSafe"]
	if choice.Type != "Choice" || choice.Default != "Reject" || len(choice.Choices) != 1 || choice.Choices[0].Variable != "$.safe" || !choice.Choices[0].BooleanEquals || choice.Choices[0].Next != "Activate" {
		t.Fatal(choice)
	}
	for name, status := range map[string]string{"Activate": "active", "Reject": "rejected"} {
		state := machine.States[name]
		if state.Type != "Task" || state.Resource != "arn:aws:states:::dynamodb:updateItem" || !state.End {
			t.Fatal(state)
		}
		var params struct {
			TableName                 string
			Key                       map[string]map[string]string
			ConditionExpression       string
			UpdateExpression          string
			ExpressionAttributeNames  map[string]string
			ExpressionAttributeValues map[string]map[string]string
		}
		encoded, _ := json.Marshal(state.Parameters)
		if err := json.Unmarshal(encoded, &params); err != nil {
			t.Fatal(err)
		}
		if params.TableName != "${TableName}" || params.Key["PK"]["S.$"] != "States.Format('LINK#{}', $.code)" || params.Key["SK"]["S"] != "META" || params.ConditionExpression != "attribute_exists(PK)" || params.UpdateExpression != "SET #s = :"+status || params.ExpressionAttributeNames["#s"] != "status" || params.ExpressionAttributeValues[":"+status]["S"] != status {
			t.Fatal(params)
		}
	}
	if fail := machine.States["MarkError"]; fail.Type != "Fail" || fail.Error != "ModerationFailed" {
		t.Fatal(fail)
	}
}
