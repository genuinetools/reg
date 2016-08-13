package registry

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/distribution/digest"
)

// Registry defines the client for retriving information from the registry API.
type Registry struct {
	URL    string
	Client *http.Client
	Logf   LogfCallback
}

// LogfCallback is the callback for formatting logs.
type LogfCallback func(format string, args ...interface{})

// Quiet discards logs silently.
func Quiet(format string, args ...interface{}) {}

// Log passes log messages to the logging package.
func Log(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// New creates a new Registry struct with the given URL and credentials.
func New(registryURL, username, password string, debug bool) (*Registry, error) {
	transport := http.DefaultTransport

	return newFromTransport(registryURL, username, password, transport, debug)
}

// NewInsecure creates a new Registry struct with the given URL and credentials,
// using a http.Transport that will not verify an SSL certificate.
func NewInsecure(registryURL, username, password string, debug bool) (*Registry, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return newFromTransport(registryURL, username, password, transport, debug)
}

func newFromTransport(registryURL, username, password string, transport http.RoundTripper, debug bool) (*Registry, error) {
	url := "https://" + strings.TrimPrefix(strings.TrimSuffix(registryURL, "/"), "https://")
	transport = wrapTransport(transport, url, username, password)

	// set the logging
	logf := Quiet
	if debug {
		logf = Log
	}

	registry := &Registry{
		URL: url,
		Client: &http.Client{
			Transport: transport,
		},
		Logf: logf,
	}

	if err := registry.Ping(); err != nil {
		return nil, err
	}

	return registry, nil
}

// wrapTransport builds the transport stack necessary to authenticate to the
// registry API. It adds in support for OAuth bearer tokens and HTTP Basic auth,
// and sets up error handling this library relies on.
func wrapTransport(transport http.RoundTripper, url, username, password string) http.RoundTripper {
	tokenTransport := &TokenTransport{
		Transport: transport,
		Username:  username,
		Password:  password,
	}
	basicAuthTransport := &BasicTransport{
		Transport: tokenTransport,
		URL:       url,
		Username:  username,
		Password:  password,
	}
	errorTransport := &ErrorTransport{
		Transport: basicAuthTransport,
	}
	return errorTransport
}

// url returns a registry URL with the passed arguements concatenated.
func (r *Registry) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

// Ping tries to contact a registry URL to make sure it is up and accessible.
func (r *Registry) Ping() error {
	url := r.url("/v2/")
	r.Logf("registry.ping url=%s", url)
	resp, err := r.Client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {

	}
	return err
}

type catalogResponse struct {
	Repositories []string `json:"repositories"`
}

// Catalog returns the repositories in a registry.
func (r *Registry) Catalog() ([]string, error) {
	url := r.url("/v2/_catalog")
	r.Logf("registry.catalog url=%s", url)

	var response catalogResponse
	if err := r.getJSON(url, &response); err != nil {
		return nil, err
	}

	return response.Repositories, nil
}

type tagsResponse struct {
	Tags []string `json:"tags"`
}

// Tags returns the tags for a specific repository.
func (r *Registry) Tags(repository string) ([]string, error) {
	url := r.url("/v2/%s/tags/list", repository)
	r.Logf("registry.tags url=%s repository=%s", url, repository)

	var response tagsResponse
	if err := r.getJSON(url, &response); err != nil {
		return nil, err
	}

	return response.Tags, nil
}

func (r *Registry) getJSON(url string, response interface{}) error {
	resp, err := r.Client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(response)
	if err != nil {
		return err
	}

	return nil
}

// DownloadLayer downloads a specific layer by digest for a repository.
func (r *Registry) DownloadLayer(repository string, digest digest.Digest) (io.ReadCloser, error) {
	url := r.url("/v2/%s/blobs/%s", repository, digest)
	r.Logf("registry.layer.download url=%s repository=%s digest=%s", url, repository, digest)

	resp, err := r.Client.Get(url)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// UploadLayer uploads a specific layer by digest for a repository.
func (r *Registry) UploadLayer(repository string, digest digest.Digest, content io.Reader) error {
	uploadURL, err := r.initiateUpload(repository)
	if err != nil {
		return err
	}
	q := uploadURL.Query()
	q.Set("digest", digest.String())
	uploadURL.RawQuery = q.Encode()

	r.Logf("registry.layer.upload url=%s repository=%s digest=%s", uploadURL, repository, digest)

	upload, err := http.NewRequest("PUT", uploadURL.String(), content)
	if err != nil {
		return err
	}
	upload.Header.Set("Content-Type", "application/octet-stream")

	_, err = r.Client.Do(upload)
	return err
}

// HasLayer returns if the registry contains the specific digest for a repository.
func (r *Registry) HasLayer(repository string, digest digest.Digest) (bool, error) {
	checkURL := r.url("/v2/%s/blobs/%s", repository, digest)
	r.Logf("registry.layer.check url=%s repository=%s digest=%s", checkURL, repository, digest)

	resp, err := r.Client.Head(checkURL)
	if err == nil {
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK, nil
	}

	urlErr, ok := err.(*url.Error)
	if !ok {
		return false, err
	}
	httpErr, ok := urlErr.Err.(*httpStatusError)
	if !ok {
		return false, err
	}
	if httpErr.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, err
}

func (r *Registry) initiateUpload(repository string) (*url.URL, error) {
	initiateURL := r.url("/v2/%s/blobs/uploads/", repository)
	r.Logf("registry.layer.initiate-upload url=%s repository=%s", initiateURL, repository)

	resp, err := r.Client.Post(initiateURL, "application/octet-stream", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	locationURL, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	return locationURL, nil
}
