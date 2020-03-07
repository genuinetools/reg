package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type configObjectBase struct{
	Created		time.Time	`json:"created"`
}

type manifestConfigWrapper struct{
	Config		manifestConfig	`json:"config"`
}

type manifestConfig struct {
	Digest		string	`json:"digest"`
	MediaType	string	`json:"mediaType"`
	Size		int64	`json:"size"`
}

type tagsResponse struct {
	Tags []string `json:"tags"`
}

type Tag struct {
	Name string
	Repository string
	Registry *Registry
}

func (r *Registry) TagNames(ctx context.Context, repository string) ([]string, error) {
	url := r.url("/v2/%s/tags/list", repository)
	r.Logf("registry.tags url=%s repository=%s", url, repository)

	var response tagsResponse
	if _, err := r.getJSON(ctx, url, &response); err != nil {
		return nil, err
	}

	return response.Tags, nil
}

// Tags returns the tags for a specific repository.
func (r *Registry) Tags(ctx context.Context, repository string) ([]Tag, error) {
    tagNames, err := r.TagNames(ctx, repository)
    if err != nil {
		return nil, err
	}

	tags := make([]Tag, len(tagNames))
	for i := range tags {
		tags[i] = Tag {
			Name: tagNames[i],
			Repository: repository,
			Registry: r,
		}
	}
	return tags, nil
}

func (t *Tag) CreatedDate(ctx context.Context) (time.Time, error) {
		// get the manifest - could be in v2 or in OCI format
		m1, err := t.Registry.Manifest(ctx, t.Repository, t.Name)
		if err != nil {
			return time.Time{}, fmt.Errorf("getting manifest for %s:%s failed: %v", t.Repository, t.Name, err)
		}
		var wrapper manifestConfigWrapper
		err = json.Unmarshal(m1, &wrapper)
		if err != nil {
			return time.Time{}, err
		}
		// Grab wrapper.config.digest object from the server
		// From there, we just get the created property from the object
		// https://docker.lerch.org/v2/fauxmo/manifests/1
	    uri := t.Registry.url("/v2/%s/blobs/%s", t.Repository, wrapper.Config.Digest)
		logrus.Infof("fetching config blob at %s", uri)
		var configBase configObjectBase
		if _, err := t.Registry.getBlob(ctx, uri, &configBase); err != nil {
			return time.Time{}, err
		}
		logrus.Infof("Got time %v", configBase.Created)
		return configBase.Created, nil
}
