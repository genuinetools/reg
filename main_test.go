package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/jessfraz/reg/testutils"
)

var (
	exeSuffix string // ".exe" on Windows

	registryConfigs = []struct {
		config   string
		username string
		password string
	}{
		{
			config:   "noauth.yml",
			username: "blah",
			password: "blah",
		},
		{
			config:   "basicauth.yml",
			username: "admin",
			password: "testing",
		},
	}
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
	// remove test binary
	defer os.Remove("testreg" + exeSuffix)

	// create the docker client
	dcli, err := newEnvDockerClient()
	if err != nil {
		panic(fmt.Errorf("could not connect to docker: %v", err))
	}

	for _, regConfig := range registryConfigs {
		// start each registry
		regID, _, err := testutils.StartRegistry(dcli, regConfig.config, regConfig.username, regConfig.password)
		if err != nil {
			testutils.RemoveContainer(dcli, regID)
			panic(fmt.Errorf("starting registry container %s failed: %v", regConfig.config, err))
		}

		flag.Parse()
		merr := m.Run()

		// remove registry
		if err := testutils.RemoveContainer(dcli, regID); err != nil {
			log.Printf("couldn't remove registry container %s: %v", regConfig.config, err)
		}

		if merr != 0 {
			fmt.Printf("testing config %s failed\n", regConfig.config)
			os.Exit(merr)
		}
	}

	os.Exit(0)
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

func newEnvDockerClient() (*client.Client, error) {
	var hc *http.Client

	if dockerCertPath := os.Getenv("DOCKER_CERT_PATH"); dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "cacert.pem"),
			CertFile:           filepath.Join(dockerCertPath, "server.cert"),
			KeyFile:            filepath.Join(dockerCertPath, "server.key"),
			InsecureSkipVerify: os.Getenv("DOCKER_TLS_VERIFY") == "",
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		hc = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
			CheckRedirect: client.CheckRedirect,
		}
	}

	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = client.DefaultDockerHost
	}
	version := os.Getenv("DOCKER_API_VERSION")
	if version == "" {
		version = api.DefaultVersion
	}

	cli, err := client.NewClient(host, version, hc, nil)
	if err != nil {
		return cli, err
	}
	return cli, nil
}
