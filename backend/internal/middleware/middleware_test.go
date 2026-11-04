package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

func TestLoggerRequestContext(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original; r.Close(); w.Close() })
	logger := NewLogger("resolve-link")
	ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{AwsRequestID: "request-123"})
	WithRequestID(ctx, logger).InfoContext(ctx, "resolved", "code", "Ab12Cd3")
	w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	var entry map[string]any
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("invalid JSON: %s: %v", data, err)
	}
	if entry["service"] != "resolve-link" || entry["requestId"] != "request-123" || entry["level"] != "INFO" || entry["msg"] != "resolved" || entry["code"] != "Ab12Cd3" {
		t.Fatalf("invalid log: %s", data)
	}
	if WithRequestID(context.Background(), logger) != logger {
		t.Fatal("no-context logger should be usable locally")
	}
}

func TestEmitCountMetric(t *testing.T) {
	for _, metric := range []string{"LinkResolved", "LinkNotFound"} {
		var buf bytes.Buffer
		before := time.Now().UnixMilli()
		if err := EmitCountMetric(&buf, "UrlShortener", "resolve-link", metric, 1); err != nil {
			t.Fatal(err)
		}
		var doc struct {
			AWS struct {
				Timestamp int64 `json:"Timestamp"`
				Metrics   []struct {
					Namespace  string                        `json:"Namespace"`
					Dimensions [][]string                    `json:"Dimensions"`
					Metrics    []struct{ Name, Unit string } `json:"Metrics"`
				} `json:"CloudWatchMetrics"`
			} `json:"_aws"`
			Service string `json:"service"`
		}
		if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Service != "resolve-link" || doc.AWS.Timestamp < before || doc.AWS.Timestamp > time.Now().UnixMilli() || len(doc.AWS.Metrics) != 1 {
			t.Fatalf("invalid EMF: %s", buf.Bytes())
		}
		m := doc.AWS.Metrics[0]
		if m.Namespace != "UrlShortener" || len(m.Dimensions) != 1 || len(m.Dimensions[0]) != 1 || m.Dimensions[0][0] != "service" || len(m.Metrics) != 1 || m.Metrics[0].Name != metric || m.Metrics[0].Unit != "Count" {
			t.Fatalf("invalid EMF directive: %s", buf.Bytes())
		}
		var fields map[string]any
		json.Unmarshal(buf.Bytes(), &fields)
		if fields[metric] != float64(1) {
			t.Fatalf("missing metric value: %s", buf.Bytes())
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestMetricErrors(t *testing.T) {
	if err := EmitCountMetric(failingWriter{}, "UrlShortener", "resolve-link", "LinkResolved", 1); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("lost write error: %v", err)
	}
	for _, tc := range []struct {
		name  string
		value float64
	}{{"service", 1}, {"_aws", 1}, {"", 1}, {"Click", math.NaN()}, {"Click", math.Inf(1)}, {"Click", -1}} {
		var buf bytes.Buffer
		if err := EmitCountMetric(&buf, "UrlShortener", "resolve-link", tc.name, tc.value); err == nil || buf.Len() != 0 {
			t.Fatalf("accepted invalid metric %+v", tc)
		}
	}
}

func TestInstrumentAWSConfigCreatesSubsegment(t *testing.T) {
	var cfg aws.Config
	InstrumentAWSConfig(&cfg)
	stack := middleware.NewStack("test", smithyhttp.NewStackRequest)
	for _, option := range cfg.APIOptions {
		if err := option(stack); err != nil {
			t.Fatal(err)
		}
	}
	ctx, parent := xray.BeginSegment(context.Background(), "test")
	defer parent.Close(nil)
	_, _, err := stack.HandleMiddleware(ctx, nil, middleware.HandlerFunc(func(ctx context.Context, request any) (any, middleware.Metadata, error) {
		segment := xray.GetSegment(ctx)
		if segment == nil || segment == parent {
			t.Error("AWS call did not create a child X-Ray segment")
		}
		if request.(*smithyhttp.Request).Header.Get("X-Amzn-Trace-Id") == "" {
			t.Error("AWS request is missing trace propagation header")
		}
		return nil, middleware.Metadata{}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
}
