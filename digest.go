package main

import (
	"fmt"

	"github.com/genuinetools/reg/registry"
	"github.com/urfave/cli"
)

var digestCommand = cli.Command{
	Name:  "digest",
	Usage: "get the digest",
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

		fmt.Println(digest.String())

		return nil
	},
}
