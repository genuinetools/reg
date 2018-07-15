package main

import (
	"context"
	"flag"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/genuinetools/pkg/cli"
	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/genuinetools/reg/version"
	"github.com/gorilla/mux"
	wordwrap "github.com/mitchellh/go-wordwrap"
	"github.com/sirupsen/logrus"
)

var (
	insecure    bool
	forceNonSSL bool
	skipPing    bool

	interval time.Duration
	timeout  time.Duration

	username       string
	password       string
	registryServer string
	clairServer    string

	once bool

	cert string
	key  string
	port string

	debug bool

	updating bool
	r        *registry.Registry
	cl       *clair.Clair
	tmpl     *template.Template
)

func main() {
	// Create a new cli program.
	p := cli.NewProgram()
	p.Name = "reg-server"
	p.Description = "Docker registry v2 static UI server"

	// Set the GitCommit and Version.
	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	// Setup the global flags.
	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.BoolVar(&insecure, "insecure", false, "do not verify tls certificates")
	p.FlagSet.BoolVar(&insecure, "k", false, "do not verify tls certificates")

	p.FlagSet.BoolVar(&forceNonSSL, "force-non-ssl", false, "force allow use of non-ssl")
	p.FlagSet.BoolVar(&forceNonSSL, "f", false, "force allow use of non-ssl")

	p.FlagSet.BoolVar(&skipPing, "skip-ping", false, "skip pinging the registry while establishing connection")

	p.FlagSet.DurationVar(&interval, "interval", time.Hour, "interval to generate new index.html's at")
	p.FlagSet.DurationVar(&timeout, "timeout", time.Minute, "timeout for HTTP requests")

	p.FlagSet.StringVar(&username, "username", "", "username for the registry")
	p.FlagSet.StringVar(&username, "u", "", "username for the registry")

	p.FlagSet.StringVar(&password, "password", "", "password for the registry")
	p.FlagSet.StringVar(&password, "p", "", "password for the registry")

	p.FlagSet.StringVar(&registryServer, "registry", "", "URL to the private registry (ex. r.j3ss.co)")
	p.FlagSet.StringVar(&registryServer, "r", "", "URL to the private registry (ex. r.j3ss.co)")

	p.FlagSet.StringVar(&clairServer, "clair", "", "url to clair instance")

	p.FlagSet.StringVar(&cert, "cert", "", "path to ssl cert")
	p.FlagSet.StringVar(&key, "key", "", "path to ssl key")
	p.FlagSet.StringVar(&port, "port", "8080", "port for server to run on")

	p.FlagSet.BoolVar(&once, "once", false, "generate an output once and then exit")

	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")

	// Set the before function.
	p.Before = func(ctx context.Context) error {
		// Set the log level.
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		return nil
	}

	// Set the main program action.
	p.Action = func(ctx context.Context) error {
		auth, err := repoutils.GetAuthConfig(username, password, registryServer)
		if err != nil {
			logrus.Fatal(err)
		}

		// Create the registry client.
		r, err = registry.New(auth, registry.Opt{
			Insecure: insecure,
			Debug:    debug,
			SkipPing: skipPing,
			Timeout:  timeout,
		})
		if err != nil {
			logrus.Fatal(err)
		}

		// create a clair instance if needed
		if len(clairServer) < 1 {
			cl, err = clair.New(clairServer, clair.Opt{
				Insecure: insecure,
				Debug:    debug,
				Timeout:  timeout,
			})
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

		// create the initial index
		logrus.Info("creating initial static index")
		if err := rc.repositories(staticDir); err != nil {
			logrus.Fatalf("Error creating index: %v", err)
		}

		if once {
			logrus.Info("Output generated")
			return nil
		}

		ticker := time.NewTicker(interval)

		go func() {
			// create more indexes every X minutes based off interval
			for range ticker.C {
				if !updating {
					logrus.Info("creating timer based static index")
					if err := rc.repositories(staticDir); err != nil {
						logrus.Warnf("creating static index failed: %v", err)
						updating = false
					}
				} else {
					logrus.Warnf("skipping timer based static index update for %s", interval.String())
				}
			}
		}()

		// create mux server
		mux := mux.NewRouter()
		mux.UseEncodedPath()

		// static files handler
		staticHandler := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("/repo/{repo}/tags", rc.tagsHandler)
		mux.HandleFunc("/repo/{repo}/tags/", rc.tagsHandler)
		mux.HandleFunc("/repo/{repo}/tag/{tag}", rc.vulnerabilitiesHandler)
		mux.HandleFunc("/repo/{repo}/tag/{tag}/", rc.vulnerabilitiesHandler)
		mux.HandleFunc("/repo/{repo}/tag/{tag}/vulns", rc.vulnerabilitiesHandler)
		mux.HandleFunc("/repo/{repo}/tag/{tag}/vulns/", rc.vulnerabilitiesHandler)
		mux.HandleFunc("/repo/{repo}/tag/{tag}/vulns.json", rc.vulnerabilitiesHandler)
		mux.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticHandler))
		mux.Handle("/", staticHandler)

		// set up the server
		server := &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		}
		logrus.Infof("Starting server on port %q", port)
		if len(cert) > 0 && len(key) > 0 {
			logrus.Fatal(server.ListenAndServeTLS(cert, key))
		} else {
			logrus.Fatal(server.ListenAndServe())
		}

		return nil
	}

	// Run our program.
	p.Run()
}
