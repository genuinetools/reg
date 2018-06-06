package main

import (
	"encoding/json"
	"fmt"

	"github.com/genuinetools/reg/repoutils"
	"github.com/urfave/cli"
)

var digestCommand = cli.Command{
	Name:  "digest",
	Usage: "get the Docker-Content-Digest",
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 1 {
			return fmt.Errorf("pass the name of the repository")
		}

		repo, ref, err := repoutils.GetRepoAndRef(c.Args()[0])
		if err != nil {
			return err
		}

		var digest interface{}
		digest, err = r.Digest(repo, ref)
		if err != nil {
			return err
		}

		b, err := json.MarshalIndent(digest, " ", "  ")
		if err != nil {
			return err
		}

		fmt.Println(string(b))

		return nil
	},
}
