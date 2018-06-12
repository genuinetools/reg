package main

import (
	"strings"
	"testing"
)

func TestVulns(t *testing.T) {
	out, err := run("vulns", "--clair", "http://localhost:6060", "alpine:3.5")
	if err != nil {
		t.Fatalf("output: %s, error: %v", string(out), err)
	}

	expected := `clair.clair resp.Status=200 OK`
	if !strings.HasSuffix(strings.TrimSpace(out), expected) {
		t.Logf("expected: %s\ngot: %s", expected, out)
	}
}
