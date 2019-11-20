package registry

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Pingable returns false for some specific registries that can be never successfuly pinged.
//
// Currently it always returns true.
func (r *Registry) Pingable() bool {
	return true
}

var ErrNoDockerHeader = errors.New("site does not return http(s) header Docker-Distribution-API-Version: registry/2.0")

// Ping tries to contact a registry URL to make sure it is up and it supports Docker v2 Registry Specification.
func (r *Registry) Ping(ctx context.Context) error {
	url := r.url("/v2/")
	r.Logf("registry.ping url=%s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := r.PingClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !strings.HasPrefix(resp.Header.Get("Docker-Distribution-API-Version"), "registry/2.") {
		return ErrNoDockerHeader
	}
	return nil
}
