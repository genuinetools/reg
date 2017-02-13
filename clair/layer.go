package clair

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetLayer displays a Layer and optionally all of its features and vulnerabilities.
func (c *Clair) GetLayer(name string, features, vulnerabilities bool) (layer Layer, err error) {
	url := c.url("/v1/layers/%s?features=%t&vulnerabilities=%t", name, features, vulnerabilities)
	c.Logf("clair.layers.get url=%s name=%s", url, name)

	if _, err := c.getJSON(url, &layer); err != nil {
		return layer, err
	}

	return layer, nil
}

// PostLayer performs the analysis of a Layer from the provided path.
func (c *Clair) PostLayer(layer Layer) (respLayer Layer, err error) {
	url := c.url("/v1/layers")
	c.Logf("clair.layers.post url=%s name=%s", url, layer.Name)

	b, err := json.Marshal(layer)
	if err != nil {
		return respLayer, err
	}

	resp, err := c.Client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return respLayer, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&respLayer); err != nil {
		return respLayer, err
	}

	return respLayer, err
}

// DeleteLayer removes a layer reference from clair.
func (c *Clair) DeleteLayer(name string) error {
	url := c.url("/v1/layers/%s", name)
	c.Logf("clair.layers.delete url=%s name=%s", url, name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	return fmt.Errorf("Got status code: %d", resp.StatusCode)
}
