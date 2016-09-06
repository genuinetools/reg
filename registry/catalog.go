package registry

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
