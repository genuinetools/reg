package registry

import (
	"fmt"

	"github.com/peterhellberg/link"
)

type catalogResponse struct {
	Repositories []string `json:"repositories"`
}

// Catalog returns the repositories in a registry.
func (r *Registry) Catalog(u string) ([]string, error) {
	if u == "" {
		u = "/v2/_catalog"
	}
	url := r.url(u)
	r.Logf("registry.catalog url=%s", url)

	var response catalogResponse
	h, err := r.getJSON(url, &response)
	if err != nil {
		return nil, err
	}
	fmt.Printf("header: %#v\n", h)

	for _, l := range link.ParseHeader(h) {
		if l.Rel == "next" {
			repos, err := r.Catalog(l.URI)
			if err != nil {
				return nil, err
			}
			response.Repositories = append(response.Repositories, repos...)
		}
	}

	return response.Repositories, nil
}
