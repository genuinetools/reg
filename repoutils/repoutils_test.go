package repoutils

import (
	"testing"

	"github.com/docker/distribution/reference"
)

func TestGetRepoAndRef(t *testing.T) {
	imageTestcases := []struct {
		// input is the repository name or name component testcase
		input string
		// err is the error expected from Parse, or nil
		err error
		// repository is the string representation for the reference
		repository string
		// ref the reference
		ref string
	}{
		{
			input:      "alpine",
			repository: "alpine",
			ref:        "latest",
		},
		{
			input:      "docker:dind",
			repository: "docker",
			ref:        "dind",
		},
		{
			input: "",
			err:   reference.ErrNameEmpty,
		},
		{
			input:      "chrome@sha256:2a6c8ad38c41ae5122d76be59b34893d7fa1bdfaddd85bf0e57d0d16c0f7f91e",
			repository: "chrome",
			ref:        "sha256:2a6c8ad38c41ae5122d76be59b34893d7fa1bdfaddd85bf0e57d0d16c0f7f91e",
		},
	}

	for _, testcase := range imageTestcases {
		repo, ref, err := GetRepoAndRef(testcase.input)
		if err != nil {
			if err.Error() != testcase.err.Error() {
				t.Fatalf("%q: expected err (%v), got err (%v)", testcase.input, testcase.err, err)
			}
			continue
		}

		if testcase.repository != repo {
			t.Fatalf("%q: expected repo (%s), got repo (%s)", testcase.input, testcase.repository, repo)
		}

		if testcase.ref != ref {
			t.Fatalf("%q: expected ref (%s), got ref (%s)", testcase.input, testcase.ref, ref)
		}
	}
}
