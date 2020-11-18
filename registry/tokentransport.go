package registry

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

var gcrMatcher = regexp.MustCompile(`https://([a-z]+\.|)gcr\.io/`)

// TokenTransport defines the data structure for authentication via tokens.
type TokenTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string
}

// RoundTrip defines the round tripper for token transport.
func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	authService, err := isTokenDemand(resp)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	if authService == nil {
		return resp, nil
	}

	resp.Body.Close()

	return t.authAndRetry(authService, req)
}

type authToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
}

func (t authToken) String() (string, error) {
	if t.Token != "" {
		return t.Token, nil
	}
	if t.AccessToken != "" {
		return t.AccessToken, nil
	}
	return "", errors.New("auth token cannot be empty")
}

func (t *TokenTransport) authAndRetry(authService *authService, req *http.Request) (*http.Response, error) {
	token, authResp, err := t.auth(req.Context(), authService)
	if err != nil {
		return authResp, err
	}

	response, err := t.retry(req, token)
	if response != nil {
		response.Header.Set("request-token", token)
	}
	return response, err
}

func (t *TokenTransport) auth(ctx context.Context, authService *authService) (string, *http.Response, error) {
	authReq, err := authService.Request(t.Username, t.Password)
	if err != nil {
		return "", nil, err
	}

	c := http.Client{
		Transport: t.Transport,
	}

	resp, err := c.Do(authReq.WithContext(ctx))
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp, err
	}

	var authToken authToken
	if err := json.NewDecoder(resp.Body).Decode(&authToken); err != nil {
		return "", nil, err
	}

	token, err := authToken.String()
	return token, nil, err
}

func (t *TokenTransport) retry(req *http.Request, token string) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return t.Transport.RoundTrip(req)
}

type authService struct {
	Realm   *url.URL
	Service string
	Scope   []string
}

func (a *authService) Request(username, password string) (*http.Request, error) {
	q := a.Realm.Query()
	if len(a.Service) > 0 {
		q.Set("service", a.Service)
	}
	for _, s := range a.Scope {
		q.Set("scope", s)
	}
	//	q.Set("scope", "repository:r.j3ss.co/htop:push,pull")
	a.Realm.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", a.Realm.String(), nil)

	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	return req, err
}

func isTokenDemand(resp *http.Response) (*authService, error) {
	if resp == nil {
		return nil, nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return nil, nil
	}
	return parseAuthHeader(resp.Header)
}

// Token returns the required token for the specific resource url. If the registry requires basic authentication, this
// function returns ErrBasicAuth.
func (r *Registry) Token(ctx context.Context, url string) (string, error) {
	r.Logf("registry.token url=%s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	client := http.DefaultClient
	if r.Opt.Insecure {
		client = &http.Client{
			Timeout: r.Opt.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden && gcrMatcher.MatchString(url) {
		// GCR is not sending HTTP 401 on missing credentials but a HTTP 403 without
		// any further information about why the request failed. Sending the credentials
		// from the Docker config fixes this.
		return "", ErrBasicAuth
	}

	a, err := isTokenDemand(resp)
	if err != nil {
		return "", err
	}

	if a == nil {
		r.Logf("registry.token authService=nil")
		return "", nil
	}

	authReq, err := a.Request(r.Username, r.Password)
	if err != nil {
		return "", err
	}
	resp, err = http.DefaultClient.Do(authReq.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("getting token failed with StatusCode != StatusOK but %d", resp.StatusCode)
	}

	var authToken authToken
	if err := json.NewDecoder(resp.Body).Decode(&authToken); err != nil {
		return "", err
	}

	return authToken.String()
}

// Headers returns the authorization headers for a specific uri.
func (r *Registry) Headers(ctx context.Context, uri string) (map[string]string, error) {
	// Get the token.
	token, err := r.Token(ctx, uri)
	if err != nil {
		if err == ErrBasicAuth {
			// If we couldn't get a token because the server requires basic auth, just return basic auth headers.
			return map[string]string{
				"Authorization": fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(r.Username+":"+r.Password))),
			}, nil
		}
	}

	if len(token) < 1 {
		r.Logf("got empty token for %s", uri)
		return map[string]string{}, nil
	}

	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}, nil
}
