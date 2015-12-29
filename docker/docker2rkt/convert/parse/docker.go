package parse

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/containerops/ameba/docker/docker2rkt/convert/manifest"
)

const (
	dockercfgFileName = ".dockercfg"
)

// splitReposName breaks a reposName into an index name and remote name
func SplitReposName(reposName string) (string, string) {
	nameParts := strings.SplitN(reposName, "/", 2)
	var indexName, remoteName string
	if len(nameParts) == 1 || (!strings.Contains(nameParts[0], ".") &&
		!strings.Contains(nameParts[0], ":") && nameParts[0] != "localhost") {
		// This is a Docker Index repos (ex: samalba/hipache or ubuntu)
		// The URL for the index is different depending on the version of the
		// API used to fetch it, so it cannot be inferred here.
		indexName = ""
		remoteName = reposName
	} else {
		indexName = nameParts[0]
		remoteName = nameParts[1]
	}
	return indexName, remoteName
}

// Get a repos name and returns the right reposName + tag
// The tag can be confusing because of a port in a repository name.
//     Ex: localhost.localdomain:5000/samalba/hipache:latest
func parseRepositoryTag(repos string) (string, string) {
	n := strings.LastIndex(repos, ":")
	if n < 0 {
		return repos, ""
	}
	if tag := repos[n+1:]; !strings.Contains(tag, "/") {
		return repos[:n], tag
	}
	return repos, ""
}

func decodeDockerAuth(s string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth configuration file")
	}
	user := parts[0]
	password := strings.Trim(parts[1], "\x00")
	return user, password, nil
}

func getHomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

// GetDockercfgAuth reads a ~/.dockercfg file and returns the username and password
// of the given docker index server.
func GetAuthInfo(indexServer string) (string, string, error) {
	dockerCfgPath := path.Join(getHomeDir(), dockercfgFileName)

	if _, err := os.Stat(dockerCfgPath); os.IsNotExist(err) {
		return "", "", nil
	}

	j, err := ioutil.ReadFile(dockerCfgPath)
	if err != nil {
		return "", "", err
	}

	var dockerAuth map[string]manifest.DockerAuthConfig
	if err := json.Unmarshal(j, &dockerAuth); err != nil {
		return "", "", err
	}

	// the official auth uses the full address instead of the hostname
	officialAddress := "https://" + indexServer + "/v1/"
	if c, ok := dockerAuth[officialAddress]; ok {
		return decodeDockerAuth(c.Auth)
	}

	// try the normal case
	if c, ok := dockerAuth[indexServer]; ok {
		return decodeDockerAuth(c.Auth)
	}

	return "", "", nil
}
