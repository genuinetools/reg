package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestRemove(t *testing.T) {
	// Make sure we have busybox in list.
	out, err := run("ls", domain)
	if err != nil {
		t.Fatalf("output: %s, error: %v", out, err)
	}
	expected := `REPO                TAGS
alpine              3.5, latest
busybox             glibc, latest, musl`
	if !strings.HasSuffix(strings.TrimSpace(out), expected) {
		t.Fatalf("expected to contain: %s\ngot: %s", expected, out)
	}

	// Remove busybox image.
	if out, err := run("rm", fmt.Sprintf("%s/busybox:glibc", domain)); err != nil {
		t.Fatalf("output: %s, error: %v", out, err)
	}

}
