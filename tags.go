package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/genuinetools/reg/registry"
	"github.com/urfave/cli"
)

var tagsCommand = cli.Command{
	Name:  "tags",
	Usage: "get the tags for a repository",
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

		tags, err := r.Tags(image.Path)
		if err != nil {
			return err
		}
		sort.Strings(tags)

		// Print the tags.
		fmt.Println(strings.Join(tags, "\n"))

		return nil
	},
}
