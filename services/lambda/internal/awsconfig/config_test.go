package awsconfig

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func environment(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{"AWS_REGION": "ap-southeast-1", "DYNAMODB_TABLE": "shortly-dev-links", "AWS_ACCESS_KEY_ID": "role-key", "AWS_SECRET_ACCESS_KEY": "role-secret", "AWS_SESSION_TOKEN": "role-token"} {
		t.Setenv(name, value)
	}
}

func TestLoadRequiresCloudConfiguration(t *testing.T) {
	for _, name := range []string{"AWS_REGION", "DYNAMODB_TABLE", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"} {
		t.Run(name, func(t *testing.T) {
			environment(t)
			t.Setenv(name, "")
			if _, err := Load(); err == nil {
				t.Fatal("missing required cloud configuration accepted")
			}
		})
	}
	for name, value := range map[string]string{"AWS_REGION": "localhost:8000", "DYNAMODB_TABLE": "a/b"} {
		t.Run("invalid "+name, func(t *testing.T) {
			environment(t)
			t.Setenv(name, value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestClientUsesExecutionRoleWithoutLocalOverrides(t *testing.T) {
	environment(t)
	t.Setenv("DYNAMODB_ENDPOINT", "http://127.0.0.1:8000")
	t.Setenv("AWS_ENDPOINT_URL", "http://127.0.0.1:8000")
	t.Setenv("AWS_PROFILE", "must-not-read")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "must-not-read")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	client := cfg.Client()
	options := client.Options()
	creds, err := options.Credentials.Retrieve(context.Background())
	if err != nil || creds.AccessKeyID != "role-key" || creds.SessionToken != "role-token" || options.Region != "ap-southeast-1" || options.BaseEndpoint != nil {
		t.Fatalf("wrong AWS client configuration: %v", err)
	}
	called := false
	_, err = client.GetItem(context.Background(), &dynamodb.GetItemInput{TableName: aws.String(cfg.Table), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "test"}}}, func(o *dynamodb.Options) {
		o.HTTPClient = noNetwork{t, &called}
	})
	if err != nil || !called {
		t.Fatalf("offline client request failed: %v", err)
	}
}

type noNetwork struct {
	t      *testing.T
	called *bool
}

func (n noNetwork) Do(request *http.Request) (*http.Response, error) {
	*n.called = true
	if request.URL.Host != "dynamodb.ap-southeast-1.amazonaws.com" || !strings.Contains(request.Header.Get("Authorization"), "Credential=role-key/") || request.Header.Get("X-Amz-Security-Token") != "role-token" {
		n.t.Fatal("client did not use the regional endpoint and execution-role credentials")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/x-amz-json-1.0"}}, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
}
