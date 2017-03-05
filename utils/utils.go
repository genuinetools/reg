package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/config"
	"github.com/jessfraz/reg/clair"
	"github.com/jessfraz/reg/registry"
	"github.com/urfave/cli"
)

// GetAuthConfig returns the docker registry AuthConfig.
func GetAuthConfig(c *cli.Context) (types.AuthConfig, error) {
	if c.GlobalString("username") != "" && c.GlobalString("password") != "" && c.GlobalString("registry") != "" {
		return types.AuthConfig{
			Username:      c.GlobalString("username"),
			Password:      c.GlobalString("password"),
			ServerAddress: c.GlobalString("registry"),
		}, nil
	}

	dcfg, err := config.Load(config.Dir())
	if err != nil {
		return types.AuthConfig{}, fmt.Errorf("Loading config file failed: %v", err)
	}

	// return error early if there are no auths saved
	if !dcfg.ContainsAuth() {
		if c.GlobalString("registry") != "" {
			return types.AuthConfig{
				ServerAddress: c.GlobalString("registry"),
			}, nil
		}
		return types.AuthConfig{}, fmt.Errorf("No auth was present in %s, please pass a registry, username, and password", config.Dir())
	}

	// if they passed a specific registry, return those creds _if_ they exist
	if c.GlobalString("registry") != "" {
		if creds, ok := dcfg.AuthConfigs[c.GlobalString("registry")]; ok {
			return creds, nil
		}
		return types.AuthConfig{}, fmt.Errorf("No authentication credentials exist for %s", c.GlobalString("registry"))
	}

	// set the auth config as the registryURL, username and Password
	for _, creds := range dcfg.AuthConfigs {
		return creds, nil
	}

	return types.AuthConfig{}, fmt.Errorf("Could not find any authentication credentials")
}

// GetRepoAndRef parses the repo name and reference.
func GetRepoAndRef(c *cli.Context) (repo, ref string, err error) {
	if len(c.Args()) < 1 {
		return "", "", errors.New("pass the name of the repository")
	}

	arg := c.Args()[0]
	parts := []string{}
	if strings.Contains(arg, "@") {
		parts = strings.Split(c.Args()[0], "@")
	} else if strings.Contains(arg, ":") {
		parts = strings.Split(c.Args()[0], ":")
	} else {
		parts = []string{arg}
	}

	repo = parts[0]
	ref = "latest"
	if len(parts) > 1 {
		ref = parts[1]
	}

	return
}

// NewClairLayer creates a clair layer from a docker registry image and fsLayers.
func NewClairLayer(r *registry.Registry, image string, fsLayers []schema1.FSLayer, index int) (*clair.Layer, error) {
	var parentName string
	if index < len(fsLayers)-1 {
		parentName = fsLayers[index+1].BlobSum.String()
	}

	// form the path
	p := strings.Join([]string{r.URL, "v2", image, "blobs", fsLayers[index].BlobSum.String()}, "/")

	// get the token
	token, err := r.Token(p)
	if err != nil {
		return nil, err
	}

	return &clair.Layer{
		Name:       fsLayers[index].BlobSum.String(),
		Path:       p,
		ParentName: parentName,
		Format:     "Docker",
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", token),
		},
	}, nil
}
