package main

import (
	"encoding/json"
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
	humanize "github.com/dustin/go-humanize"
	"github.com/jessfraz/reg/clair"
	"github.com/jessfraz/reg/registry"
	"github.com/jessfraz/reg/utils"
	wordwrap "github.com/mitchellh/go-wordwrap"
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
	tmpl     *template.Template
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

		// get the path to the static directory
		wd, err := os.Getwd()
		if err != nil {
			logrus.Fatal(err)
		}
		staticDir := filepath.Join(wd, "static")

		// create the template
		templateDir := filepath.Join(staticDir, "../templates")
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
		vulns := filepath.Join(templateDir, "vulns.html")
		if _, err := os.Stat(vulns); os.IsNotExist(err) {
			logrus.Fatalf("Template %s not found", vulns)
		}
		layout := filepath.Join(templateDir, "layout.html")
		if _, err := os.Stat(layout); os.IsNotExist(err) {
			logrus.Fatalf("Template %s not found", layout)
		}
		tmpl = template.Must(template.New("").Funcs(funcMap).ParseFiles(vulns, layout))

		// create the initial index
		logrus.Info("creating initial static index")
		if err := createStaticIndex(r, staticDir, c.GlobalString("clair")); err != nil {
			logrus.Fatalf("Error creating index: %v", err)
		}

		// parse the duration
		dur, err := time.ParseDuration(c.String("interval"))
		if err != nil {
			logrus.Fatalf("parsing %s as duration failed: %v", c.String("interval"), err)
		}
		ticker := time.NewTicker(dur)

		go func() {
			// create more indexes every X minutes based off interval
			for range ticker.C {
				if !updating {
					logrus.Info("creating timer based static index")
					if err := createStaticIndex(r, staticDir, c.GlobalString("clair")); err != nil {
						logrus.Warnf("creating static index failed: %v", err)
						wg.Wait()
						updating = false
					}
					wg.Wait()
					logrus.Info("finished waiting for vulns wait group")
				} else {
					logrus.Warnf("skipping timer based static index update for %s", c.String("interval"))
				}
			}
		}()

		// create mux server
		mux := http.NewServeMux()

		// static files handler
		staticHandler := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", staticHandler)

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

type data struct {
	RegistryURL string
	LastUpdated string
	Repos       []repository
}

type repository struct {
	Name        string
	Tag         string
	RepoURI     string
	CreatedDate string
	VulnURI     string
}

type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
}

func createStaticIndex(r *registry.Registry, staticDir, clairURI string) error {
	updating = true
	logrus.Info("fetching catalog")
	repoList, err := r.Catalog("")
	if err != nil {
		return fmt.Errorf("getting catalog failed: %v", err)
	}

	logrus.Info("fetching tags")
	var repos []repository
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

			var createdDate string
			for _, h := range m1.History {
				var comp v1Compatibility
				if err := json.Unmarshal([]byte(h.V1Compatibility), &comp); err != nil {
					return fmt.Errorf("unmarshal v1compatibility failed: %v", err)
				}
				createdDate = humanize.Time(comp.Created)
				break
			}

			repoURI := fmt.Sprintf("%s/%s", r.Domain, repo)
			if tag != "latest" {
				repoURI += ":" + tag
			}

			newrepo := repository{
				Name:        repo,
				Tag:         tag,
				RepoURI:     repoURI,
				CreatedDate: createdDate,
			}

			if clairURI != "" {
				wg.Add(1)

				go func(repo, tag string, i, j int) {
					defer wg.Done()

					throttle := time.Tick(time.Duration(time.Duration((i+1)*(j+1)*4) * time.Second))
					<-throttle

					logrus.Infof("creating vulns.txt for %s:%s", repo, tag)

					if err := createVulnStaticPage(r, staticDir, clairURI, repo, tag, m1); err != nil {
						// return fmt.Errorf("creating vuln static page for %s:%s failed: %v", repo, tag, err)
						logrus.Warnf("creating vuln static page for %s:%s failed: %v", repo, tag, err)
					}
				}(repo, tag, i, j)

				newrepo.VulnURI = filepath.Join(repo, tag)
			}
			repos = append(repos, newrepo)
		}
	}

	d := data{
		RegistryURL: r.Domain,
		Repos:       repos,
		LastUpdated: time.Now().Local().Format(time.RFC1123),
	}

	logrus.Info("rendering index template")
	if err := renderTemplate(staticDir, "index", "index.html", d); err != nil {
		return err
	}
	updating = false
	return nil
}

type vulnsReport struct {
	RegistryURL     string
	Repo            string
	Tag             string
	Date            string
	Vulns           []clair.Vulnerability
	VulnsBySeverity map[string][]clair.Vulnerability
	BadVulns        int
}

func createVulnStaticPage(r *registry.Registry, staticDir, clairURI, repo, tag string, m schema1.SignedManifest) error {
	report := vulnsReport{
		RegistryURL:     r.Domain,
		Repo:            repo,
		Tag:             tag,
		Date:            time.Now().Local().Format(time.RFC1123),
		VulnsBySeverity: make(map[string][]clair.Vulnerability),
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
		fmt.Printf("No need to analyse image %s:%s as there is no non-emtpy layer", repo, tag)
		return nil
	}

	// initialize clair
	cr, err := clair.New(clairURI, false)
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
	for _, f := range vl.Features {
		for _, v := range f.Vulnerabilities {
			report.Vulns = append(report.Vulns, v)
		}
	}

	vulnsBy := func(sev string, store map[string][]clair.Vulnerability) []clair.Vulnerability {
		items, found := store[sev]
		if !found {
			items = make([]clair.Vulnerability, 0)
			store[sev] = items
		}
		return items
	}

	// group by severity
	for _, v := range report.Vulns {
		sevRow := vulnsBy(v.Severity, report.VulnsBySeverity)
		report.VulnsBySeverity[v.Severity] = append(sevRow, v)
	}

	// calculate number of bad vulns
	report.BadVulns = len(report.VulnsBySeverity["High"]) + len(report.VulnsBySeverity["Critical"]) + len(report.VulnsBySeverity["Defcon1"])

	path := filepath.Join(repo, tag, "index.html")
	if err := renderTemplate(staticDir, "vulns", path, report); err != nil {
		return err
	}
	return nil
}

func renderTemplate(staticDir, templateName, dest string, data interface{}) error {
	// parse & execute the template
	logrus.Debugf("executing the template %s", templateName)

	path := filepath.Join(staticDir, dest)
	if err := os.MkdirAll(filepath.Dir(path), 0644); err != nil {
		return err
	}
	logrus.Debugf("creating/opening file %s", path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, templateName, data); err != nil {
		f.Close()
		return fmt.Errorf("execute template %s failed: %v", templateName, err)
	}

	return nil
}
