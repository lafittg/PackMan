package nodejs

import (
	"sync"

	"github.com/gregoirelafitte/packman/internal/registry"
	"github.com/gregoirelafitte/packman/pkg/types"
	"golang.org/x/sync/errgroup"
)

// fetchCostData fetches cost info for all dependencies from npm.
func fetchCostData(client *registry.Client, deps []types.Dependency) ([]types.CostInfo, error) {
	npm := registry.NewNpmClient(client)

	results := make([]types.CostInfo, len(deps))
	var mu sync.Mutex

	g := new(errgroup.Group)
	g.SetLimit(10) // bounded concurrency

	for i, dep := range deps {
		g.Go(func() error {
			version := dep.ResolvedVersion
			if version == "" {
				version = dep.Version
			}

			info, err := npm.FetchPackageInfo(dep.Name, version)
			if err != nil {
				// Non-fatal: store partial info
				info = types.CostInfo{
					PackageName: dep.Name,
					Version:     version,
				}
			}

			mu.Lock()
			results[i] = info
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return results, err
	}

	return results, nil
}
