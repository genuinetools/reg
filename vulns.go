package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var vulnsCommand = cli.Command{
	Name:  "vulns",
	Usage: "get a vulnerability report for the image from CoreOS Clair",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "clair",
			Usage: "url to clair instance",
		},
		cli.IntFlag{
			Name:  "fixable-threshold",
			Usage: "number of fixable issues permitted",
			Value: 0,
		},
	},
	Action: func(c *cli.Context) error {
		if c.String("clair") == "" {
			return errors.New("clair url cannot be empty, pass --clair")
		}

		if c.Int("fixable-threshold") < 0 {
			return errors.New("fixable threshold must be a positive integer")
		}

		if len(c.Args()) < 1 {
			return fmt.Errorf("pass the name of the repository")
		}

		image, err := registry.ParseImage(c.Args().First())
		if err != nil {
			return err
		}

		// Create the registry client.
		r, err := createRegistryClient(c, image.Domain)
		if err != nil {
			return err
		}

		// Parse the timeout.
		timeout, err := time.ParseDuration(c.GlobalString("timeout"))
		if err != nil {
			return fmt.Errorf("parsing %s as duration failed: %v", c.GlobalString("timeout"), err)
		}

		// Initialize clair client.
		cr, err := clair.New(c.String("clair"), clair.Opt{
			Debug:    c.GlobalBool("debug"),
			Timeout:  timeout,
			Insecure: c.GlobalBool("insecure"),
		})
		if err != nil {
			return err
		}

		// Get the vulnerability report.
		report, err := cr.Vulnerabilities(r, image.Path, image.Reference())
		if err != nil {
			return err
		}

		// Iterate over the vulnerabilities by severity list.
		for sev, vulns := range report.VulnsBySeverity {
			for _, v := range vulns {
				if sev == "Fixable" {
					fmt.Printf("%s: [%s] \n%s\n%s\n", v.Name, v.Severity+" - Fixable", v.Description, v.Link)
					fmt.Printf("Fixed by: %s\n", v.FixedBy)
				} else {
					fmt.Printf("%s: [%s] \n%s\n%s\n", v.Name, v.Severity, v.Description, v.Link)
				}
				fmt.Println("-----------------------------------------")
			}
		}

		// Print summary and count.
		for sev, vulns := range report.VulnsBySeverity {
			fmt.Printf("%s: %d\n", sev, len(vulns))
		}

		// Return an error if there are more than 1 fixable vulns.
		fixable, ok := report.VulnsBySeverity["Fixable"]
		if ok {
			if len(fixable) > c.Int("fixable-threshold") {
				logrus.Fatalf("%d fixable vulnerabilities found", len(fixable))
			}
		}

		// Return an error if there are more than 10 bad vulns.
		badVulns := 0
		// Include any high vulns.
		if highVulns, ok := report.VulnsBySeverity["High"]; ok {
			badVulns += len(highVulns)
		}
		// Include any critical vulns.
		if criticalVulns, ok := report.VulnsBySeverity["Critical"]; ok {
			badVulns += len(criticalVulns)
		}
		// Include any defcon1 vulns.
		if defcon1Vulns, ok := report.VulnsBySeverity["Defcon1"]; ok {
			badVulns += len(defcon1Vulns)
		}
		if badVulns > 10 {
			logrus.Fatalf("%d bad vulnerabilities found", badVulns)
		}

		return nil
	},
}
