package main

import (
	"strings"
	"testing"
)

func TestDelete(t *testing.T) {
	// Make sure we have busybox in list.
	out, err := run("ls")
	if err != nil {
		t.Fatalf("output: %s, error: %v", string(out), err)
	}
	expected := `REPO                TAGS
busybox             glibc, musl, latest
alpine              latest
`
	if !strings.HasSuffix(out, expected) {
		t.Fatalf("expected: %s\ngot: %s", expected, out)
	}

	// Remove busybox image.
	if out, err := run("rm", "busybox"); err != nil {
		t.Fatalf("output: %s, error: %v", string(out), err)
	}

	// Make sure there is no busybox in list.
	out, err = run("ls")
	if err != nil {
		t.Fatalf("output: %s, error: %v", string(out), err)
	}
	expected = `REPO                TAGS
busybox             glibc, musl
alpine              latest
`
	if !strings.HasSuffix(out, expected) {
		t.Fatalf("expected: %s\ngot: %s", expected, out)
	}
}
