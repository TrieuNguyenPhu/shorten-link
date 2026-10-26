// Command smoke verifies a deployed Phase 03 dev stack using Cognito PKCE tokens.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type result struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}

type tokenSet struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

type probe struct {
	client  *http.Client
	region  string
	results []result
}

func newProbe(region string) *probe {
	return &probe{
		region: region,
		client: &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (p *probe) check(ctx context.Context, name, method, target, body, bearerToken string, want int) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, target, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", name, err)
	}
	p.results = append(p.results, result{Name: name, Status: response.StatusCode})
	if response.StatusCode != want {
		raw, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		response.Body.Close()
		return nil, fmt.Errorf("%s: status %d, expected %d (%s)", name, response.StatusCode, want, strings.TrimSpace(string(raw)))
	}
	return response, nil
}

func (p *probe) status(ctx context.Context, name, method, target, body, bearerToken string, want int) error {
	response, err := p.check(ctx, name, method, target, body, bearerToken, want)
	if err == nil {
		response.Body.Close()
	}
	return err
}

func (p *probe) redirectTo(ctx context.Context, name, target string, want int, wantLocation string) error {
	response, err := p.check(ctx, name, "GET", target, "", "", want)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.Header.Get("Cache-Control") != "no-store" || (want == 302 && response.Header.Get("Location") != wantLocation) {
		return fmt.Errorf("%s: redirect headers differ from contract (cache=%q location=%q)", name, response.Header.Get("Cache-Control"), response.Header.Get("Location"))
	}
	return nil
}

func (p *probe) redirect(ctx context.Context, name, target string, want int) error {
	return p.redirectTo(ctx, name, target, want, "https://example.com/phase-02-smoke?q=1#verified")
}

func redirectTargets(api, site string) []string {
	targets := []string{api}
	if site != "" {
		targets = append(targets, site)
	}
	return targets
}

var csrfInputPattern = regexp.MustCompile(`name="_csrf"\s+value="([^"]+)"`)

func authenticatePKCE(ctx context.Context, cognitoDomain, clientID, redirectURI, username, password, scope string) (tokenSet, error) {
	var tokens tokenSet
	domainURL, err := url.Parse(cognitoDomain)
	if err != nil {
		return tokens, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return tokens, err
	}
	client := &http.Client{
		Timeout: 20 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if !strings.EqualFold(req.URL.Host, domainURL.Host) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return tokens, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	authQuery := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {scope},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	authURL := cognitoDomain + "/oauth2/authorize?" + authQuery.Encode()
	loginPageReq, err := http.NewRequestWithContext(ctx, "GET", authURL, nil)
	if err != nil {
		return tokens, err
	}
	loginPageResp, err := client.Do(loginPageReq)
	if err != nil {
		return tokens, fmt.Errorf("fetch Hosted UI login page: %w", err)
	}
	loginHTML, _ := io.ReadAll(io.LimitReader(loginPageResp.Body, 256*1024))
	loginPageResp.Body.Close()

	csrf := ""
	if match := csrfInputPattern.FindSubmatch(loginHTML); len(match) == 2 {
		csrf = string(match[1])
	}
	if csrf == "" {
		for _, c := range jar.Cookies(domainURL) {
			if c.Name == "XSRF-TOKEN" {
				csrf = c.Value
				break
			}
		}
	}
	if csrf == "" {
		return tokens, fmt.Errorf("Hosted UI login page did not include a CSRF token")
	}

	form := url.Values{
		"_csrf":              {csrf},
		"username":           {username},
		"password":           {password},
		"cognitoAsfData":     {""},
		"signInSubmitButton": {"Sign in"},
	}
	postReq, err := http.NewRequestWithContext(ctx, "POST", loginPageResp.Request.URL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return tokens, err
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postResp, err := client.Do(postReq)
	if err != nil {
		return tokens, fmt.Errorf("submit Hosted UI credentials: %w", err)
	}
	postResp.Body.Close()
	if postResp.StatusCode != http.StatusFound && postResp.StatusCode != http.StatusSeeOther {
		return tokens, fmt.Errorf("Hosted UI login returned status %d", postResp.StatusCode)
	}
	callbackURL, err := url.Parse(postResp.Header.Get("Location"))
	if err != nil {
		return tokens, err
	}
	if callbackURL.Query().Get("state") != state || callbackURL.Query().Get("code") == "" {
		return tokens, fmt.Errorf("Hosted UI callback did not return expected authorization code (error=%q)", callbackURL.Query().Get("error"))
	}

	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {callbackURL.Query().Get("code")},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	tokenReq, err := http.NewRequestWithContext(ctx, "POST", cognitoDomain+"/oauth2/token", strings.NewReader(tokenForm.Encode()))
	if err != nil {
		return tokens, err
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return tokens, fmt.Errorf("exchange PKCE code: %w", err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		return tokens, fmt.Errorf("token endpoint returned status %d", tokenResp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(tokenResp.Body, 65536)).Decode(&tokens); err != nil {
		return tokens, err
	}
	if tokens.AccessToken == "" || tokens.IDToken == "" {
		return tokens, fmt.Errorf("token endpoint returned incomplete token set")
	}
	return tokens, nil
}

func runAWSCLI(ctx context.Context, profile, region string, args ...string) error {
	fullArgs := append([]string{"--profile", profile, "--region", region}, args...)
	cmd := exec.CommandContext(ctx, "aws", fullArgs...)
	cmd.Env = append(os.Environ(), "AWS_PAGER=")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("aws %s failed: %w (%s)", strings.Join(args[:2], " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func provisionTempCognitoUser(ctx context.Context, profile, region, poolID, email, password string) error {
	if err := runAWSCLI(ctx, profile, region,
		"cognito-idp", "admin-create-user",
		"--user-pool-id", poolID,
		"--username", email,
		"--user-attributes", "Name=email,Value="+email, "Name=email_verified,Value=true",
		"--message-action", "SUPPRESS",
	); err != nil {
		return err
	}
	return runAWSCLI(ctx, profile, region,
		"cognito-idp", "admin-set-user-password",
		"--user-pool-id", poolID,
		"--username", email,
		"--password", password,
		"--permanent",
	)
}

func deleteTempCognitoUser(profile, region, poolID, email string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = runAWSCLI(ctx, profile, region,
		"cognito-idp", "admin-delete-user",
		"--user-pool-id", poolID,
		"--username", email,
	)
}

func (p *probe) run(ctx context.Context, api, site, bucket string, userA, userB, readOnly tokenSet) error {
	if site != "" {
		for _, tc := range []struct{ name, path string }{
			{"home", "/"},
			{"dashboard deep link", "/dashboard/"},
			{"auth callback deep link", "/auth/callback/"},
			{"CloudFront health", "/health"},
		} {
			if err := p.status(ctx, tc.name, "GET", site+tc.path, "", "", 200); err != nil {
				return err
			}
		}
	}
	if err := p.status(ctx, "API health", "GET", api+"/health", "", "", 200); err != nil {
		return err
	}

	// Gateway JWT authorizer and scope enforcement checks.
	if err := p.status(ctx, "missing token create denied", "POST", api+"/api/links", `{"url":"https://example.com"}`, "", 401); err != nil {
		return err
	}
	if err := p.status(ctx, "missing token list denied", "GET", api+"/api/links", "", "", 401); err != nil {
		return err
	}
	if err := p.status(ctx, "malformed token denied", "GET", api+"/api/links", "", "not-a-valid-jwt", 401); err != nil {
		return err
	}
	fakeJWT := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"attacker","iss":"https://evil.example","exp":4102444800}`)) + ".sig"
	if err := p.status(ctx, "untrusted issuer token denied", "GET", api+"/api/links", "", fakeJWT, 401); err != nil {
		return err
	}
	if err := p.status(ctx, "id token missing scope denied", "GET", api+"/api/links", "", userA.IDToken, 403); err != nil {
		return err
	}
	if err := p.status(ctx, "read-only token on write route denied", "POST", api+"/api/links", `{"url":"https://example.com"}`, readOnly.AccessToken, 403); err != nil {
		return err
	}
	if err := p.status(ctx, "read-only token on read route allowed", "GET", api+"/api/links", "", readOnly.AccessToken, 200); err != nil {
		return err
	}

	// Multi-user A/B creation and isolation checks.
	ts := time.Now().UnixMilli()
	aliasA := fmt.Sprintf("smoke-a-%d", ts)
	aliasB := fmt.Sprintf("smoke-b-%d", ts)
	initialURL := "https://example.com/phase-02-smoke?q=1#verified"
	updatedURL := "https://example.com/phase-03-updated?q=2#edited"

	bodyA, _ := json.Marshal(map[string]string{"url": initialURL, "alias": aliasA})
	respA, err := p.check(ctx, "user A create", "POST", api+"/api/links", string(bodyA), userA.AccessToken, 201)
	if err != nil {
		return err
	}
	var createdA struct {
		Code    string `json:"code"`
		Enabled bool   `json:"enabled"`
		Version int64  `json:"version"`
	}
	err = json.NewDecoder(io.LimitReader(respA.Body, 16384)).Decode(&createdA)
	respA.Body.Close()
	if err != nil || createdA.Code != aliasA || !createdA.Enabled || createdA.Version != 1 {
		return fmt.Errorf("user A create response does not match contract: %+v err=%v", createdA, err)
	}

	bodyB, _ := json.Marshal(map[string]string{"url": initialURL, "alias": aliasB})
	if err := p.status(ctx, "user B create", "POST", api+"/api/links", string(bodyB), userB.AccessToken, 201); err != nil {
		return err
	}

	if err := p.status(ctx, "user A metadata", "GET", api+"/api/links/"+aliasA, "", userA.AccessToken, 200); err != nil {
		return err
	}
	if err := p.status(ctx, "user B cross-owner metadata denied", "GET", api+"/api/links/"+aliasA, "", userB.AccessToken, 404); err != nil {
		return err
	}
	if err := p.status(ctx, "user B cross-owner update denied", "PATCH", api+"/api/links/"+aliasA, `{"enabled":false,"version":1}`, userB.AccessToken, 404); err != nil {
		return err
	}
	if err := p.status(ctx, "user B cross-owner delete denied", "DELETE", api+"/api/links/"+aliasA+"?version=1", "", userB.AccessToken, 404); err != nil {
		return err
	}

	// Verify list isolation for User A and User B.
	listRespA, err := p.check(ctx, "user A list", "GET", api+"/api/links", "", userA.AccessToken, 200)
	if err != nil {
		return err
	}
	rawListA, _ := io.ReadAll(io.LimitReader(listRespA.Body, 65536))
	listRespA.Body.Close()
	if !strings.Contains(string(rawListA), aliasA) || strings.Contains(string(rawListA), aliasB) {
		return fmt.Errorf("user A list leaked or omitted expected links")
	}

	listRespB, err := p.check(ctx, "user B list", "GET", api+"/api/links", "", userB.AccessToken, 200)
	if err != nil {
		return err
	}
	rawListB, _ := io.ReadAll(io.LimitReader(listRespB.Body, 65536))
	listRespB.Body.Close()
	if !strings.Contains(string(rawListB), aliasB) || strings.Contains(string(rawListB), aliasA) {
		return fmt.Errorf("user B list leaked or omitted expected links")
	}

	forgedCursor := base64.RawURLEncoding.EncodeToString([]byte(`{"PK":"LINK#` + aliasA + `","GSI1PK":"OWNER#other-owner","GSI1SK":"2026-10-26T00:00:00Z#` + aliasA + `"}`))
	if err := p.status(ctx, "forged cursor rejected", "GET", api+"/api/links?cursor="+forgedCursor, "", userB.AccessToken, 400); err != nil {
		return err
	}

	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "active redirect", base+"/r/"+aliasA, 302, initialURL); err != nil {
			return err
		}
	}

	// Edit destination and verify optimistic concurrency version increment.
	editBody, _ := json.Marshal(map[string]any{"url": updatedURL, "version": 1})
	if err := p.status(ctx, "user A edit destination", "PATCH", api+"/api/links/"+aliasA, string(editBody), userA.AccessToken, 200); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "updated redirect", base+"/r/"+aliasA, 302, updatedURL); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "stale version update conflict", "PATCH", api+"/api/links/"+aliasA, `{"enabled":false,"version":1}`, userA.AccessToken, 409); err != nil {
		return err
	}

	// Disable and re-enable with current version.
	if err := p.status(ctx, "user A disable", "PATCH", api+"/api/links/"+aliasA, `{"enabled":false,"version":2}`, userA.AccessToken, 200); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "disabled redirect uncached", base+"/r/"+aliasA, 410, ""); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "user A re-enable", "PATCH", api+"/api/links/"+aliasA, `{"enabled":true,"version":3}`, userA.AccessToken, 200); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "re-enabled redirect", base+"/r/"+aliasA, 302, updatedURL); err != nil {
			return err
		}
		if err := p.redirectTo(ctx, "missing redirect", base+"/r/"+aliasA+"-missing", 404, ""); err != nil {
			return err
		}
	}

	// Delete with tombstone semantics.
	if err := p.status(ctx, "stale version delete conflict", "DELETE", api+"/api/links/"+aliasA+"?version=1", "", userA.AccessToken, 409); err != nil {
		return err
	}
	if err := p.status(ctx, "user A delete tombstone", "DELETE", api+"/api/links/"+aliasA+"?version=4", "", userA.AccessToken, 204); err != nil {
		return err
	}
	if err := p.status(ctx, "deleted metadata not found", "GET", api+"/api/links/"+aliasA, "", userA.AccessToken, 404); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "deleted redirect gone", base+"/r/"+aliasA, 410, ""); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "deleted alias reuse conflict", "POST", api+"/api/links", string(bodyA), userA.AccessToken, 409); err != nil {
		return err
	}

	if err := p.status(ctx, "invalid URL rejected", "POST", api+"/api/links", `{"url":"https://example.com:99999"}`, userA.AccessToken, 400); err != nil {
		return err
	}
	expiring := aliasB + "-exp"
	expires := time.Now().Add(15 * time.Second)
	expBody, _ := json.Marshal(map[string]string{"url": initialURL, "alias": expiring, "expiresAt": expires.UTC().Format(time.RFC3339Nano)})
	if err := p.status(ctx, "create expiring link", "POST", api+"/api/links", string(expBody), userA.AccessToken, 201); err != nil {
		return err
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "before expiry", base+"/r/"+expiring, 302, initialURL); err != nil {
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(expires) + time.Second):
	}
	for _, base := range redirectTargets(api, site) {
		if err := p.redirectTo(ctx, "expired redirect", base+"/r/"+expiring, 410, ""); err != nil {
			return err
		}
	}
	if err := p.status(ctx, "private S3 denies direct access", "GET", "https://"+bucket+".s3."+p.region+".amazonaws.com/index.html", "", "", 403); err != nil {
		return err
	}
	if err := p.status(ctx, "clean up expiring link", "DELETE", api+"/api/links/"+expiring+"?version=1", "", userA.AccessToken, 204); err != nil {
		return err
	}
	return p.status(ctx, "clean up user B smoke link", "DELETE", api+"/api/links/"+aliasB+"?version=1", "", userB.AccessToken, 204)
}

func main() {
	api := flag.String("api", "", "direct HTTPS API Gateway origin")
	site := flag.String("site", "", "HTTPS CloudFront/site origin; omit for backend-only checks")
	bucket := flag.String("bucket", "", "private web bucket")
	region := flag.String("region", "", "deployed AWS region")
	profile := flag.String("profile", "default", "AWS CLI credential profile")
	userPoolID := flag.String("user-pool-id", "", "Cognito User Pool ID")
	clientID := flag.String("client-id", "", "Cognito User Pool Web Client ID")
	cognitoDomain := flag.String("cognito-domain", "", "HTTPS Cognito Hosted UI domain origin")
	flag.Parse()

	for _, origin := range append(redirectTargets(*api, *site), *cognitoDomain) {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			fmt.Fprintln(os.Stderr, "-api, -site, and -cognito-domain must be HTTPS origins without paths")
			os.Exit(1)
		}
	}
	if *region == "" || *bucket == "" || *userPoolID == "" || *clientID == "" {
		fmt.Fprintln(os.Stderr, "-region, -bucket, -user-pool-id, and -client-id are required")
		os.Exit(1)
	}
	redirectURI := "https://npt-shortenlink.dev/auth/callback/"
	if *site != "" {
		redirectURI = *site + "/auth/callback/"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	ts := time.Now().UnixMilli()
	emailA := fmt.Sprintf("smoke-user-a-%d@example.com", ts)
	emailB := fmt.Sprintf("smoke-user-b-%d@example.com", ts)
	randBytes := make([]byte, 12)
	_, _ = rand.Read(randBytes)
	tempPassword := "Sm0ke!" + hex.EncodeToString(randBytes) + "9aA"

	defer deleteTempCognitoUser(*profile, *region, *userPoolID, emailA)
	defer deleteTempCognitoUser(*profile, *region, *userPoolID, emailB)

	if err := provisionTempCognitoUser(ctx, *profile, *region, *userPoolID, emailA, tempPassword); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := provisionTempCognitoUser(ctx, *profile, *region, *userPoolID, emailB, tempPassword); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fullScopes := "openid email shortly-api/links.read shortly-api/links.write"
	readOnlyScopes := "openid email shortly-api/links.read"

	tokensA, err := authenticatePKCE(ctx, *cognitoDomain, *clientID, redirectURI, emailA, tempPassword, fullScopes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "user A PKCE auth failed:", err)
		os.Exit(1)
	}
	tokensB, err := authenticatePKCE(ctx, *cognitoDomain, *clientID, redirectURI, emailB, tempPassword, fullScopes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "user B PKCE auth failed:", err)
		os.Exit(1)
	}
	tokensRO, err := authenticatePKCE(ctx, *cognitoDomain, *clientID, redirectURI, emailA, tempPassword, readOnlyScopes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read-only PKCE auth failed:", err)
		os.Exit(1)
	}

	p := newProbe(*region)
	err = p.run(ctx, *api, *site, *bucket, tokensA, tokensB, tokensRO)
	_ = json.NewEncoder(os.Stdout).Encode(p.results)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
