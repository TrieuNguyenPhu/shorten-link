// Command smoke verifies a deployed Phase 02 dev stack using the caller's AWS CLI profile.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

type result struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}
type probe struct {
	client      *http.Client
	credentials aws.Credentials
	region      string
	results     []result
}

func newProbe(credentials aws.Credentials, region string) *probe {
	return &probe{credentials: credentials, region: region, client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (p *probe) check(ctx context.Context, name, method, target, body string, signed bool, want int) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, target, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if signed {
		digest := sha256.Sum256([]byte(body))
		if err := v4.NewSigner().SignHTTP(ctx, p.credentials, request, hex.EncodeToString(digest[:]), "execute-api", p.region, time.Now()); err != nil {
			return nil, fmt.Errorf("%s: signing failed", name)
		}
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", name, err)
	}
	p.results = append(p.results, result{Name: name, Status: response.StatusCode})
	if response.StatusCode != want {
		response.Body.Close()
		return nil, fmt.Errorf("%s: status %d, expected %d", name, response.StatusCode, want)
	}
	return response, nil
}

func (p *probe) status(ctx context.Context, name, method, target, body string, signed bool, want int) error {
	response, err := p.check(ctx, name, method, target, body, signed, want)
	if err == nil {
		response.Body.Close()
	}
	return err
}

func (p *probe) redirect(ctx context.Context, name, target string, want int) error {
	response, err := p.check(ctx, name, "GET", target, "", false, want)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.Header.Get("Cache-Control") != "no-store" || (want == 302 && response.Header.Get("Location") != "https://example.com/phase-02-smoke?q=1#verified") {
		return fmt.Errorf("%s: redirect headers differ from contract", name)
	}
	return nil
}

func redirectTargets(api, site string) []string {
	targets := []string{api}
	if site != "" {
		targets = append(targets, site)
	}
	return targets
}

func (p *probe) run(ctx context.Context, api, site, bucket string) error {
	if site != "" {
		for _, tc := range []struct{ name, path string }{{"home", "/"}, {"dashboard deep link", "/dashboard/"}, {"CloudFront health", "/health"}} {
			if err := p.status(ctx, tc.name, "GET", site+tc.path, "", false, 200); err != nil {
				return err
			}
		}
	}
	if err := p.status(ctx, "API health", "GET", api+"/health", "", false, 200); err != nil {
		return err
	}
	if err := p.status(ctx, "unsigned create denied", "POST", api+"/api/links", `{"url":"https://example.com"}`, false, 403); err != nil {
		return err
	}
	if err := p.status(ctx, "unsigned list denied", "GET", api+"/api/links", "", false, 403); err != nil {
		return err
	}
	alias := fmt.Sprintf("smoke-%d", time.Now().UnixMilli())
	body, _ := json.Marshal(map[string]string{"url": "https://example.com/phase-02-smoke?q=1#verified", "alias": alias})
	response, err := p.check(ctx, "signed create", "POST", api+"/api/links", string(body), true, 201)
	if err != nil {
		return err
	}
	var created struct {
		Code    string `json:"code"`
		Enabled bool   `json:"enabled"`
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(&created)
	response.Body.Close()
	if err != nil || created.Code != alias || !created.Enabled {
		return fmt.Errorf("create response does not match the persisted link")
	}
	if err := p.status(ctx, "unsigned metadata denied", "GET", api+"/api/links/"+alias, "", false, 403); err != nil {
		return err
	}
	if err := p.status(ctx, "unsigned update denied", "PATCH", api+"/api/links/"+alias, `{"enabled":false}`, false, 403); err != nil {
		return err
	}
	if err := p.status(ctx, "signed metadata", "GET", api+"/api/links/"+alias, "", true, 200); err != nil {
		return err
	}
	if err := p.status(ctx, "signed list", "GET", api+"/api/links", "", true, 200); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirect(ctx, "active redirect", base+"/r/"+alias, 302); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "signed disable", "PATCH", api+"/api/links/"+alias, `{"enabled":false}`, true, 200); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirect(ctx, "disabled redirect uncached", base+"/r/"+alias, 410); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "signed re-enable", "PATCH", api+"/api/links/"+alias, `{"enabled":true}`, true, 200); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirect(ctx, "re-enabled redirect", base+"/r/"+alias, 302); err != nil {
			return err
		}
		if err := p.redirect(ctx, "missing redirect", base+"/r/"+alias+"-missing", 404); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "invalid URL rejected", "POST", api+"/api/links", `{"url":"https://example.com:99999"}`, true, 400); err != nil {
		return err
	}
	expiring := alias + "-exp"
	expires := time.Now().Add(60 * time.Second)
	body, _ = json.Marshal(map[string]string{"url": "https://example.com/phase-02-smoke?q=1#verified", "alias": expiring, "expiresAt": expires.UTC().Format(time.RFC3339Nano)})
	if err := p.status(ctx, "create expiring link", "POST", api+"/api/links", string(body), true, 201); err != nil {
		return err
	}
	if err := p.redirect(ctx, "before expiry", api+"/r/"+expiring, 302); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(expires) + time.Second):
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirect(ctx, "expired redirect", base+"/r/"+expiring, 410); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "private S3 denies direct access", "GET", "https://"+bucket+".s3."+p.region+".amazonaws.com/index.html", "", false, 403); err != nil {
		return err
	}
	return p.status(ctx, "leave smoke link disabled", "PATCH", api+"/api/links/"+alias, `{"enabled":false}`, true, 200)
}

func main() {
	api := flag.String("api", "", "direct HTTPS API Gateway origin")
	site := flag.String("site", "", "HTTPS CloudFront/site origin; omit for backend-only checks")
	bucket := flag.String("bucket", "", "private web bucket")
	region := flag.String("region", "", "deployed AWS region")
	profile := flag.String("profile", "default", "AWS CLI credential profile")
	flag.Parse()
	for _, origin := range redirectTargets(*api, *site) {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			fmt.Fprintln(os.Stderr, "-api and -site must be HTTPS origins without paths")
			os.Exit(1)
		}
	}
	if *site == "" {
		fmt.Fprintln(os.Stderr, "Backend-only smoke: CloudFront and static-site checks are not included.")
	}
	if *region == "" || *bucket == "" {
		fmt.Fprintln(os.Stderr, "-region and -bucket are required")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	// Capture the configured CLI provider in memory; never print or persist credentials.
	raw, err := exec.CommandContext(ctx, "aws", "configure", "export-credentials", "--profile", *profile, "--format", "process").Output()
	var exported struct{ AccessKeyId, SecretAccessKey, SessionToken string }
	if err != nil || json.Unmarshal(raw, &exported) != nil || exported.AccessKeyId == "" || exported.SecretAccessKey == "" {
		fmt.Fprintln(os.Stderr, "unable to resolve AWS CLI credentials")
		os.Exit(1)
	}
	p := newProbe(aws.Credentials{AccessKeyID: exported.AccessKeyId, SecretAccessKey: exported.SecretAccessKey, SessionToken: exported.SessionToken}, *region)
	err = p.run(ctx, *api, *site, *bucket)
	_ = json.NewEncoder(os.Stdout).Encode(p.results)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
