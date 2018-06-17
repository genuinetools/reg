package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/genuinetools/reg/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "reg"
	app.Version = fmt.Sprintf("version %s, build %s", version.VERSION, version.GITCOMMIT)
	app.Author = "The Genuinetools Authors"
	app.Email = "no-reply@butts.com"
	app.Usage = "Docker registry v2 client."

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "run in debug mode",
		},
		cli.BoolFlag{
			Name:  "insecure, k",
			Usage: "do not verify tls certificates",
		},
		cli.BoolFlag{
			Name:  "force-non-ssl, f",
			Usage: "force allow use of non-ssl",
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
			Name:  "timeout",
			Value: "1m",
			Usage: "timeout for HTTP requests",
		},
		cli.BoolFlag{
			Name:  "skip-ping",
			Usage: "skip pinging the registry while establishing connection",
		},
	}

	app.Commands = []cli.Command{
		deleteCommand,
		digestCommand,
		layerCommand,
		listCommand,
		manifestCommand,
		tagsCommand,
		vulnsCommand,
	}

	app.Before = func(c *cli.Context) (err error) {
		// Preload initializes any global options and configuration
		// before the main or sub commands are run.
		if c.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if len(c.Args()) == 0 {
			return
		}

		if c.Args()[0] == "help" {
			return
		}

		return
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func createRegistryClient(c *cli.Context, domain string) (*registry.Registry, error) {
	auth, err := repoutils.GetAuthConfig(c.GlobalString("username"), c.GlobalString("password"), domain)
	if err != nil {
		return nil, err
	}

	// Prevent non-ssl unless explicitly forced
	if !c.GlobalBool("force-non-ssl") && strings.HasPrefix(auth.ServerAddress, "http:") {
		return nil, fmt.Errorf("Attempt to use insecure protocol! Use non-ssl option to force")
	}

	// Parse the timeout.
	timeout, err := time.ParseDuration(c.GlobalString("timeout"))
	if err != nil {
		return nil, fmt.Errorf("parsing %s as duration failed: %v", c.GlobalString("timeout"), err)
	}

	// Create the registry client.
	return registry.New(auth, registry.Opt{
		Insecure: c.GlobalBool("insecure"),
		Debug:    c.GlobalBool("debug"),
		SkipPing: c.GlobalBool("skip-ping"),
		Timeout:  timeout,
	})
}
