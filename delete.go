package main

import (
	"fmt"

	"github.com/genuinetools/reg/registry"
	"github.com/urfave/cli"
)

var deleteCommand = cli.Command{
	Name:    "delete",
	Aliases: []string{"rm"},
	Usage:   "delete a specific reference of a repository",
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

		if err := image.WithDigest(digest); err != nil {
			return err
		}

		// Delete the reference.
		if err := r.Delete(image.Path, digest); err != nil {
			return err
		}
		fmt.Printf("Deleted %s\n", image.String())

		return nil
	},
}
