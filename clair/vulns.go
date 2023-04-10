package clair

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/clair/api/v3/clairpb"
	"github.com/genuinetools/reg/registry"
)

func (c *Clair) ScanImage(ctx context.Context, r *registry.Registry, repo, tag string) (interface{}, error) {
	result, err := c.VulnerabilitiesV3(ctx, r, repo, tag)
	if err != nil {
		// Fallback to Clair v2 API.
		result, err = c.Vulnerabilities(ctx, r, repo, tag)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

// Vulnerabilities scans the given repo and tag.
func (c *Clair) Vulnerabilities(ctx context.Context, r *registry.Registry, repo, tag string) (VulnerabilityReport, error) {
	report := VulnerabilityReport{
		RegistryURL:     r.Domain,
		Repo:            repo,
		Tag:             tag,
		Date:            time.Now().Local().Format(time.RFC1123),
		VulnsBySeverity: make(map[string][]Vulnerability),
	}

	filteredLayers, _, err := c.getLayers(ctx, r, repo, tag, true)
	if err != nil {
		return report, fmt.Errorf("getting filtered layers failed: %v", err)
	}

	if len(filteredLayers) == 0 {
		fmt.Printf("No need to analyse image %s:%s as there is no non-empty layer", repo, tag)
		return report, nil
	}

	for i := len(filteredLayers) - 1; i >= 0; i-- {
		// Form the clair layer.
		l, err := c.NewClairLayer(ctx, r, repo, filteredLayers, i)
		if err != nil {
			return report, err
		}

		// Post the layer.
		if _, err := c.PostLayer(ctx, l); err != nil {
			return report, err
		}
	}

	report.Name = filteredLayers[0].Digest.String()

	vl, err := c.GetLayer(ctx, filteredLayers[0].Digest.String(), true, true)
	if err != nil {
		return report, err
	}

	// Get the vulns.
	for _, f := range vl.Features {
		report.Vulns = append(report.Vulns, f.Vulnerabilities...)
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

// VulnerabilitiesV3 scans the given repo and tag using the clair v3 API.
func (c *Clair) VulnerabilitiesV3(ctx context.Context, r *registry.Registry, repo, tag string) (VulnerabilityReport, error) {
	report := VulnerabilityReport{
		RegistryURL:     r.Domain,
		Repo:            repo,
		Tag:             tag,
		Date:            time.Now().Local().Format(time.RFC1123),
		VulnsBySeverity: make(map[string][]Vulnerability),
	}

	layers, reportName, err := c.getLayers(ctx, r, repo, tag, false)
	if err != nil {
		return report, fmt.Errorf("getting filtered layers failed: %v", err)
	}

	if len(layers) == 0 {
		fmt.Printf("No need to analyse image %s:%s as there is no non-empty layer", repo, tag)
		return report, nil
	}

	report.Name = reportName

	clairLayers := []*clairpb.PostAncestryRequest_PostLayer{}
	for i := len(layers) - 1; i >= 0; i-- {
		// Form the clair layer.
		l, err := c.NewClairV3Layer(ctx, r, repo, layers[i])
		if err != nil {
			return report, err
		}

		// Append the layer.
		clairLayers = append(clairLayers, l)
	}

	// Post the ancestry.
	if err := c.PostAncestry(ctx, reportName, clairLayers); err != nil {
		return report, fmt.Errorf("posting ancestry failed: %v", err)
	}

	// Get the ancestry.
	vl, err := c.GetAncestry(ctx, reportName)
	if err != nil {
		return report, err
	}

	if vl == nil {
		return report, errors.New("ancestry response was nil")
	}

	// Get the vulns.
	for _, l := range vl.GetLayers() {
		for _, f := range l.GetDetectedFeatures() {
			for _, v := range f.GetVulnerabilities() {
				report.Vulns = append(report.Vulns, Vulnerability{
					Name:          v.Name,
					NamespaceName: v.NamespaceName,
					Description:   v.Description,
					Link:          v.Link,
					Severity:      v.Severity,
					Metadata:      map[string]interface{}{v.Metadata: ""},
					FixedBy:       v.FixedBy,
				})
			}
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

	// Group by severity.
	for _, v := range report.Vulns {
		sevRow := vulnsBy(v.Severity, report.VulnsBySeverity)
		report.VulnsBySeverity[v.Severity] = append(sevRow, v)
	}

	// calculate number of bad vulns
	report.BadVulns = len(report.VulnsBySeverity["High"]) + len(report.VulnsBySeverity["Critical"]) + len(report.VulnsBySeverity["Defcon1"])

	return report, nil
}
