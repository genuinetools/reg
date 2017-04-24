package testutils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

const (
	registryConfig = `version: 0.1
log:
  level: debug
  formatter: text
  fields:
    service: registry
storage:
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: 0.0.0.0:5000
  headers:
    X-Content-Type-Options: [nosniff]
  host: https://localhost:5000
  tls:
    certificate: /etc/docker/registry/ssl/cert.pem
    key: /etc/docker/registry/ssl/key.pem`

	// admin:testing
	htpasswd = "admin:$apr1$2a7OBK4C$pZEqDfaxN3Qaywsi5hMKt1\n"
)

// StartRegistry starts a new registry container.
func StartRegistry(dcli *client.Client) (string, string, error) {
	image := "registry:2"

	hostConfig, err := createRegistryConfig()
	if err != nil {
		return "", "", err
	}

	hostHtpasswd, err := createRegistryHtpasswd()
	if err != nil {
		return "", "", err
	}

	if err := pullDockerImage(dcli, image); err != nil {
		return "", "", err
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", "", errors.New("No caller information")
	}

	fmt.Println(filename)

	r, err := dcli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: image,
			/*ExposedPorts: map[nat.Port]struct{}{
				"5000": {},
			},*/
		},
		&container.HostConfig{
			/*	PortBindings: map[nat.Port][]nat.PortBinding{
				"5000": []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: "5000",
				}},
			},*/
			NetworkMode: "host",
			Binds: []string{
				hostConfig + ":" + "/etc/docker/registry/config.yml" + ":ro",
				hostHtpasswd + ":" + "/etc/docker/registry/htpasswd" + ":ro",
				filepath.Join(filepath.Dir(filename), "snakeoil") + ":" + "/etc/docker/registry/ssl" + ":ro",
			},
		},
		nil, "")
	if err != nil {
		return "", "", err
	}

	// start the container
	if err := dcli.ContainerStart(context.Background(), r.ID, types.ContainerStartOptions{}); err != nil {
		return r.ID, "", err
	}

	// get the container's IP
	/*info, err := dcli.ContainerInspect(context.Background(), r.ID)
	if err != nil {
		return r.ID, "", err
	}*/
	port := ":5000"
	// addr := "http://" + info.NetworkSettings.IPAddress + port
	addr := "https://localhost" + port

	// waitForConn(info.NetworkSettings.IPAddress + port)
	waitForConn("localhost" + port)

	if err := prefillRegistry(dcli, "localhost"+port); err != nil {
		return r.ID, addr, err
	}

	return r.ID, addr, nil
}

// RemoveContainer removes with force a container by it's container ID.
func RemoveContainer(dcli *client.Client, ctrID string) error {
	if err := dcli.ContainerRemove(context.Background(), ctrID,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
		return err
	}

	return nil
}

// prefillRegistry adds images to a registry.
func prefillRegistry(dcli *client.Client, addr string) error {
	image := "alpine:latest"

	if err := pullDockerImage(dcli, image); err != nil {
		return err
	}

	if err := dcli.ImageTag(context.Background(), image, addr+"/"+image); err != nil {
		return err
	}

	auth, err := constructRegistryAuth("admin", "testing")
	if err != nil {
		return err
	}

	resp, err := dcli.ImagePush(context.Background(), addr+"/"+image, types.ImagePushOptions{
		RegistryAuth: auth,
	})
	if err != nil {
		return err
	}
	defer resp.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)

	return jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, fd, isTerm, nil)
}

func pullDockerImage(dcli *client.Client, image string) error {
	exists, err := imageExists(dcli, image)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	resp, err := dcli.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer resp.Close()

	fd, isTerm := term.GetFdInfo(os.Stdout)

	return jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, fd, isTerm, nil)
}

func imageExists(dcli *client.Client, image string) (bool, error) {
	_, _, err := dcli.ImageInspectWithRaw(context.Background(), image)
	if err == nil {
		return true, nil
	}

	if client.IsErrImageNotFound(err) {
		return false, nil
	}

	return false, err
}

func createRegistryConfig() (string, error) {
	tmpfile, err := ioutil.TempFile("", "registry")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.WriteString(registryConfig); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

func createRegistryHtpasswd() (string, error) {
	tmpfile, err := ioutil.TempFile("", "registry-htpasswd")
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.WriteString(htpasswd); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

// waitForConn takes a tcp addr and waits until it is reachable
func waitForConn(addr string) {
	n := 0
	max := 10
	for n < max {
		if _, err := net.Dial("tcp", addr); err != nil {
			fmt.Printf("try number %d to dial %s: %v\n", n, addr, err)
			n++
			if n != max {
				fmt.Println("sleeping for 1 second then will try again...")
				time.Sleep(time.Second)
			} else {
				fmt.Printf("[WHOOPS]: maximum retries for %s exceeded\n", addr)
			}
			continue
		} else {
			break
		}
	}
}

// constructRegistryAuth serializes the auth configuration as JSON base64 payload.
func constructRegistryAuth(identity, secret string) (string, error) {
	buf, err := json.Marshal(types.AuthConfig{Username: identity, Password: secret})
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}
