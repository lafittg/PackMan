package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/gregoirelafitte/packman/internal/cost"
	"github.com/gregoirelafitte/packman/internal/plugin"
	"github.com/gregoirelafitte/packman/pkg/types"
)

// ProgressFunc is called with status updates during analysis.
type ProgressFunc func(step string)

// Run performs the full analysis pipeline on the given project root.
func Run(projectRoot string, onProgress ProgressFunc) ([]types.ProjectReport, error) {
	if onProgress == nil {
		onProgress = func(string) {}
	}

	onProgress("Detecting ecosystems...")
	detected := plugin.DetectAll(projectRoot)
	if len(detected) == 0 {
		return nil, fmt.Errorf("no supported ecosystem detected in %s", projectRoot)
	}

	var reports []types.ProjectReport

	for _, p := range detected {
		report, err := analyzeEcosystem(projectRoot, p, onProgress)
		if err != nil {
			return nil, fmt.Errorf("analyzing %s: %w", p.Name(), err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func analyzeEcosystem(projectRoot string, p plugin.Plugin, onProgress ProgressFunc) (types.ProjectReport, error) {
	report := types.ProjectReport{
		ProjectRoot: projectRoot,
		Ecosystem:   p.Name(),
		AnalyzedAt:  time.Now(),
	}

	// Step 1: Parse dependencies
	onProgress(fmt.Sprintf("[%s] Parsing dependencies...", p.Name()))
	deps, err := p.ParseDependencies(projectRoot)
	if err != nil {
		return report, fmt.Errorf("parsing dependencies: %w", err)
	}

	if len(deps) == 0 {
		return report, nil
	}

	// Step 2: Fetch cost data and analyze usage in parallel
	var (
		costResults  []types.CostInfo
		usageResults []types.UsageInfo
		costErr      error
		usageErr     error
		wg           sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		onProgress(fmt.Sprintf("[%s] Fetching package metadata (%d packages)...", p.Name(), len(deps)))
		costResults, costErr = p.FetchCostData(deps)
	}()

	go func() {
		defer wg.Done()
		onProgress(fmt.Sprintf("[%s] Analyzing source code usage...", p.Name()))
		usageResults, usageErr = p.AnalyzeUsage(projectRoot, deps)
	}()

	wg.Wait()

	if costErr != nil {
		return report, fmt.Errorf("fetching cost data: %w", costErr)
	}
	if usageErr != nil {
		return report, fmt.Errorf("analyzing usage: %w", usageErr)
	}

	// Step 3: Combine results
	onProgress(fmt.Sprintf("[%s] Computing health scores...", p.Name()))
	costMap := make(map[string]types.CostInfo, len(costResults))
	for _, c := range costResults {
		costMap[c.PackageName] = c
	}

	usageMap := make(map[string]types.UsageInfo, len(usageResults))
	for _, u := range usageResults {
		usageMap[u.PackageName] = u
	}

	for _, dep := range deps {
		c := costMap[dep.Name]
		u := usageMap[dep.Name]
		health := cost.ComputeHealthScore(c, u)

		report.Dependencies = append(report.Dependencies, types.AnalysisResult{
			Dependency:  dep,
			Cost:        c,
			Usage:       u,
			HealthScore: health,
		})
	}

	report.Summary = cost.ComputeSummary(report.Dependencies)

	return report, nil
}
