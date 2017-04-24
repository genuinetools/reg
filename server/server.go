package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/gorilla/mux"
	"github.com/jessfraz/reg/clair"
	"github.com/jessfraz/reg/registry"
	"github.com/jessfraz/reg/utils"
	wordwrap "github.com/mitchellh/go-wordwrap"
	"github.com/urfave/cli"
)

const (
	// VERSION is the binary version.
	VERSION = "v0.2.0"

	dockerConfigPath = ".docker/config.json"
)

var (
	updating = false
	wg       sync.WaitGroup
	r        *registry.Registry
	cl       *clair.Clair
	tmpl     *template.Template
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

		// create a clair instance if needed
		if c.GlobalString("clair") != "" {
			cl, err = clair.New(c.GlobalString("clair"), c.GlobalBool("debug"))
			if err != nil {
				logrus.Warnf("creation of clair failed: %v", err)
			}
		}

		// create the initial index
		logrus.Info("creating initial static index")
		if err := analyseRepositories(r, cl, c.GlobalBool("debug"), c.GlobalInt("workers")); err != nil {
			logrus.Fatalf("Error creating index: %v", err)
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
					start := time.Now()
					if err := analyseRepositories(r, cl, c.GlobalBool("debug"), c.GlobalInt("workers")); err != nil {
						logrus.Warnf("repository analysis failed: %v", err)
						wg.Wait()
						updating = false
					}
					wg.Wait()
					elapsed := time.Since(start)
					logrus.Infof("finished repository analysis in %s", elapsed)
				} else {
					logrus.Warnf("skipping timer based repository analysis for %s", c.String("interval"))
				}
			}
		}()

		// get the path to the static directory
		wd, err := os.Getwd()
		if err != nil {
			logrus.Fatal(err)
		}
		staticDir := filepath.Join(wd, "static")

		// create the template
		templateDir := filepath.Join(staticDir, "../templates")

		// make sure all the templates exist
		vulns := filepath.Join(templateDir, "vulns.html")
		if _, err := os.Stat(vulns); os.IsNotExist(err) {
			logrus.Fatalf("Template %s not found", vulns)
		}
		layout := filepath.Join(templateDir, "repositories.html")
		if _, err := os.Stat(layout); os.IsNotExist(err) {
			logrus.Fatalf("Template %s not found", layout)
		}
		tags := filepath.Join(templateDir, "tags.html")
		if _, err := os.Stat(tags); os.IsNotExist(err) {
			logrus.Fatalf("Template %s not found", tags)
		}

		funcMap := template.FuncMap{
			"trim": func(s string) string {
				return wordwrap.WrapString(s, 80)
			},
			"color": func(s string) string {
				switch s = strings.ToLower(s); s {
				case "high":
					return "danger"
				case "critical":
					return "danger"
				case "defcon1":
					return "danger"
				case "medium":
					return "warning"
				case "low":
					return "info"
				case "negligible":
					return "info"
				case "unknown":
					return "default"
				default:
					return "default"
				}
			},
		}

		tmpl = template.Must(template.New("").Funcs(funcMap).ParseGlob(templateDir + "/*.html"))

		rc := registryController{
			reg: r,
			cl:  cl,
		}

		// create mux server
		mux := mux.NewRouter()

		// static files handler
		staticHandler := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("/", rc.tagsHandler)
		mux.Handle("/static", staticHandler)
		mux.HandleFunc("/repo/{repo}", rc.tagsHandler)
		mux.HandleFunc("/repo/{repo}/{tag}", rc.tagHandler)
		mux.HandleFunc("/repo/{repo}/{tag}/vulns", rc.vulnerabilitiesHandler)

		// set up the server
		port := c.String("port")
		server := &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		}
		logrus.Infof("Starting server on port %q", port)
		if c.String("cert") != "" && c.String("key") != "" {
			logrus.Fatal(server.ListenAndServeTLS(c.String("cert"), c.String("key")))
		} else {
			logrus.Fatal(server.ListenAndServe())
		}

		return nil
	}

	app.Run(os.Args)
}

type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
}

func analyseRepositories(r *registry.Registry, cl *clair.Clair, debug bool, workers int) error {
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

			if cl != nil {
				wg.Add(1)
				sem <- 1
				go func(repo, tag string, i, j int) {
					defer func() {
						wg.Done()
						<-sem
					}()

					logrus.Infof("search vulnerabilities for %s:%s", repo, tag)

					if err := searchVulnerabilities(r, cl, repo, tag, m1, debug); err != nil {
						logrus.Warnf("searching vulnerabilities for %s:%s failed: %v", repo, tag, err)
					}
				}(repo, tag, i, j)
			}
		}
	}

	updating = false
	return nil
}

func searchVulnerabilities(r *registry.Registry, cl *clair.Clair, repo, tag string, m schema1.SignedManifest, debug bool) error {
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

	for i := len(m.FSLayers) - 1; i >= 0; i-- {
		// form the clair layer
		l, err := cl.NewClairLayer(r, repo, m.FSLayers, i)
		if err != nil {
			return err
		}

		// post the layer
		if _, err := cl.PostLayer(l); err != nil {
			return err
		}
	}

	return nil
}
