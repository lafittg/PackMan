package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/gregoirelafitte/packman/pkg/types"
)

const npmRegistryURL = "https://registry.npmjs.org"
const npmDownloadsURL = "https://api.npmjs.org/downloads/point/last-week"

// NpmClient fetches package metadata from the npm registry.
type NpmClient struct {
	client *Client
}

// NewNpmClient creates a new npm registry client.
func NewNpmClient(client *Client) *NpmClient {
	return &NpmClient{client: client}
}

// npmPackageResponse represents relevant fields from the npm registry API.
type npmPackageResponse struct {
	Name    string                        `json:"name"`
	Version string                        `json:"version"`
	Dist    npmDist                       `json:"dist"`
	Dependencies map[string]string        `json:"dependencies"`
	Time    map[string]time.Time          `json:"time"`
}

type npmDist struct {
	Tarball      string `json:"tarball"`
	UnpackedSize int64  `json:"unpackedSize"`
}

type npmDownloadsResponse struct {
	Downloads int64 `json:"downloads"`
}

// npmFullPackageResponse is the full package doc (used for transitive dep counting).
type npmFullPackageResponse struct {
	Name     string                        `json:"name"`
	Versions map[string]npmVersionInfo     `json:"versions"`
	Time     map[string]time.Time          `json:"time"`
}

type npmVersionInfo struct {
	Version      string            `json:"version"`
	Dist         npmDist           `json:"dist"`
	Dependencies map[string]string `json:"dependencies"`
}

// FetchPackageInfo fetches cost information for a single package.
func (n *NpmClient) FetchPackageInfo(name, version string) (types.CostInfo, error) {
	info := types.CostInfo{
		PackageName: name,
		Version:     version,
	}

	// Clean version (remove ^ ~ >= etc.)
	cleanVer := cleanVersion(version)

	// Fetch specific version metadata
	url := fmt.Sprintf("%s/%s/%s", npmRegistryURL, name, cleanVer)
	var pkgResp npmPackageResponse
	if err := n.client.GetJSON(url, &pkgResp); err != nil {
		// Try fetching latest if specific version fails
		url = fmt.Sprintf("%s/%s/latest", npmRegistryURL, name)
		if err2 := n.client.GetJSON(url, &pkgResp); err2 != nil {
			return info, fmt.Errorf("fetching %s: %w", name, err)
		}
	}

	info.Version = pkgResp.Version
	info.InstallSize = pkgResp.Dist.UnpackedSize
	info.DirectDeps = len(pkgResp.Dependencies)

	// Get tarball size
	if pkgResp.Dist.Tarball != "" {
		if size, err := n.client.HeadContentLength(pkgResp.Dist.Tarball); err == nil && size > 0 {
			info.PublishSize = size
		}
	}

	// Fetch weekly downloads
	dlURL := fmt.Sprintf("%s/%s", npmDownloadsURL, name)
	var dlResp npmDownloadsResponse
	if err := n.client.GetJSON(dlURL, &dlResp); err == nil {
		info.WeeklyDownloads = dlResp.Downloads
	}

	// Estimate install time (assume 10 MB/s average)
	if info.PublishSize > 0 {
		info.EstInstallTime = time.Duration(float64(info.PublishSize) / (10 * 1024 * 1024) * float64(time.Second))
	}

	// Count transitive dependencies from the full package doc
	transitive, depTree := n.countTransitiveDeps(name, cleanVer)
	info.TransitiveDeps = transitive
	info.DepTree = depTree

	return info, nil
}

// countTransitiveDeps does a BFS to count all transitive dependencies.
func (n *NpmClient) countTransitiveDeps(name, version string) (int, []string) {
	visited := map[string]bool{}
	queue := []string{name}
	var depTree []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if current != name {
			depTree = append(depTree, current)
		}

		url := fmt.Sprintf("%s/%s/latest", npmRegistryURL, current)
		var resp npmPackageResponse
		if err := n.client.GetJSON(url, &resp); err != nil {
			continue
		}

		for dep := range resp.Dependencies {
			if !visited[dep] {
				queue = append(queue, dep)
			}
		}
	}

	return len(depTree), depTree
}

// cleanVersion strips semver prefixes like ^, ~, >=, etc.
func cleanVersion(v string) string {
	v = strings.TrimLeft(v, "^~>=<! ")
	// Handle ranges like "1.0.0 - 2.0.0" by taking the first part
	if idx := strings.Index(v, " "); idx != -1 {
		v = v[:idx]
	}
	return v
}
