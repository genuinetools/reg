package main

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
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
	r    *registry.Registry
	cl   *clair.Clair
	tmpl *template.Template
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

		// create a clair instance if needed
		if c.GlobalString("clair") != "" {
			cl, err = clair.New(c.GlobalString("clair"), c.GlobalBool("debug"))
			if err != nil {
				logrus.Warnf("creation of clair failed: %v", err)
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
		mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))
		mux.HandleFunc("/repo/{repo}", rc.tagsHandler)
		mux.HandleFunc("/repo/{repo}/{tag}", rc.tagHandler)
		mux.HandleFunc("/repo/{repo}/{tag}/vulns", rc.vulnerabilitiesHandler)
		mux.HandleFunc("/", rc.repositoriesHandler)

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
