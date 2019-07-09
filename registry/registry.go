package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/api/types"
)

// Registry defines the client for retrieving information from the registry API.
type Registry struct {
	URL      string
	Domain   string
	Username string
	Password string
	Client   *http.Client
	Logf     LogfCallback
	Opt      Opt
}

var reProtocol = regexp.MustCompile("^https?://")

// LogfCallback is the callback for formatting logs.
type LogfCallback func(format string, args ...interface{})

// Quiet discards logs silently.
func Quiet(format string, args ...interface{}) {}

// Log passes log messages to the logging package.
func Log(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Opt holds the options for a new registry.
type Opt struct {
	Domain   string
	Insecure bool
	Debug    bool
	SkipPing bool
	NonSSL   bool
	Timeout  time.Duration
	Headers  map[string]string
}

// New creates a new Registry struct with the given URL and credentials.
func New(ctx context.Context, auth types.AuthConfig, opt Opt) (*Registry, error) {
	transport := http.DefaultTransport

	if opt.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return newFromTransport(ctx, auth, transport, opt)
}

func newFromTransport(ctx context.Context, auth types.AuthConfig, transport http.RoundTripper, opt Opt) (*Registry, error) {
	if len(opt.Domain) < 1 || opt.Domain == "docker.io" {
		opt.Domain = auth.ServerAddress
	}
	url := strings.TrimSuffix(opt.Domain, "/")
	authURL := strings.TrimSuffix(auth.ServerAddress, "/")

	if !reProtocol.MatchString(url) {
		if !opt.NonSSL {
			url = "https://" + url
		} else {
			url = "http://" + url
		}
	}

	if !reProtocol.MatchString(authURL) {
		if !opt.NonSSL {
			authURL = "https://" + authURL
		} else {
			authURL = "http://" + authURL
		}
	}

	tokenTransport := &TokenTransport{
		Transport: transport,
		Username:  auth.Username,
		Password:  auth.Password,
	}
	basicAuthTransport := &BasicTransport{
		Transport: tokenTransport,
		URL:       authURL,
		Username:  auth.Username,
		Password:  auth.Password,
	}
	errorTransport := &ErrorTransport{
		Transport: basicAuthTransport,
	}
	customTransport := &CustomTransport{
		Transport: errorTransport,
		Headers:   opt.Headers,
	}

	// set the logging
	logf := Quiet
	if opt.Debug {
		logf = Log
	}

	registry := &Registry{
		URL:    url,
		Domain: reProtocol.ReplaceAllString(url, ""),
		Client: &http.Client{
			Timeout:   opt.Timeout,
			Transport: customTransport,
		},
		Username: auth.Username,
		Password: auth.Password,
		Logf:     logf,
		Opt:      opt,
	}

	if registry.Pingable() && !opt.SkipPing {
		if err := registry.Ping(ctx); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// url returns a registry URL with the passed arguements concatenated.
func (r *Registry) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

func (r *Registry) getJSON(ctx context.Context, url string, response interface{}) (http.Header, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	switch response.(type) {
	case *schema2.Manifest:
		req.Header.Add("Accept", schema2.MediaTypeManifest)
	case *manifestlist.ManifestList:
		req.Header.Add("Accept", manifestlist.MediaTypeManifestList)
	}

	resp, err := r.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	r.Logf("registry.registry resp.Status=%s", resp.Status)

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return resp.Header, nil
}
