package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/genuinetools/reg/registry"
	"github.com/urfave/cli"
)

var layerCommand = cli.Command{
	Name:    "layer",
	Aliases: []string{"download"},
	Usage:   "download a layer for the specific reference of a repository",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "output, o",
			Usage: "output file, default to stdout",
		},
	},
	Action: func(c *cli.Context) error {
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

		// Get the digest.
		digest, err := r.Digest(image)
		if err != nil {
			return err
		}

		// Download the layer.
		layer, err := r.DownloadLayer(image.Path, digest)
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
}
