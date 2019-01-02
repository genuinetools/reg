package clair

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetLayer displays a Layer and optionally all of its features and vulnerabilities.
func (c *Clair) GetLayer(ctx context.Context, name string, features, vulnerabilities bool) (*Layer, error) {
	url := c.url("/v1/layers/%s?features=%t&vulnerabilities=%t", name, features, vulnerabilities)
	c.Logf("clair.layers.get url=%s name=%s", url, name)

	var respLayer layerEnvelope
	if _, err := c.getJSON(ctx, url, &respLayer); err != nil {
		return nil, err
	}

	if respLayer.Error != nil {
		return nil, fmt.Errorf("clair error: %s", respLayer.Error.Message)
	}

	return respLayer.Layer, nil
}

// PostLayer performs the analysis of a Layer from the provided path.
func (c *Clair) PostLayer(ctx context.Context, layer *Layer) (*Layer, error) {
	url := c.url("/v1/layers")
	c.Logf("clair.layers.post url=%s name=%s", url, layer.Name)

	b, err := json.Marshal(layerEnvelope{Layer: layer})
	if err != nil {
		return nil, err
	}

	c.Logf("clair.layers.post req.Body=%s", string(b))

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.Logf("clair.layers.post resp.Status=%s", resp.Status)

	var respLayer layerEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&respLayer); err != nil {
		return nil, err
	}

	if respLayer.Error != nil {
		return nil, fmt.Errorf("clair error: %s", respLayer.Error.Message)
	}

	return respLayer.Layer, err
}

// DeleteLayer removes a layer reference from clair.
func (c *Clair) DeleteLayer(ctx context.Context, name string) error {
	url := c.url("/v1/layers/%s", name)
	c.Logf("clair.layers.delete url=%s name=%s", url, name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.Logf("clair.clair resp.Status=%s", resp.Status)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return fmt.Errorf("got status code: %d", resp.StatusCode)
}
