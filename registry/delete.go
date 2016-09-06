package registry

import (
	"fmt"
	"net/http"
)

// Delete removes a repository reference from the registry.
func (r *Registry) Delete(repository, ref string) error {
	url := r.url("/v2/%s/manifests/%s", repository, ref)
	r.Logf("registry.manifests.delete url=%s repository=%s ref=%s", url, repository, ref)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		return nil
	}

	return fmt.Errorf("Got status code: %d", resp.StatusCode)
}
