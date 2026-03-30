package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gregoirelafitte/packman/internal/analyzer"
	"github.com/gregoirelafitte/packman/internal/tui"
	"github.com/spf13/cobra"
)

var (
	outputJSON bool
	ciMode     bool
)

var rootCmd = &cobra.Command{
	Use:   "packman",
	Short: "Analyze project dependencies: cost, size, and usage",
	Long: `PackMan analyzes your project's dependencies to surface their cost
(disk size, transitive deps, install time) and usage (unused, underused, heavily used).

Run without arguments in a project directory, or specify a path:
  packman analyze
  packman analyze /path/to/project`,
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Analyze dependencies in a project",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := "."
		if len(args) > 0 {
			projectRoot = args[0]
		}

		// Resolve to absolute path
		absPath, err := filepath.Abs(projectRoot)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}

		if outputJSON || ciMode {
			return runNonInteractive(absPath)
		}

		return tui.Run(absPath)
	},
}

func runNonInteractive(projectRoot string) error {
	reports, err := analyzer.Run(projectRoot, func(step string) {
		fmt.Fprintf(os.Stderr, "%s\n", step)
	})
	if err != nil {
		return err
	}

	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(reports)
	}

	// CI mode: check for unused deps
	if ciMode {
		hasUnused := false
		for _, report := range reports {
			for _, dep := range report.Dependencies {
				if dep.Usage.Level == 0 && !dep.Dependency.IsDev {
					fmt.Fprintf(os.Stderr, "UNUSED: %s (%s)\n", dep.Dependency.Name, report.Ecosystem)
					hasUnused = true
				}
			}
		}
		if hasUnused {
			return fmt.Errorf("unused dependencies found")
		}
		fmt.Fprintln(os.Stderr, "All dependencies are in use.")
	}

	return nil
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	analyzeCmd.Flags().BoolVar(&outputJSON, "json", false, "Output results as JSON")
	analyzeCmd.Flags().BoolVar(&ciMode, "ci", false, "CI mode: exit non-zero if unused deps found")
	rootCmd.AddCommand(analyzeCmd)
}
