package trivy

// Error describes the structure of a clair error.
type Error struct {
	Message string `json:"Message,omitempty"`
}

// Vulnerability represents vulnerability entity returned by Clair.
type Vulnerability struct {
	Name          string                 `json:"Name,omitempty"`
	NamespaceName string                 `json:"NamespaceName,omitempty"`
	Description   string                 `json:"Description,omitempty"`
	Link          string                 `json:"Link,omitempty"`
	Severity      string                 `json:"Severity,omitempty"`
	Metadata      map[string]interface{} `json:"Metadata,omitempty"`
	FixedBy       string                 `json:"FixedBy,omitempty"`
	FixedIn       []feature              `json:"FixedIn,omitempty"`
}

// VulnerabilityReport represents the result of a vulnerability scan of a repo.
type VulnerabilityReport struct {
	Name            string
	RegistryURL     string
	Repo            string
	Tag             string
	Date            string
	Vulns           []Vulnerability
	VulnsBySeverity map[string][]Vulnerability
	BadVulns        int
}
type feature struct {
	Name            string          `json:"Name,omitempty"`
	NamespaceName   string          `json:"NamespaceName,omitempty"`
	VersionFormat   string          `json:"VersionFormat,omitempty"`
	Version         string          `json:"Version,omitempty"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities,omitempty"`
	AddedBy         string          `json:"AddedBy,omitempty"`
}
