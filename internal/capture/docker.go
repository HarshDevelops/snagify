package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// composeFileNames are the compose files looked for in a project root.
var composeFileNames = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

// captureDocker gathers non-invasive Docker state. It never starts containers
// or mutates anything. If Docker is unavailable, Runtime.Present is false.
func captureDocker(root string) model.DockerInfo {
	d := model.DockerInfo{
		Runtime: probe("docker", "--version"),
		Compose: detectCompose(),
	}

	for _, name := range composeFileNames {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			d.ComposeFiles = append(d.ComposeFiles, name)
		}
	}

	if !d.Runtime.Present {
		return d
	}

	d.Images = dockerImages()
	d.Containers = dockerContainers()
	return d
}

// detectCompose prefers the `docker compose` plugin and falls back to the
// legacy `docker-compose` binary.
func detectCompose() model.VersionInfo {
	if out, ok := runCmd("docker", "compose", "version", "--short"); ok {
		return model.VersionInfo{Present: true, Version: parseVersion(out)}
	}
	if out, ok := runCmd("docker", "compose", "version"); ok {
		return model.VersionInfo{Present: true, Version: parseVersion(out)}
	}
	return probe("docker-compose", "--version")
}

// dockerImages lists local images via JSON output, tolerating older daemons.
func dockerImages() []model.DockerImageInfo {
	out, ok := runCmd("docker", "images", "--format", "{{json .}}")
	if !ok {
		return nil
	}
	var images []model.DockerImageInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Repository string `json:"Repository"`
			Tag        string `json:"Tag"`
			ID         string `json:"ID"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		if row.Repository == "<none>" {
			continue
		}
		images = append(images, model.DockerImageInfo{
			Repository: row.Repository,
			Tag:        row.Tag,
			ID:         row.ID,
		})
	}
	return images
}

// dockerContainers lists running containers via JSON output.
func dockerContainers() []model.DockerContainerInfo {
	out, ok := runCmd("docker", "ps", "--format", "{{json .}}")
	if !ok {
		return nil
	}
	var containers []model.DockerContainerInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Names  string `json:"Names"`
			Image  string `json:"Image"`
			Status string `json:"Status"`
			Ports  string `json:"Ports"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		c := model.DockerContainerInfo{
			Name:   row.Names,
			Image:  row.Image,
			Status: row.Status,
		}
		if row.Ports != "" {
			c.Ports = strings.Split(row.Ports, ", ")
		}
		containers = append(containers, c)
	}
	return containers
}
