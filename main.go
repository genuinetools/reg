package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/api/types"
	"github.com/jessfraz/reg/clair"
	"github.com/jessfraz/reg/registry"
	"github.com/jessfraz/reg/utils"
	digest "github.com/opencontainers/go-digest"
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
			auth, err = utils.GetAuthConfig(c)
			if err != nil {
				return err
			}

			// create the registry client
			if c.GlobalBool("insecure") {
				r, err = registry.NewInsecure(auth, c.GlobalBool("debug"))
				if err != nil {
					return err
				}
			} else {
				r, err = registry.New(auth, c.GlobalBool("debug"))
				if err != nil {
					return err
				}
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
		cli.BoolFlag{
			Name:  "insecure, k",
			Usage: "do not verify tls certificates",
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
			Usage: "URL to the private registry (ex. r.j3ss.co)",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "delete",
			Aliases: []string{"rm"},
			Usage:   "delete a specific reference of a repository",
			Action: func(c *cli.Context) error {
				repo, ref, err := utils.GetRepoAndRef(c)
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
				repos, err := r.Catalog("")
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
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "v1",
					Usage: "force get v1 manifest",
				},
			},
			Action: func(c *cli.Context) error {
				repo, ref, err := utils.GetRepoAndRef(c)
				if err != nil {
					return err
				}

				var manifest interface{}
				if c.Bool("v1") {
					manifest, err = r.ManifestV1(repo, ref)
					if err != nil {
						return err
					}
				} else {
					manifest, err = r.Manifest(repo, ref)
					if err != nil {
						return err
					}
				}

				b, err := json.MarshalIndent(manifest, " ", "  ")
				if err != nil {
					return err
				}

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
		{
			Name:    "download",
			Aliases: []string{"layer"},
			Usage:   "download a layer for the specific reference of a repository",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "output, o",
					Usage: "output file, default to stdout",
				},
			},
			Action: func(c *cli.Context) error {
				repo, ref, err := utils.GetRepoAndRef(c)
				if err != nil {
					return err
				}

				layer, err := r.DownloadLayer(repo, digest.Digest(ref))
				if err != nil {
					return err
				}
				defer layer.Close()

				b, err := ioutil.ReadAll(layer)
				if err != nil {
					return err
				}

				if c.String("output") != "" {
					return ioutil.WriteFile(c.String("output"), b, 0644)
				}

				fmt.Fprint(os.Stdout, string(b))

				return nil
			},
		},
		{
			Name:  "vulns",
			Usage: "get a vulnerability report for the image from CoreOS Clair",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "clair",
					Usage: "url to clair instance",
				},
			},
			Action: func(c *cli.Context) error {
				if c.String("clair") == "" {
					return errors.New("clair url cannot be empty, pass --clair")
				}

				repo, ref, err := utils.GetRepoAndRef(c)
				if err != nil {
					return err
				}

				// get the manifest
				m, err := r.ManifestV1(repo, ref)
				if err != nil {
					return err
				}

				// filter out the empty layers
				var filteredLayers []schema1.FSLayer
				for _, layer := range m.FSLayers {
					if layer.BlobSum != clair.EmptyLayerBlobSum {
						filteredLayers = append(filteredLayers, layer)
					}
				}
				m.FSLayers = filteredLayers
				if len(m.FSLayers) == 0 {
					fmt.Printf("No need to analyse image %s:%s as there is no non-emtpy layer", repo, ref)
					return nil
				}

				// initialize clair
				cr, err := clair.New(c.String("clair"), c.GlobalBool("debug"))
				if err != nil {
					return err
				}

				for i := len(m.FSLayers) - 1; i >= 0; i-- {
					// form the clair layer
					l, err := utils.NewClairLayer(r, repo, m.FSLayers, i)
					if err != nil {
						return err
					}

					// post the layer
					if _, err := cr.PostLayer(l); err != nil {
						return err
					}
				}

				vl, err := cr.GetLayer(m.FSLayers[0].BlobSum.String(), false, true)
				if err != nil {
					return err
				}

				// get the vulns
				var vulns []clair.Vulnerability
				for _, f := range vl.Features {
					for _, v := range f.Vulnerabilities {
						vulns = append(vulns, v)
					}
				}
				fmt.Printf("Found %d vulnerabilities \n", len(vulns))

				vulnsBy := func(sev string, store map[string][]clair.Vulnerability) []clair.Vulnerability {
					items, found := store[sev]
					if !found {
						items = make([]clair.Vulnerability, 0)
						store[sev] = items
					}
					return items
				}

				// group by severity
				store := make(map[string][]clair.Vulnerability)
				for _, v := range vulns {
					sevRow := vulnsBy(v.Severity, store)
					store[v.Severity] = append(sevRow, v)
				}

				// iterate over the priorities list
				iteratePriorities := func(f func(sev string)) {
					for _, sev := range clair.Priorities {
						if len(store[sev]) != 0 {
							f(sev)
						}
					}
				}
				iteratePriorities(func(sev string) {
					for _, v := range store[sev] {
						fmt.Printf("%s: [%s] \n%s\n%s\n", v.Name, v.Severity, v.Description, v.Link)
						fmt.Println("-----------------------------------------")
					}
				})
				iteratePriorities(func(sev string) {
					fmt.Printf("%s: %d\n", sev, len(store[sev]))
				})

				// return an error if there are more than 10 bad vulns
				lenBadVulns := len(store["High"]) + len(store["Critical"]) + len(store["Defcon1"])
				if lenBadVulns > 10 {
					logrus.Fatalf("%d bad vunerabilities found", lenBadVulns)
				}

				return nil
			},
		},
	}

	app.Run(os.Args)
}
