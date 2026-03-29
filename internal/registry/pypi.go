package registry

import (
	"fmt"
	"strings"
	"time"

	"github.com/gregoirelafitte/packman/pkg/types"
)

const pypiRegistryURL = "https://pypi.org/pypi"
const pypistatsURL = "https://pypistats.org/api/packages"

// PypiClient fetches package metadata from the PyPI registry.
type PypiClient struct {
	client *Client
}

// NewPypiClient creates a new PyPI registry client.
func NewPypiClient(client *Client) *PypiClient {
	return &PypiClient{client: client}
}

// pypiPackageResponse represents relevant fields from the PyPI JSON API.
type pypiPackageResponse struct {
	Info pypiInfo              `json:"info"`
	URLs []pypiReleaseFileInfo `json:"urls"`
}

type pypiInfo struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	RequiresDist    []string `json:"requires_dist"`
	Summary         string `json:"summary"`
	ProjectURL      string `json:"project_url"`
	PackageURL      string `json:"package_url"`
}

type pypiReleaseFileInfo struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	PackageType string `json:"packagetype"`
	UploadTime  string `json:"upload_time_iso_8601"`
}

type pypistatsResponse struct {
	Data []pypistatsData `json:"data"`
}

type pypistatsData struct {
	Category  string `json:"category"`
	Downloads int64  `json:"downloads"`
}

// FetchPackageInfo fetches cost information for a single Python package.
func (p *PypiClient) FetchPackageInfo(name, version string) (types.CostInfo, error) {
	info := types.CostInfo{
		PackageName: name,
		Version:     version,
	}

	// Normalize package name for PyPI (PEP 503: lowercase, replace - and _ with -)
	normalizedName := normalizePyPIName(name)

	// Fetch specific version or latest
	var url string
	cleanVer := cleanPythonVersion(version)
	if cleanVer != "" {
		url = fmt.Sprintf("%s/%s/%s/json", pypiRegistryURL, normalizedName, cleanVer)
	} else {
		url = fmt.Sprintf("%s/%s/json", pypiRegistryURL, normalizedName)
	}

	var pkgResp pypiPackageResponse
	if err := p.client.GetJSON(url, &pkgResp); err != nil {
		// Try fetching latest if specific version fails
		url = fmt.Sprintf("%s/%s/json", pypiRegistryURL, normalizedName)
		if err2 := p.client.GetJSON(url, &pkgResp); err2 != nil {
			return info, fmt.Errorf("fetching %s: %w", name, err)
		}
	}

	info.Version = pkgResp.Info.Version

	// Calculate sizes from release files
	// Prefer wheel (.whl), then sdist (.tar.gz)
	var publishSize, installSize int64
	for _, f := range pkgResp.URLs {
		switch f.PackageType {
		case "bdist_wheel":
			if publishSize == 0 || f.Size < publishSize {
				publishSize = f.Size
				// Wheel install size ≈ 1.2x the wheel file size (already unpacked mostly)
				installSize = int64(float64(f.Size) * 1.2)
			}
		case "sdist":
			if publishSize == 0 {
				publishSize = f.Size
				// Sdist install size ≈ 2x (needs compilation, source expansion)
				installSize = int64(float64(f.Size) * 2.0)
			}
		}
	}
	info.PublishSize = publishSize
	info.InstallSize = installSize

	// Parse upload time from first URL
	if len(pkgResp.URLs) > 0 && pkgResp.URLs[0].UploadTime != "" {
		if t, err := time.Parse(time.RFC3339, pkgResp.URLs[0].UploadTime); err == nil {
			info.LastPublish = t
		}
	}

	// Count direct dependencies from requires_dist
	directDeps := countDirectDeps(pkgResp.Info.RequiresDist)
	info.DirectDeps = directDeps

	// Fetch weekly downloads from pypistats
	dlURL := fmt.Sprintf("%s/%s/recent?period=week", pypistatsURL, normalizedName)
	var dlResp pypistatsResponse
	if err := p.client.GetJSON(dlURL, &dlResp); err == nil {
		for _, d := range dlResp.Data {
			if d.Category == "without_mirrors" {
				info.WeeklyDownloads = d.Downloads
				break
			}
		}
		// Fallback to total if without_mirrors not found
		if info.WeeklyDownloads == 0 {
			for _, d := range dlResp.Data {
				info.WeeklyDownloads += d.Downloads
			}
		}
	}

	// Estimate install time (assume 10 MB/s average)
	if info.PublishSize > 0 {
		info.EstInstallTime = time.Duration(float64(info.PublishSize) / (10 * 1024 * 1024) * float64(time.Second))
	}

	// Count transitive dependencies via BFS
	transitive, depTree := p.countTransitiveDeps(name)
	info.TransitiveDeps = transitive
	info.DepTree = depTree

	return info, nil
}

// countTransitiveDeps performs BFS through PyPI requires_dist to count all transitive deps.
func (p *PypiClient) countTransitiveDeps(name string) (int, []string) {
	visited := map[string]bool{}
	queue := []string{normalizePyPIName(name)}
	var depTree []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		normalized := normalizePyPIName(current)
		if visited[normalized] {
			continue
		}
		visited[normalized] = true

		if normalized != normalizePyPIName(name) {
			depTree = append(depTree, current)
		}

		// Limit BFS depth to avoid excessive API calls
		if len(visited) > 100 {
			break
		}

		url := fmt.Sprintf("%s/%s/json", pypiRegistryURL, normalized)
		var resp pypiPackageResponse
		if err := p.client.GetJSON(url, &resp); err != nil {
			continue
		}

		for _, req := range resp.Info.RequiresDist {
			depName := parseRequiresDist(req)
			if depName == "" {
				continue
			}
			// Skip extras-only dependencies (conditional on extra markers)
			if isExtraOnly(req) {
				continue
			}
			normalizedDep := normalizePyPIName(depName)
			if !visited[normalizedDep] {
				queue = append(queue, depName)
			}
		}
	}

	return len(depTree), depTree
}

// countDirectDeps counts non-extra dependencies from requires_dist.
func countDirectDeps(requiresDist []string) int {
	count := 0
	for _, req := range requiresDist {
		if !isExtraOnly(req) {
			count++
		}
	}
	return count
}

// parseRequiresDist extracts the package name from a requires_dist entry.
// Format: "package-name (>=1.0)" or "package-name; extra == 'dev'"
func parseRequiresDist(entry string) string {
	// Trim whitespace
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}

	// Take everything before version specifiers or markers
	for i, c := range entry {
		if c == ' ' || c == '(' || c == ';' || c == '>' || c == '<' || c == '=' || c == '!' || c == '[' {
			return strings.TrimSpace(entry[:i])
		}
	}
	return entry
}

// isExtraOnly checks if a requires_dist entry is conditional on an extra marker.
// These are optional dependencies that are only installed with extras_require.
func isExtraOnly(entry string) bool {
	lower := strings.ToLower(entry)
	return strings.Contains(lower, "extra ==") || strings.Contains(lower, "extra ==")
}

// normalizePyPIName normalizes a Python package name according to PEP 503.
// Lowercases and replaces [-_.] with -.
func normalizePyPIName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

// cleanPythonVersion strips Python version specifiers to get a clean version string.
func cleanPythonVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimLeft(v, "~=!<>^")
	if idx := strings.IndexAny(v, ",;"); idx != -1 {
		v = v[:idx]
	}
	v = strings.TrimSpace(v)
	// Must look like a version (starts with digit)
	if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
		return v
	}
	return ""
}
