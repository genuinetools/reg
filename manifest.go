package main

import (
	"encoding/json"
	"fmt"

	"github.com/genuinetools/reg/registry"
	"github.com/urfave/cli"
)

var manifestCommand = cli.Command{
	Name:  "manifest",
	Usage: "get the json manifest for the specific reference of a repository",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "v1",
			Usage: "force the version of the manifest retrieved to v1",
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

		var manifest interface{}
		if c.Bool("v1") {
			// Get the v1 manifest if it was explicitly asked for.
			manifest, err = r.ManifestV1(image.Path, image.Reference())
			if err != nil {
				return err
			}
		} else {
			// Get the v2 manifest.
			manifest, err = r.Manifest(image.Path, image.Reference())
			if err != nil {
				return err
			}
		}

		b, err := json.MarshalIndent(manifest, " ", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(b))

		return nil
	},
}
