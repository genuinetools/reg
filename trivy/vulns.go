package trivy

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/genuinetools/reg/registry"
	"github.com/sirupsen/logrus"
)
type TrivyVulnerabilityReport []struct {
	Target          string               `json:"Target",omitempty`
	Type            string               `json:"Type",omitempty`
	Vulnerabilities []TrivyVulnerability `json:"Vulnerabilities",omitempty`
}
type Layer struct {
	Digest string `json:"Digest",omitempty`
	DiffID string `json:"DiffID",omitempty`
}
type TrivyVulnerability struct {
	VulnerabilityID  string                           `json:"VulnerabilityID",omitempty`
	PkgName          string                           `json:"PkgName",omitempty`
	InstalledVersion string                           `json:"InstalledVersion",omitempty`
	FixedVersion     string                           `json:"FixedVersion",omitempty`
	Layer            Layer                            `json:"Layer",omitempty`
	SeveritySource   string                           `json:"SeveritySource",omitempty`
	Title            string                           `json:"Title",omitempty`
	Description      string                           `json:"Description",omitempty`
	Severity         string                           `json:"Severity",omitempty`
	VendorVectors    map[string]map[string]string     `json:"VendorVectors",omitempty`
	References       []string                         `json:"References",omitempty`
}

// Vulnerabilities scans the given repo and tag using trivy
func (c *Trivy) ScanImage(ctx context.Context, r *registry.Registry, repo, tag string) (interface{}, error) {
	var trivyReport TrivyVulnerabilityReport
	report := VulnerabilityReport{
		RegistryURL:     r.Domain,
		Repo:            repo,
		Tag:             tag,
		Date:            time.Now().Local().Format(time.RFC1123),
		VulnsBySeverity: make(map[string][]Vulnerability),
	}
	// TODO: Figure out what we should do with the image cache. We
	//       don't want to clear it each time, but "latest" image is a real
	//       issue here
	cmd := exec.Command(c.Location, "-q", "-f", "json", fmt.Sprintf("%s/%s:%s", r.Domain, repo, tag))
	logrus.Infof("%v: trivy scan starting: %s/%s:%s", time.Now().Format("2006-01-02 15:04:05"), r.Domain, repo, tag)
	output, err := cmd.Output()
	logrus.Infof("%v: trivy scan complete: %s/%s:%s", time.Now().Format("2006-01-02 15:04:05"), r.Domain, repo, tag)
	if err != nil {
		return report, err
	}
	err = json.Unmarshal(output, &trivyReport)
	if err != nil {
		return report, err
	}
	if (len(trivyReport) > 0) {
		c.Logf("trivy.ScanImage %d targets found for %s/%s:%s", len(trivyReport), r.Domain, repo, tag)
		lgth := 0
		for _, v := range trivyReport {
			lgth += len(v.Vulnerabilities)
		}
		alltitles := make(map[string]bool, lgth)
		for i, _ := range trivyReport {
			err = c.copyFromTrivyToRegReport(trivyReport, i,  &report, alltitles)
			if err != nil {
				// what else would we do with this?
				c.Logf("trivy.ScanImage got error from copyFromTrivyToRegReport: %v", err)
			}
		}
		for _, v := range report.Vulns {
			vulns := report.VulnsBySeverity[v.Severity]
			report.VulnsBySeverity[v.Severity] = append(vulns, v)
		}
		// Clair defines this with Pascal-cased sev names like "high", "critical" and "defcon1"
		// Trivy has HIGH/CRITICAL only. We converted these to Pascal case
		// (see toVulnerability) to match
		report.BadVulns = len(report.VulnsBySeverity["High"]) + len(report.VulnsBySeverity["Critical"])
	}
	return report, nil
}

func (c *Trivy) copyFromTrivyToRegReport(tReport TrivyVulnerabilityReport, inx int, report *VulnerabilityReport, existing map[string]bool) (error) {
	report.Name = tReport[inx].Target

	for _, v := range tReport[inx].Vulnerabilities {
		vuln := v.toVulnerability()
		if !existing[vuln.Name] {
			existing[vuln.Name] = true
			report.Vulns = append(report.Vulns, vuln)
		}
	}
	return nil
}

func (v *TrivyVulnerability) toVulnerability() (Vulnerability) {
	link := ""
	if len(v.References) > 0 {
		link = v.References[0] // TODO: This really should be the whole list - would need changes in clair, the return struct and templates
	}
    var rc = Vulnerability{
		Name : v.VulnerabilityID, // This seems right. Could also be Title prop
		NamespaceName : v.PkgName,
		Description : v.Description,
		Link : link,
		Severity : severity(v.Severity),
		FixedIn : []feature{
			feature {
				Name : v.PkgName,
				Version : v.FixedVersion,
			},
		},
	}
	return rc
}

func severity(sev string) (string) {
	switch sev {
	case "CRITICAL":
		return "Critical"
	case "HIGH":
		return "High"
	case "MEDIUM":
		return "Medium"
	case "LOW":
		return "Low"
	case "UNKNOWN":
		return "Unknown"
	default:
		return sev
	}
}
