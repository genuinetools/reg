package clair

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/jessfraz/reg/registry"
)

func (c *Clair) Vulnerabilities(r *registry.Registry, repo, tag string, m schema1.SignedManifest) (VulnerabilityReport, error) {
	report := VulnerabilityReport{
		RegistryURL:     r.Domain,
		Repo:            repo,
		Tag:             tag,
		Date:            time.Now().Local().Format(time.RFC1123),
		VulnsBySeverity: make(map[string][]Vulnerability),
	}

	// filter out the empty layers
	var filteredLayers []schema1.FSLayer
	for _, layer := range m.FSLayers {
		if layer.BlobSum != EmptyLayerBlobSum {
			filteredLayers = append(filteredLayers, layer)
		}
	}
	m.FSLayers = filteredLayers
	if len(m.FSLayers) == 0 {
		fmt.Printf("No need to analyse image %s:%s as there is no non-emtpy layer", repo, tag)
		return report, nil
	}

	for i := len(m.FSLayers) - 1; i >= 0; i-- {
		// form the clair layer
		l, err := c.NewClairLayer(r, repo, m.FSLayers, i)
		if err != nil {
			return report, err
		}

		// post the layer
		if _, err := c.PostLayer(l); err != nil {
			return report, err
		}
	}

	vl, err := c.GetLayer(m.FSLayers[0].BlobSum.String(), false, true)
	if err != nil {
		return report, err
	}

	// get the vulns
	for _, f := range vl.Features {
		for _, v := range f.Vulnerabilities {
			report.Vulns = append(report.Vulns, v)
		}
	}

	vulnsBy := func(sev string, store map[string][]Vulnerability) []Vulnerability {
		items, found := store[sev]
		if !found {
			items = make([]Vulnerability, 0)
			store[sev] = items
		}
		return items
	}

	// group by severity
	for _, v := range report.Vulns {
		sevRow := vulnsBy(v.Severity, report.VulnsBySeverity)
		report.VulnsBySeverity[v.Severity] = append(sevRow, v)
	}

	// calculate number of bad vulns
	report.BadVulns = len(report.VulnsBySeverity["High"]) + len(report.VulnsBySeverity["Critical"]) + len(report.VulnsBySeverity["Defcon1"])

	return report, nil
}

func (c *Clair) NewClairLayer(r *registry.Registry, image string, fsLayers []schema1.FSLayer, index int) (*Layer, error) {
	var parentName string
	if index < len(fsLayers)-1 {
		parentName = fsLayers[index+1].BlobSum.String()
	}

	// form the path
	p := strings.Join([]string{r.URL, "v2", image, "blobs", fsLayers[index].BlobSum.String()}, "/")

	// get the token
	token, err := r.Token(p)
	if err != nil {
		return nil, err
	}

	h := make(map[string]string)
	if token != "" {
		h = map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", token),
		}
	}

	return &Layer{
		Name:       fsLayers[index].BlobSum.String(),
		Path:       p,
		ParentName: parentName,
		Format:     "Docker",
		Headers:    h,
	}, nil
}
