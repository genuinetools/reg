package main

import "testing"

func TestList(t *testing.T) {
	out, err := run("ls")
	if err != nil {
		t.Fatalf("output: %s, error: %v", string(out), err)
	}
	expected := `Repositories for localhost:5000
REPO                TAGS
alpine              latest
`
	if out != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, out)
	}
}
