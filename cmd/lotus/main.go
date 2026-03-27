package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
	"github.com/texliao/lotus/catalogdata"
	"github.com/texliao/lotus/internal/analyzer"
	"github.com/texliao/lotus/internal/benchmark"
	"github.com/texliao/lotus/internal/catalog"
	"github.com/texliao/lotus/internal/generator"
	"github.com/texliao/lotus/internal/recommender"
)

var dataFS fs.FS

func main() {
	dataFS, _ = fs.Sub(catalogdata.FS, "data")
	catalog.SetFS(dataFS)

	rootCmd := &cobra.Command{
		Use:   "lotus",
		Short: "Minimal AI config recommender",
		Long:  "Lotus analyzes your project and recommends the minimum effective set of AI coding assistant configurations.",
	}

	analyzeCmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Detect stack and scan existing AI config",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			profile, err := analyzer.Analyze(path)
			if err != nil {
				return err
			}
			profile.Print()
			return nil
		},
	}

	recommendCmd := &cobra.Command{
		Use:   "recommend [path]",
		Short: "Analyze project and recommend optimal AI config",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			profile, err := analyzer.Analyze(path)
			if err != nil {
				return err
			}
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			recs := recommender.Recommend(profile, cat)
			recs.Print()
			return nil
		},
	}

	applyCmd := &cobra.Command{
		Use:   "apply [path]",
		Short: "Apply recommended config to project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			profile, err := analyzer.Analyze(path)
			if err != nil {
				return err
			}
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			recs := recommender.Recommend(profile, cat)
			return generator.Apply(path, recs, dryRun)
		},
	}
	applyCmd.Flags().Bool("dry-run", false, "Preview changes without writing files")

	catalogCmd := &cobra.Command{
		Use:   "catalog",
		Short: "Browse the lotus catalog",
	}

	catalogListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all catalog entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			kind, _ := cmd.Flags().GetString("kind")
			stack, _ := cmd.Flags().GetString("stack")
			cat.PrintList(kind, stack)
			return nil
		},
	}
	catalogListCmd.Flags().String("kind", "", "Filter by kind (skill, agent, bundle, source, mcp-server, hook)")
	catalogListCmd.Flags().String("stack", "", "Filter by stack (go, node, python, rust)")

	catalogShowCmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Show details of a catalog entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			return cat.PrintShow(args[0])
		},
	}

	// benchmark commands
	benchmarkCmd := &cobra.Command{
		Use:   "benchmark",
		Short: "View and compare benchmark scenarios",
	}

	benchmarkListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all benchmark scenarios",
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarios, err := benchmark.LoadScenarios(dataFS)
			if err != nil {
				return err
			}
			if len(scenarios) == 0 {
				fmt.Println("No benchmark scenarios found.")
				return nil
			}
			benchmark.PrintList(scenarios)
			return nil
		},
	}

	benchmarkShowCmd := &cobra.Command{
		Use:   "show [scenario-id]",
		Short: "Show scenario details and results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarios, err := benchmark.LoadScenarios(dataFS)
			if err != nil {
				return err
			}
			s := benchmark.FindScenario(scenarios, args[0])
			if s == nil {
				return fmt.Errorf("scenario %q not found", args[0])
			}
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			benchmark.PrintShow(s, cat)
			return nil
		},
	}

	benchmarkCompareCmd := &cobra.Command{
		Use:   "compare [id1] [id2]",
		Short: "Compare two solutions on a scenario",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarioID, _ := cmd.Flags().GetString("scenario")
			if scenarioID == "" {
				return fmt.Errorf("--scenario flag is required")
			}
			cat, err := catalog.Load()
			if err != nil {
				return err
			}
			return benchmark.PrintCompare(args[0], args[1], scenarioID, cat)
		},
	}
	benchmarkCompareCmd.Flags().String("scenario", "", "Scenario ID to compare against (required)")

	benchmarkCmd.AddCommand(benchmarkListCmd, benchmarkShowCmd, benchmarkCompareCmd)
	catalogCmd.AddCommand(catalogListCmd, catalogShowCmd)
	rootCmd.AddCommand(analyzeCmd, recommendCmd, applyCmd, catalogCmd, benchmarkCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
