package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/cliconfig"
	"github.com/docker/engine-api/types"
	"github.com/jessfraz/reg/registry"
	"github.com/urfave/cli"
)

const (
	// VERSION is the binary version.
	VERSION = "v0.1.0"

	dockerConfigPath = ".docker/config.json"
)

var (
	updating = false
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
	}
	app.Action = func(c *cli.Context) error {
		auth, err := getAuthConfig(c)
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
		if err := createStaticIndex(r, staticDir); err != nil {
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
					if err := createStaticIndex(r, staticDir); err != nil {
						logrus.Warnf("creating static index failed: %v", err)
					}
				}
			}
		}()

		// create mux server
		mux := http.NewServeMux()

		// static files handler
		staticHandler := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", staticHandler)
		// TODO: add handler for individual repos

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
}

type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
}

func createStaticIndex(r *registry.Registry, staticDir string) error {
	updating = true
	logrus.Info("fetching catalog")
	repoList, err := r.Catalog()
	if err != nil {
		return fmt.Errorf("getting catalog failed: %v", err)
	}

	logrus.Info("fetching tags")
	var repos []repository
	for _, repo := range repoList {
		// get the tags
		tags, err := r.Tags(repo)
		if err != nil {
			return fmt.Errorf("getting tags for %s failed: %v", repo, err)
		}
		for _, tag := range tags {
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
					createdDate = comp.Created.Format(time.RFC1123)
				}
			}

			repoURI := fmt.Sprintf("%s/%s", r.Domain, repo)
			if tag != "latest" {
				repoURI += ":" + tag
			}

			repos = append(repos, repository{
				Name:        repo,
				Tag:         tag,
				RepoURI:     repoURI,
				CreatedDate: createdDate,
			})
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
		LastUpdated: time.Now().Format(time.RFC1123),
	}
	tmpl := template.Must(template.New("").ParseFiles(lp))
	if err := tmpl.ExecuteTemplate(f, "layout", d); err != nil {
		return fmt.Errorf("execute template failed: %v", err)
	}
	f.Close()

	logrus.Info("renaming the temporary file to index.html")
	index := filepath.Join(staticDir, "index.html")
	if err := os.Rename(f.Name(), index); err != nil {
		return fmt.Errorf("renaming result from %s to %s failed: %v", f.Name(), index, err)
	}
	updating = false
	return nil
}
