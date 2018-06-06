package main

import (
	"fmt"

	"github.com/genuinetools/reg/repoutils"
	"github.com/urfave/cli"
)

var digestCommand = cli.Command{
	Name:  "digest",
	Usage: "get the digest",
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 1 {
			return fmt.Errorf("pass the name of the repository")
		}

		repo, ref, err := repoutils.GetRepoAndRef(c.Args()[0])
		if err != nil {
			return err
		}

		digest, err := r.Digest(repo, ref)
		if err != nil {
			return err
		}

		fmt.Println(digest)

		return nil
	},
}
