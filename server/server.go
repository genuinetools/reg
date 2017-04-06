package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/jessfraz/reg/clair"
	"github.com/jessfraz/reg/registry"
	"github.com/jessfraz/reg/utils"
	"github.com/urfave/cli"
)

const (
	// VERSION is the binary version.
	VERSION = "v0.1.0"

	dockerConfigPath = ".docker/config.json"
)

var (
	updating = false
	wg       sync.WaitGroup
	r        *registry.Registry
)

// preload initializes any global options and configuration
// before the main or sub commands are run.
func preload(c *cli.Context) (err error) {
	if c.GlobalBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "reg-server"
	app.Version = VERSION
	app.Author = "@jessfraz"
	app.Email = "no-reply@butts.com"
	app.Usage = "Docker registry v2 static UI server."
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
			Usage: "URL to the private registry (ex. r.j3ss.co)",
		},
		cli.BoolFlag{
			Name:  "insecure, k",
			Usage: "do not verify tls certificates of registry",
		},
		cli.StringFlag{
			Name:  "port",
			Value: "8080",
			Usage: "port for server to run on",
		},
		cli.StringFlag{
			Name:  "cert",
			Usage: "path to ssl cert",
		},
		cli.StringFlag{
			Name:  "key",
			Usage: "path to ssl key",
		},
		cli.StringFlag{
			Name:  "interval",
			Value: "5m",
			Usage: "interval to generate new index.html's at",
		},
		cli.StringFlag{
			Name:  "clair",
			Usage: "url to clair instance",
		},
		cli.IntFlag{
			Name:  "workers, w",
			Value: 20,
			Usage: "number of workers to analyse for vulnerabilities",
		},
	}
	app.Action = func(c *cli.Context) error {
		auth, err := utils.GetAuthConfig(c)
		if err != nil {
			logrus.Fatal(err)
		}

		// create the registry client
		if c.GlobalBool("insecure") {
			r, err = registry.NewInsecure(auth, c.GlobalBool("debug"))
			if err != nil {
				logrus.Fatal(err)
			}
		} else {
			r, err = registry.New(auth, c.GlobalBool("debug"))
			if err != nil {
				logrus.Fatal(err)
			}
		}

		// parse the duration
		dur, err := time.ParseDuration(c.String("interval"))
		if err != nil {
			logrus.Fatalf("parsing %s as duration failed: %v", c.String("interval"), err)
		}
		ticker := time.NewTicker(dur)

		go func() {
			// analyse repositories every X minutes based off interval
			for range ticker.C {
				if !updating {
					logrus.Info("start repository analysis")
					if err := analyseRepositories(r, c.GlobalString("clair"), c.GlobalBool("debug"), c.GlobalInt("workers")); err != nil {
						logrus.Warnf("repository analysis failed: %v", err)
						wg.Wait()
						updating = false
					}
					wg.Wait()
					logrus.Info("finished waiting for vulns wait group")
				} else {
					logrus.Warnf("skipping timer based repository analysis for %s", c.String("interval"))
				}
			}
		}()

		port := c.String("port")
		keyfile := c.String("key")
		certfile := c.String("cert")
		cl, err := clair.New(c.GlobalString("clair"), c.GlobalBool("debug"))
		if err != nil {
			logrus.Warnf("creation of clair failed: %v", err)
		}
		logrus.Fatal(listenAndServe(port, keyfile, certfile, r, cl))
		return nil
	}

	app.Run(os.Args)
}

type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
}

func analyseRepositories(r *registry.Registry, clairURI string, debug bool, workers int) error {
	updating = true
	logrus.Info("fetching catalog")
	repoList, err := r.Catalog("")
	if err != nil {
		return fmt.Errorf("getting catalog failed: %v", err)
	}

	logrus.Info("fetching tags")
	sem := make(chan int, workers)
	for i, repo := range repoList {
		// get the tags
		tags, err := r.Tags(repo)
		if err != nil {
			return fmt.Errorf("getting tags for %s failed: %v", repo, err)
		}
		for j, tag := range tags {
			// get the manifest

			m1, err := r.ManifestV1(repo, tag)
			if err != nil {
				logrus.Warnf("getting v1 manifest for %s:%s failed: %v", repo, tag, err)
			}

			if clairURI != "" {
				wg.Add(1)
				sem <- 1
				go func(repo, tag string, i, j int) {
					defer func() {
						wg.Done()
						<-sem
					}()

					logrus.Infof("search vulnerabilities for %s:%s", repo, tag)

					if err := searchVulnerabilities(r, clairURI, repo, tag, m1, debug); err != nil {
						logrus.Warnf("searching vulnerabilities for %s:%s failed: %v", repo, tag, err)
					}
				}(repo, tag, i, j)
			}
		}
	}

	updating = false
	return nil
}

func searchVulnerabilities(r *registry.Registry, clairURI, repo, tag string, m schema1.SignedManifest, debug bool) error {
	// filter out the empty layers
	var filteredLayers []schema1.FSLayer
	for _, layer := range m.FSLayers {
		if layer.BlobSum != clair.EmptyLayerBlobSum {
			filteredLayers = append(filteredLayers, layer)
		}
	}
	m.FSLayers = filteredLayers
	if len(m.FSLayers) == 0 {
		fmt.Printf("No need to analyse image %s:%s as there is no non-emtpy layer", repo, tag)
		return nil
	}

	// initialize clair
	cr, err := clair.New(clairURI, debug)
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

	return nil
}
