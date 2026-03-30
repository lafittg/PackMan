package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/gregoirelafitte/packman/pkg/types"
)

const goProxyURL = "https://proxy.golang.org"
const goPkgDevAPI = "https://api.deps.dev/v3alpha"

// GoProxyClient fetches package metadata from the Go module proxy.
type GoProxyClient struct {
	client *Client
}

// NewGoProxyClient creates a new Go module proxy client.
func NewGoProxyClient(client *Client) *GoProxyClient {
	return &GoProxyClient{client: client}
}

// goModuleInfo represents the JSON response from proxy.golang.org/<module>/@v/<version>.info
type goModuleInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

// goModFile represents a parsed go.mod for extracting dependencies.
type goModFile struct {
	Module  goModModule   `json:"Module"`
	Require []goModEntry  `json:"Require"`
}

type goModModule struct {
	Path string `json:"Path"`
}

type goModEntry struct {
	Path     string `json:"Path"`
	Version  string `json:"Version"`
	Indirect bool   `json:"Indirect"`
}

// depsDevVersionResponse is the response from deps.dev API for module version info.
type depsDevVersionResponse struct {
	VersionKey struct {
		System  string `json:"system"`
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"versionKey"`
	PublishedAt string `json:"publishedAt"`
	Links       []struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	} `json:"links"`
}

// depsDevDepsResponse is the response from deps.dev for dependency graph.
type depsDevDepsResponse struct {
	Nodes []struct {
		VersionKey struct {
			System  string `json:"system"`
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"versionKey"`
		Relation string `json:"relation"`
	} `json:"nodes"`
}

// FetchPackageInfo fetches cost information for a single Go module.
func (g *GoProxyClient) FetchPackageInfo(modulePath, version string) (types.CostInfo, error) {
	info := types.CostInfo{
		PackageName: modulePath,
		Version:     version,
	}

	// Encode module path for URL (uppercase → !lowercase per Go proxy spec)
	encodedPath := encodeModulePath(modulePath)

	// Clean version
	cleanVer := version
	if cleanVer == "" {
		// Fetch the latest version from proxy
		latestURL := fmt.Sprintf("%s/%s/@latest", goProxyURL, encodedPath)
		var modInfo goModuleInfo
		if err := g.client.GetJSON(latestURL, &modInfo); err != nil {
			return info, fmt.Errorf("fetching latest version for %s: %w", modulePath, err)
		}
		cleanVer = modInfo.Version
		info.Version = cleanVer
	}

	// Fetch version info (includes timestamp)
	infoURL := fmt.Sprintf("%s/%s/@v/%s.info", goProxyURL, encodedPath, cleanVer)
	var modInfo goModuleInfo
	if err := g.client.GetJSON(infoURL, &modInfo); err == nil {
		info.LastPublish = modInfo.Time
		if info.Version == "" {
			info.Version = modInfo.Version
		}
	}

	// Get the zip size (this is the publish/install size for Go modules)
	zipURL := fmt.Sprintf("%s/%s/@v/%s.zip", goProxyURL, encodedPath, cleanVer)
	if size, err := g.client.HeadContentLength(zipURL); err == nil && size > 0 {
		info.PublishSize = size
		// Go modules are source code, install size ≈ publish size (zip decompressed ~2x)
		info.InstallSize = int64(float64(size) * 2.0)
	}

	// Estimate install time (assume 10 MB/s)
	if info.PublishSize > 0 {
		info.EstInstallTime = time.Duration(float64(info.PublishSize) / (10 * 1024 * 1024) * float64(time.Second))
	}

	// Use deps.dev API for dependency info (much richer than parsing go.mod ourselves)
	g.fetchDepsDevInfo(&info, modulePath, cleanVer)

	return info, nil
}

// fetchDepsDevInfo uses the deps.dev API to get dependency and popularity data.
func (g *GoProxyClient) fetchDepsDevInfo(info *types.CostInfo, modulePath, version string) {
	encodedModule := encodeModulePath(modulePath)

	// Fetch dependency graph
	depsURL := fmt.Sprintf("%s/systems/go/packages/%s/versions/%s:dependencies",
		goPkgDevAPI, encodedModule, version)
	var depsResp depsDevDepsResponse
	if err := g.client.GetJSON(depsURL, &depsResp); err == nil {
		var depTree []string
		directCount := 0
		for _, node := range depsResp.Nodes {
			if node.VersionKey.Name == modulePath {
				continue // skip self
			}
			if node.Relation == "DIRECT" {
				directCount++
			}
			depTree = append(depTree, node.VersionKey.Name)
		}
		info.DirectDeps = directCount
		info.TransitiveDeps = len(depTree)
		info.DepTree = depTree
	}
}

// encodeModulePath encodes a Go module path for use in proxy URLs.
// Per the Go module proxy spec, uppercase letters are encoded as !lowercase.
func encodeModulePath(path string) string {
	var b strings.Builder
	for _, c := range path {
		if c >= 'A' && c <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(c + ('a' - 'A'))
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}
