package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/engine-api/types"
	"github.com/jessfraz/reg/registry"
	"github.com/urfave/cli"
)

const (
	// VERSION is the binary version.
	VERSION = "v0.2.0"

	dockerConfigPath = ".docker/config.json"
)

var (
	auth types.AuthConfig
	r    *registry.Registry
)

// preload initializes any global options and configuration
// before the main or sub commands are run.
func preload(c *cli.Context) (err error) {
	if c.GlobalBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if len(c.Args()) > 0 {
		if c.Args()[0] != "help" {
			auth, err = getAuthConfig(c)
			if err != nil {
				return err
			}

			// create the registry client
			r, err = registry.New(auth, c.GlobalBool("debug"))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "reg"
	app.Version = VERSION
	app.Author = "@jessfraz"
	app.Email = "no-reply@butts.com"
	app.Usage = "Docker registry v2 client."
	app.Before = preload
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "run in debug mode",
		},
		cli.StringFlag{
			Name:  "username, u",
			Usage: "username for the registry",
		},
		cli.StringFlag{
			Name:  "password, p",
			Usage: "password for the registry",
		},
		cli.StringFlag{
			Name:  "registry, r",
			Usage: "URL to the provate registry (ex. r.j3ss.co)",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "delete",
			Aliases: []string{"rm"},
			Usage:   "delete a specific reference of a repository",
			Action: func(c *cli.Context) error {
				repo, ref, err := getRepoAndRef(c)
				if err != nil {
					return err
				}

				if err := r.Delete(repo, ref); err != nil {
					return fmt.Errorf("Delete %s@%s failed: %v", repo, ref, err)
				}
				fmt.Printf("Deleted %s@%s\n", repo, ref)

				return nil
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list all repositories",
			Action: func(c *cli.Context) error {
				// get the repositories via catalog
				repos, err := r.Catalog()
				if err != nil {
					return err
				}

				fmt.Printf("Repositories for %s\n", auth.ServerAddress)

				// setup the tab writer
				w := tabwriter.NewWriter(os.Stdout, 20, 1, 3, ' ', 0)

				// print header
				fmt.Fprintln(w, "REPO\tTAGS")

				for _, repo := range repos {
					// get the tags and print to stdout
					tags, err := r.Tags(repo)
					if err != nil {
						return err
					}

					fmt.Fprintf(w, "%s\t%s\n", repo, strings.Join(tags, ", "))
				}

				w.Flush()
				return nil
			},
		},
		{
			Name:  "manifest",
			Usage: "get the json manifest for the specific reference of a repository",
			Action: func(c *cli.Context) error {
				repo, ref, err := getRepoAndRef(c)
				if err != nil {
					return err
				}

				manifest, err := r.Manifest(repo, ref)
				if err != nil {
					return err
				}

				b, err := json.MarshalIndent(manifest, " ", "  ")
				if err != nil {
					return err
				}

				// print the tags
				fmt.Println(string(b))

				return nil
			},
		},
		{
			Name:  "tags",
			Usage: "get the tags for a repository",
			Action: func(c *cli.Context) error {
				if len(c.Args()) < 1 {
					return fmt.Errorf("pass the name of the repository")
				}

				tags, err := r.Tags(c.Args()[0])
				if err != nil {
					return err
				}

				// print the tags
				fmt.Println(strings.Join(tags, "\n"))

				return nil
			},
		},
	}

	app.Run(os.Args)
}

func getAuthConfig(c *cli.Context) (types.AuthConfig, error) {
	if c.GlobalString("username") != "" && c.GlobalString("password") != "" && c.GlobalString("registry") != "" {
		return types.AuthConfig{
			Username:      c.GlobalString("username"),
			Password:      c.GlobalString("password"),
			ServerAddress: c.GlobalString("registry"),
		}, nil
	}

	dcfg, err := cliconfig.Load(cliconfig.ConfigDir())
	if err != nil {
		return types.AuthConfig{}, fmt.Errorf("Loading config file failed: %v", err)
	}

	// return error early if there are no auths saved
	if !dcfg.ContainsAuth() {
		if c.GlobalString("registry") != "" {
			return types.AuthConfig{
				ServerAddress: c.GlobalString("registry"),
			}, nil
		}
		return types.AuthConfig{}, fmt.Errorf("No auth was present in %s, please pass a registry, username, and password", cliconfig.ConfigDir())
	}

	// if they passed a specific registry, return those creds _if_ they exist
	if c.GlobalString("registry") != "" {
		if creds, ok := dcfg.AuthConfigs[c.GlobalString("registry")]; ok {
			return creds, nil
		}
		return types.AuthConfig{}, fmt.Errorf("No authentication credentials exist for %s", c.GlobalString("registry"))
	}

	// set the auth config as the registryURL, username and Password
	for _, creds := range dcfg.AuthConfigs {
		return creds, nil
	}

	return types.AuthConfig{}, fmt.Errorf("Could not find any authentication credentials")
}

func getRepoAndRef(c *cli.Context) (repo, ref string, err error) {
	if len(c.Args()) < 1 {
		return "", "", errors.New("pass the name of the repository")
	}

	arg := c.Args()[0]
	parts := []string{}
	if strings.Contains(arg, "@") {
		parts = strings.Split(c.Args()[0], "@")
	} else if strings.Contains(arg, ":") {
		parts = strings.Split(c.Args()[0], ":")
	} else {
		parts = []string{arg}
	}

	repo = parts[0]
	ref = "latest"
	if len(parts) > 1 {
		ref = parts[1]
	}

	return
}
