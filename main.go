package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/cliconfig"
	"github.com/jfrazelle/junk/reg/registry"
)

const (
	// BANNER is what is printed for help/info output
	BANNER = ` _ __ ___  __ _
| '__/ _ \/ _` + "`" + ` |
| | |  __/ (_| |
|_|  \___|\__, |
          |___/

 Docker registry v2 client.
 Version: %s

`
	// VERSION is the binary version.
	VERSION = "v0.1.0"

	dockerConfigPath = ".docker/config.json"
)

var (
	registryURL string
	username    string
	password    string

	debug   bool
	version bool
)

func init() {
	// Parse flags
	flag.StringVar(&registryURL, "r", "", "Url to the private registry (ex. https://registry.jess.co)")
	flag.StringVar(&username, "u", "", "Username for the registry")
	flag.StringVar(&password, "p", "", "Password for the registry")
	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&version, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&debug, "d", false, "run in debug mode")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(BANNER, VERSION))
		flag.PrintDefaults()
	}

	flag.Parse()

	if version {
		fmt.Printf("%s", VERSION)
		os.Exit(0)
	}

	// Set log level
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func main() {
	// try to read the docker config, if they did not pass
	// a registry URL, username, or password
	if registryURL == "" || username == "" || password == "" {
		if err := readDockerConfig(); err != nil {
			logrus.Fatal(err)
		}
	}

	// create the registry client
	r, err := registry.New(registryURL, username, password, debug)
	if err != nil {
		logrus.Fatal(err)
	}

	// get the repositories via catalog
	repos, err := r.Catalog()
	if err != nil {
		logrus.Fatal(err)
	}

	fmt.Printf("Repositories for %s\n", registryURL)

	// setup the tab writer
	w := tabwriter.NewWriter(os.Stdout, 20, 1, 3, ' ', 0)

	// print header
	fmt.Fprintln(w, "REPO\tTAGS")

	for _, repo := range repos {
		// get the tags and print to stdout
		tags, err := r.Tags(repo)
		if err != nil {
			logrus.Fatal(err)
		}

		fmt.Fprintf(w, "%s\t%s\n", repo, strings.Join(tags, ", "))
	}
	w.Flush()
}

func readDockerConfig() error {
	dcfg, err := cliconfig.Load(cliconfig.ConfigDir())
	if err != nil {
		return fmt.Errorf("Loading config file failed: %v", err)
	}
	if !dcfg.ContainsAuth() {
		return fmt.Errorf("No auth was present in %s, please pass a registry URL, username, and password", cliconfig.ConfigDir())
	}

	// if they passed the registryURL let's return those creds _if_ they exist
	if registryURL != "" {
		if creds, ok := dcfg.AuthConfigs[registryURL]; ok {
			username = creds.Username
			password = creds.Password
			return nil
		}
		return fmt.Errorf("User passed registry URL as %s but no auth creds exist", registryURL)
	}

	// set the auth config as the registryURL, username and Password
	for _, creds := range dcfg.AuthConfigs {
		username = creds.Username
		password = creds.Password
		registryURL = creds.ServerAddress
		return nil
	}

	return nil
}
