package clair

import (
	"fmt"
	"strings"

	"github.com/coreos/clair/api/v3/clairpb"
	"github.com/docker/distribution"
	"github.com/genuinetools/reg/registry"
)

// NewClairLayer will form a layer struct required for a clair scan.
func (c *Clair) NewClairLayer(r *registry.Registry, image string, fsLayers []distribution.Descriptor, index int) (*Layer, error) {
	var parentName string
	if index < len(fsLayers)-1 {
		parentName = fsLayers[index+1].Digest.String()
	}

	// Form the path.
	p := strings.Join([]string{r.URL, "v2", image, "blobs", fsLayers[index].Digest.String()}, "/")

	// Get the headers.
	h, err := r.Headers(p)
	if err != nil {
		return nil, err
	}

	return &Layer{
		Name:       fsLayers[index].Digest.String(),
		Path:       p,
		ParentName: parentName,
		Format:     "Docker",
		Headers:    h,
	}, nil
}

// NewClairV3Layer will form a layer struct required for a clair scan.
func (c *Clair) NewClairV3Layer(r *registry.Registry, image string, fsLayer distribution.Descriptor) (*clairpb.PostAncestryRequest_PostLayer, error) {
	// Form the path.
	p := strings.Join([]string{r.URL, "v2", image, "blobs", fsLayer.Digest.String()}, "/")

	// Get the headers.
	h, err := r.Headers(p)
	if err != nil {
		return nil, err
	}

	return &clairpb.PostAncestryRequest_PostLayer{
		Hash:    fsLayer.Digest.String(),
		Path:    p,
		Headers: h,
	}, nil
}

func (c *Clair) getFilteredLayers(r *registry.Registry, repo, tag string) ([]distribution.Descriptor, error) {
	ok := true
	// Get the manifest to pass to clair.
	mf, err := r.ManifestV2(repo, tag)
	if err != nil {
		ok = false
		c.Logf("couldn't retrieve manifest v2, falling back to v1")
		//	return nil, fmt.Errorf("getting the v2 manifest for %s:%s failed: %v", repo, tag, err)
	}

	var filteredLayers []distribution.Descriptor

	// Filter out the empty layers.
	if ok {
		for _, layer := range mf.Layers {
			if !IsEmptyLayer(layer.Digest) {
				filteredLayers = append(filteredLayers, layer)
			}
		}
		return filteredLayers, nil
	}

	m, err := r.ManifestV1(repo, tag)
	if err != nil {
		return nil, fmt.Errorf("getting the v1 manifest for %s:%s failed: %v", repo, tag, err)
	}

	for _, layer := range m.FSLayers {
		if !IsEmptyLayer(layer.BlobSum) {
			newLayer := distribution.Descriptor{
				Digest: layer.BlobSum,
			}

			filteredLayers = append(filteredLayers, newLayer)
		}
	}

	return filteredLayers, nil
}
