package clair

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/coreos/clair/api/v3/clairpb"
)

// GetAncestry displays an ancestry and optionally all of its features and vulnerabilities.
func (c *Clair) GetAncestry(name string, features, vulnerabilities bool) (*clairpb.Ancestry, error) {
	url := c.url("/v3/ancestry")
	c.Logf("clair.ancestry.get url=%s name=%s", url, name)

	b, err := json.Marshal(clairpb.GetAncestryRequest{AncestryName: name, WithVulnerabilities: vulnerabilities, WithFeatures: features})
	if err != nil {
		return nil, err
	}

	c.Logf("clair.ancestry.get req.Body=%s", string(b))

	req, err := http.NewRequest("GET", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.Logf("clair.ancestry.get resp.Status=%s", resp.Status)

	var aResp clairpb.GetAncestryResponse
	if err := json.NewDecoder(resp.Body).Decode(&aResp); err != nil {
		return nil, err
	}

	if aResp.GetStatus() != nil {
		c.Logf("clair.ancestry.get ClairStatus=%#v", *aResp.GetStatus())
	}

	return aResp.GetAncestry(), nil
}

// PostAncestry performs the analysis of all layers from the provided path.
func (c *Clair) PostAncestry(name string, layers []*clairpb.PostAncestryRequest_PostLayer) error {
	url := c.url("/v3/ancestry")
	c.Logf("clair.ancestry.post url=%s name=%s", url, name)

	b, err := json.Marshal(clairpb.PostAncestryRequest{AncestryName: name, Layers: layers})
	if err != nil {
		return err
	}

	c.Logf("clair.ancestry.post req.Body=%s", string(b))

	resp, err := c.Client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.Logf("clair.ancestry.post resp.Status=%s", resp.Status)

	var aResp clairpb.PostAncestryResponse
	if err := json.NewDecoder(resp.Body).Decode(&aResp); err != nil {
		return err
	}

	if aResp.GetStatus() != nil {
		c.Logf("clair.ancestry.post ClairStatus=%#v", *aResp.GetStatus())
	}

	return nil
}
