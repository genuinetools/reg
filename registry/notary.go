package registry

// NotaryTimestamp returns a notary timestamp for a specific repository:tag.
func (r *Registry) NotaryTimestamp(repository, ref string) (interface{}, error) {
	url := r.url("/v2/%s/%s/_trust/tuf/timestamp.json", r.Domain, repository)
	r.Logf("registry.manifests url=%s repository=%s ref=%s", url, repository, ref)

	var ts interface{}
	_, err := r.getJSON(url, &ts)
	if err != nil {
		return ts, err
	}

	return ts, nil
}
