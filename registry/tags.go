package registry

import (
	"context"
	"net/url"

	"github.com/peterhellberg/link"
)

type tagsResponse struct {
	Tags []string `json:"tags"`
}

func (r *Registry) tags(ctx context.Context, u string, repository string) ([]string, error) {
	var uri string
	if u == "" {
		r.Logf("registry.tags url=%s repository=%s", u, repository)
		uri = r.url("/v2/%s/tags/list", repository)
	} else {
		uri = r.url(u)
	}

	var response tagsResponse
	h, err := r.getJSON(ctx, uri, &response)
	if err != nil {
		return nil, err
	}

	for _, l := range link.ParseHeader(h) {
		if l.Rel == "next" {
			unescaped, _ := url.QueryUnescape(l.URI)
			tags, err := r.tags(ctx, unescaped, repository)
			if err != nil {
				return nil, err
			}
			response.Tags = append(response.Tags, tags...)
		}
	}

	return response.Tags, nil
}

// Tags returns the tags for a specific repository.
func (r *Registry) Tags(ctx context.Context, repository string) ([]string, error) {
	return r.tags(ctx, "", repository)
}
