package registry

import (
	"context"
	"net/http"
)

// Ping tries to contact a registry URL to make sure it is up and accessible.
func (r *Registry) Ping(ctx context.Context) error {
	url := r.url("/v2/")
	r.Logf("registry.ping url=%s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := r.Client.Do(req.WithContext(ctx))
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
