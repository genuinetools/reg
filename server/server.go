package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/manifest/schema1"
	humanize "github.com/dustin/go-humanize"
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
			Usage: "URL to the provate registry (ex. r.j3ss.co)",
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
			return err
		}

		// create the registry client
		r, err := registry.New(auth, c.GlobalBool("debug"))
		if err != nil {
			return err
		}

		// get the path to the static directory
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		staticDir := filepath.Join(wd, "static")

		// create the initial index
		if err := createStaticIndex(r, staticDir, c.GlobalString("clair")); err != nil {
			return err
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
					if err := createStaticIndex(r, staticDir, c.GlobalString("clair")); err != nil {
						logrus.Warnf("creating static index failed: %v", err)
						wg.Wait()
						updating = false
					}
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

			manifest, err := r.Manifest(repo, tag)
			if err != nil {
				return fmt.Errorf("getting tags for %s:%s failed: %v", repo, tag, err)
			}

			var createdDate string
			if m1, ok := manifest.(schema1.SignedManifest); ok {
				history := m1.History
				for _, h := range history {
					var comp v1Compatibility
					if err := json.Unmarshal([]byte(h.V1Compatibility), &comp); err != nil {
						return fmt.Errorf("unmarshal v1compatibility failed: %v", err)
					}
					createdDate = humanize.Time(comp.Created)
					break
				}
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

					throttle := time.Tick(time.Duration(time.Duration((i+1)*(j+1)) * time.Microsecond))
					<-throttle

					logrus.Infof("creating vulns.txt for %s:%s", repo, tag)

					if err := createVulnStaticPage(r, staticDir, clairURI, repo, tag); err != nil {
						// return fmt.Errorf("creating vuln static page for %s:%s failed: %v", repo, tag, err)
						logrus.Warnf("creating vuln static page for %s:%s failed: %v", repo, tag, err)
					}
				}(repo, tag, i, j)

				newrepo.VulnURI = filepath.Join(repo, tag, "vulns.txt")
			}
			repos = append(repos, newrepo)
		}
	}

	// create temporoary file to save template to
	logrus.Info("creating temporary file for template")
	f, err := ioutil.TempFile("", "reg-server")
	if err != nil {
		return fmt.Errorf("creating temp file failed: %v", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	// parse & execute the template
	logrus.Info("parsing and executing the template")
	templateDir := filepath.Join(staticDir, "../templates")
	lp := filepath.Join(templateDir, "layout.html")

	d := data{
		RegistryURL: r.Domain,
		Repos:       repos,
		LastUpdated: time.Now().Local().Format(time.RFC1123),
	}
	tmpl := template.Must(template.New("").ParseFiles(lp))
	if err := tmpl.ExecuteTemplate(f, "layout", d); err != nil {
		return fmt.Errorf("execute template failed: %v", err)
	}
	f.Close()

	index := filepath.Join(staticDir, "index.html")
	logrus.Infof("renaming the temporary file %s to %s", f.Name(), index)
	if err := os.Rename(f.Name(), index); err != nil {
		return fmt.Errorf("renaming result from %s to %s failed: %v", f.Name(), index, err)
	}
	updating = false
	return nil
}

func createVulnStaticPage(r *registry.Registry, staticDir, clairURI, repo, tag string) error {
	// get the manifest
	m, err := r.ManifestV1(repo, tag)
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
	var vulns []clair.Vulnerability
	for _, f := range vl.Features {
		for _, v := range f.Vulnerabilities {
			vulns = append(vulns, v)
		}
	}

	path := filepath.Join(staticDir, repo, tag, "vulns.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0644); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Found %d vulnerabilities \n", len(vulns))

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
		fmt.Fprintf(file, "%s: %d\n", sev, len(store[sev]))
	})
	fmt.Fprintln(file, "")

	// return an error if there are more than 10 bad vulns
	lenBadVulns := len(store["High"]) + len(store["Critical"]) + len(store["Defcon1"])
	if lenBadVulns > 10 {
		fmt.Fprintln(file, "--------------- ALERT ---------------")
		fmt.Fprintf(file, "%d bad vunerabilities found", lenBadVulns)
	}
	fmt.Fprintln(file, "")

	iteratePriorities(func(sev string) {
		for _, v := range store[sev] {
			fmt.Fprintf(file, "%s: [%s] \n%s\n%s\n", v.Name, v.Severity, v.Description, v.Link)
			fmt.Fprintln(file, "-----------------------------------------")
		}
	})

	return nil
}
