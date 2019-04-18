package registry

import (
	"context"
	"net/http"
	"strings"
)

// Pingable checks pingable
func (r *Registry) Pingable() bool {
	// Currently *.gcr.io/v2 can't be ping if users have each projects auth
	return !strings.HasSuffix(r.URL, "gcr.io")
}

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
