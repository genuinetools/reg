package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/docker/docker/client"
	"github.com/jessfraz/reg/testutils"
)

var (
	exeSuffix    string // ".exe" on Windows
	registryAddr string
)

func init() {
	switch runtime.GOOS {
	case "windows":
		exeSuffix = ".exe"
	}
}

// The TestMain function creates a reg command for testing purposes and
// deletes it after the tests have been run.
// It also spins up a local registry prefilled with an alpine image and
// removes that after the tests have been run.
func TestMain(m *testing.M) {
	// build the test binary
	args := []string{"build", "-o", "testreg" + exeSuffix}
	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "building testreg failed: %v\n%s", err, out)
		os.Exit(2)
	}

	// create the docker client
	dcli, err := client.NewEnvClient()
	if err != nil {
		panic(fmt.Errorf("could not connect to docker: %v", err))
	}

	// start registry
	regID, addr, err := testutils.StartRegistry(dcli)
	if err != nil {
		testutils.RemoveContainer(dcli, regID)
		panic(fmt.Errorf("starting registry container failed: %v", err))
	}
	registryAddr = addr

	flag.Parse()
	merr := m.Run()

	// remove registry
	if err := testutils.RemoveContainer(dcli, regID); err != nil {
		log.Printf("couldn't remove registry container: %v", err)
	}

	// remove test binary
	os.Remove("testreg" + exeSuffix)

	os.Exit(merr)
}

func run(args ...string) (string, error) {
	prog := "./testreg" + exeSuffix
	// always add trust insecure, and the registry
	newargs := append([]string{"-k", "-r", "localhost:5000"}, args...)
	cmd := exec.Command(prog, newargs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestList(t *testing.T) {
	out, err := run("ls")
	if err != nil {
		t.Fatal(err)
	}
	expected := `Repositories for localhost:5000
REPO                TAGS
alpine              latest
`
	if out != expected {
		t.Fatalf("expected: %s\ngot: %s", expected, out)
	}
}
