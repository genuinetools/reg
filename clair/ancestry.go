package clair

import (
	"context"
	"errors"

	"github.com/coreos/clair/api/v3/clairpb"
)

// GetAncestry displays an ancestry and optionally all of its features and vulnerabilities.
func (c *Clair) GetAncestry(name string, features, vulnerabilities bool) (*clairpb.GetAncestryResponse_Ancestry, error) {
	c.Logf("clair.ancestry.get name=%s", name)

	client := clairpb.NewAncestryServiceClient(c.grpcConn)

	resp, err := client.GetAncestry(context.Background(), &clairpb.GetAncestryRequest{
		AncestryName:        name,
		WithVulnerabilities: vulnerabilities,
		WithFeatures:        features,
	})
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, errors.New("ancestry response was nil")
	}

	if resp.GetStatus() != nil {
		c.Logf("clair.ancestry.get ClairStatus=%#v", *resp.GetStatus())
	}

	return resp.GetAncestry(), nil
}

// PostAncestry performs the analysis of all layers from the provided path.
func (c *Clair) PostAncestry(name string, layers []*clairpb.PostAncestryRequest_PostLayer) error {
	c.Logf("clair.ancestry.post name=%s", name)

	client := clairpb.NewAncestryServiceClient(c.grpcConn)

	resp, err := client.PostAncestry(context.Background(), &clairpb.PostAncestryRequest{
		AncestryName: name,
		Layers:       layers,
		Format:       "Docker",
	})
	if err != nil {
		return err
	}

	if resp == nil {
		return errors.New("ancestry response was nil")
	}

	if resp.GetStatus() != nil {
		c.Logf("clair.ancestry.post ClairStatus=%#v", *resp.GetStatus())
	}

	return nil
}
