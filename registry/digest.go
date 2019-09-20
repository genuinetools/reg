package registry

import (
	"context"
	"fmt"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/opencontainers/go-digest"
)

// Digest returns the digest for an image.
func (r *Registry) Digest(ctx context.Context, image Image, mediatypes ...string) (digest.Digest, error) {
	if len(image.Digest) > 1 {
		// return early if we already have an image digest.
		return image.Digest, nil
	}

	url := r.url("/v2/%s/manifests/%s", image.Path, image.Tag)
	r.Logf("registry.manifests.get url=%s repository=%s ref=%s",
		url, image.Path, image.Tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	if mediatypes == nil {
		mediatypes = []string{schema2.MediaTypeManifest}
	}
	logrus.Debugf("Using media types %s", mediatypes)
	req.Header.Add("Accept", strings.Join(mediatypes, ", "))

	resp, err := r.Client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return "", fmt.Errorf("got status code: %d", resp.StatusCode)
	}

	return digest.Parse(resp.Header.Get("Docker-Content-Digest"))
}
