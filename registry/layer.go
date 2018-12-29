package registry

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"fmt"

	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
)

// DownloadLayer downloads a specific layer by digest for a repository.
func (r *Registry) DownloadLayer(ctx context.Context, repository string, digest digest.Digest) (io.ReadCloser, error) {
	url := r.url("/v2/%s/blobs/%s", repository, digest)
	r.Logf("registry.layer.download url=%s repository=%s digest=%s", url, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// UploadLayer uploads a specific layer by digest for a repository.
func (r *Registry) UploadLayer(ctx context.Context, repository string, digest reference.Reference, content io.Reader) error {
	uploadURL, token, err := r.initiateUpload(ctx, repository)
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
	upload.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	_, err = r.Client.Do(upload.WithContext(ctx))
	return err
}

// HasLayer returns if the registry contains the specific digest for a repository.
func (r *Registry) HasLayer(ctx context.Context, repository string, digest digest.Digest) (bool, error) {
	checkURL := r.url("/v2/%s/blobs/%s", repository, digest)
	r.Logf("registry.layer.check url=%s repository=%s digest=%s", checkURL, repository, digest)

	req, err := http.NewRequest("HEAD", checkURL, nil)
	if err != nil {
		return false, err
	}
	resp, err := r.Client.Do(req.WithContext(ctx))
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

func (r *Registry) initiateUpload(ctx context.Context, repository string) (*url.URL, string, error) {
	initiateURL := r.url("/v2/%s/blobs/uploads/", repository)
	r.Logf("registry.layer.initiate-upload url=%s repository=%s", initiateURL, repository)

	req, err := http.NewRequest("POST", initiateURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := r.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, "", err
	}
	token := resp.Header.Get("Request-Token")
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	locationURL, err := url.Parse(location)
	if err != nil {
		return nil, token, err
	}
	return locationURL, token, nil
}
