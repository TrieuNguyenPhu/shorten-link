// Command local-http runs the real HTTP handlers against explicit loopback dependencies.
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	appevents "github.com/TrieuNguyenPhu/shorten-link/backend/internal/events"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/httpapi"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type localConfig struct {
	Listen              string   `json:"listen"`
	DynamoDBEndpoint    string   `json:"dynamodbEndpoint"`
	EventBridgeEndpoint string   `json:"eventBridgeEndpoint"`
	TableName           string   `json:"tableName"`
	EventBus            string   `json:"eventBus"`
	Users               []string `json:"users"`
}

func loopbackEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "http" && ip != nil && ip.IsLoopback() && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && (u.Path == "" || u.Path == "/") && u.Port() != ""
}

func (c localConfig) validate() error {
	host, _, err := net.SplitHostPort(c.Listen)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || !ip.IsLoopback() {
		return errors.New("listen must use a loopback IP and port")
	}
	if !loopbackEndpoint(c.DynamoDBEndpoint) || !loopbackEndpoint(c.EventBridgeEndpoint) {
		return errors.New("dependency endpoints must be explicit loopback HTTP URLs")
	}
	if strings.TrimSpace(c.TableName) == "" || strings.TrimSpace(c.EventBus) == "" || len(c.Users) == 0 {
		return errors.New("tableName, eventBus and users are required")
	}
	for _, user := range c.Users {
		if strings.TrimSpace(user) == "" {
			return errors.New("empty test identity")
		}
	}
	return nil
}

func loadConfig(path string) (localConfig, error) {
	var cfg localConfig
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 65536))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return cfg, errors.New("config must contain one JSON object")
	}
	return cfg, cfg.validate()
}

func localClients(c localConfig) (*dynamodb.Client, *eventbridge.Client) {
	// Never load operator credentials, environment endpoints or the default AWS chain.
	sdk := aws.Config{Region: "ap-southeast-1", Credentials: credentials.NewStaticCredentialsProvider("local", "local", ""), Retryer: func() aws.Retryer { return aws.NopRetryer{} }}
	ddb := dynamodb.NewFromConfig(sdk, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(c.DynamoDBEndpoint) })
	eb := eventbridge.NewFromConfig(sdk, func(o *eventbridge.Options) { o.BaseEndpoint = aws.String(c.EventBridgeEndpoint) })
	return ddb, eb
}

func ensureTable(ctx context.Context, client *dynamodb.Client, table string) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{TableName: aws.String(table), BillingMode: types.BillingModePayPerRequest, AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS}}, KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash}, {AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange}}, GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{IndexName: aws.String("GSI1"), KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("GSI1PK"), KeyType: types.KeyTypeHash}, {AttributeName: aws.String("GSI1SK"), KeyType: types.KeyTypeRange}}, Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll}}}})
	var exists *types.ResourceInUseException
	if err != nil && !errors.As(err, &exists) {
		return err
	}
	return dynamodb.NewTableExistsWaiter(client).Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)}, 10*time.Second)
}

func newAdapter(handler *httpapi.Handler, c localConfig) http.Handler {
	mux := http.NewServeMux()
	wrap := func(next httpapi.LambdaHandler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(io.LimitReader(r.Body, 16385))
			if err != nil || len(body) > 16384 {
				http.Error(w, "invalid body", 400)
				return
			}
			query := make(map[string]string)
			for key := range r.URL.Query() {
				query[key] = r.URL.Query().Get(key)
			}
			req := events.APIGatewayV2HTTPRequest{QueryStringParameters: query, Version: "2.0", Body: string(body), PathParameters: map[string]string{"code": r.PathValue("code")}, RequestContext: events.APIGatewayV2HTTPRequestContext{DomainName: c.Listen, RequestID: rand.Text()}}
			user := r.Header.Get("X-Local-User")
			for _, allowed := range c.Users {
				if user == allowed {
					req.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": user}}}
					break
				}
			}
			res, err := httpapi.WithDeadline(next)(r.Context(), req)
			if err != nil {
				http.Error(w, "internal server error", http.StatusBadGateway)
				return
			}
			for key, value := range res.Headers {
				w.Header().Set(key, value)
			}
			w.WriteHeader(res.StatusCode)
			_, _ = io.WriteString(w, res.Body)
		}
	}
	mux.HandleFunc("POST /links", wrap(handler.Create))
	mux.HandleFunc("GET /links", wrap(handler.List))
	mux.HandleFunc("DELETE /links/{code}", wrap(handler.Delete))
	mux.HandleFunc("GET /{code}", wrap(handler.Resolve))
	// Keep method mismatch responses uniform with unknown routes in the local adapter.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "POST" && r.Method != "DELETE" {
			http.NotFound(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// newEventRecorder is an opt-in protocol fixture, not an EventBridge emulator.
// It records publication only; it does not produce analytics or deliver events.
func newEventRecorder(writer io.Writer) http.Handler {
	var mu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Entries []struct{ EventBusName, Source, DetailType, Detail string }
		}
		if r.Method != "POST" || r.Header.Get("X-Amz-Target") != "AWSEvents.PutEvents" {
			http.Error(w, "expected PutEvents", 400)
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, 65536))
		if err := decoder.Decode(&in); err != nil || len(in.Entries) != 1 {
			http.Error(w, "invalid entries", 400)
			return
		}
		entry := in.Entries[0]
		var detail domain.ClickEvent
		if err := json.Unmarshal([]byte(entry.Detail), &detail); err != nil || entry.EventBusName == "" || entry.Source != "urlshortener" || entry.DetailType != "link.clicked" || detail.Code == "" || detail.At == "" {
			http.Error(w, "invalid event", 400)
			return
		}
		mu.Lock()
		err := json.NewEncoder(writer).Encode(map[string]any{"localRecordedClick": detail})
		mu.Unlock()
		if err != nil {
			http.Error(w, "record failed", 500)
			return
		}
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_ = json.NewEncoder(w).Encode(map[string]any{"FailedEntryCount": 0, "Entries": []map[string]string{{"EventId": rand.Text()}}})
	})
}

func main() {
	configPath := flag.String("config", "../scripts/local-http/config.example.json", "local JSON config")
	initTable := flag.Bool("init-table", false, "create the local test table")
	record := flag.Bool("record-events", false, "start an opt-in local PutEvents protocol recorder (no event delivery)")
	flag.Parse()
	if err := run(*configPath, *initTable, *record); err != nil {
		log.Fatal(err)
	}
}

func run(path string, initTable, record bool) error {
	c, err := loadConfig(path)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	client, eb := localClients(c)
	if initTable {
		setup, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := ensureTable(setup, client, c.TableName); err != nil {
			return err
		}
	}
	logger := middleware.NewLogger("local-http")
	h := httpapi.New(service.New(repository.New(client, c.TableName), appevents.NewPublisher(eb, c.EventBus)), "http://"+c.Listen, logger, os.Stdout)
	servers := []*http.Server{{Addr: c.Listen, Handler: newAdapter(h, c), ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}}
	if record {
		u, _ := url.Parse(c.EventBridgeEndpoint)
		servers = append(servers, &http.Server{Addr: u.Host, Handler: newEventRecorder(os.Stdout), ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second})
	}
	failures := make(chan error, len(servers))
	for _, server := range servers {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			for _, s := range servers {
				_ = s.Close()
			}
			return err
		}
		go func() { failures <- server.Serve(listener) }()
	}
	logger.Info("local harness started", "address", c.Listen, "identityHeader", "X-Local-User", "eventRecorder", record)
	select {
	case <-ctx.Done():
	case err = <-failures:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, s := range servers {
		err = errors.Join(err, s.Shutdown(shutdown))
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("local server: %w", err)
	}
	return nil
}
