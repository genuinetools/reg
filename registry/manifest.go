package registry

import (
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
)

// Manifest returns the manifest for a specific repository:tag.
func (r *Registry) Manifest(repository, ref string) (interface{}, error) {
	url := r.url("/v2/%s/manifests/%s", repository, ref)
	r.Logf("registry.manifests url=%s repository=%s ref=%s", url, repository, ref)

	var m schema2.Manifest
	if err := r.getJSON(url, &m); err != nil {
		return m, err
	}

	if m.Versioned.SchemaVersion == 1 {
		return r.v1Manifest(repository, ref)
	}

	return m, nil
}

func (r *Registry) v1Manifest(repository, ref string) (schema1.SignedManifest, error) {
	url := r.url("/v2/%s/manifests/%s", repository, ref)
	r.Logf("registry.manifests url=%s repository=%s ref=%s", url, repository, ref)

	var m schema1.SignedManifest
	if err := r.getJSON(url, &m); err != nil {
		return m, err
	}

	return m, nil
}
