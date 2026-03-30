package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
)

type BenchmarkResult struct {
	Scenario        string `json:"scenario"`
	Tier            string `json:"tier"`
	TierName        string `json:"tier_name"`
	BuildSuccess    string `json:"build_success"`
	FirstTryCompile string `json:"first_try_compile"`
	TestPass        string `json:"test_pass"`
	CoveragePct     string `json:"coverage_pct"`
	TestFuncCount   string `json:"test_func_count"`
	TableCaseCount  string `json:"table_case_count"`
	WallTime        int    `json:"wall_time_seconds"`
	TokensIn        int    `json:"tokens_in"`
	TokensOut       int    `json:"tokens_out"`
	CostUSD         float64 `json:"cost_usd"`
	NumTurns        int    `json:"num_turns"`
	Error           string `json:"error,omitempty"`
}

func LoadResults(resultsDir string, scenarioFilter string) ([]BenchmarkResult, error) {
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return nil, fmt.Errorf("reading results dir: %w", err)
	}

	var results []BenchmarkResult
	for _, f := range entries {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		// skip claude_output debug files
		if strings.Contains(f.Name(), "claude_output") {
			continue
		}
		if scenarioFilter != "" && !strings.HasPrefix(f.Name(), scenarioFilter+"_") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(resultsDir, f.Name()))
		if err != nil {
			continue
		}
		var r BenchmarkResult
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		results = append(results, r)
	}

	// sort by scenario, then tier
	sort.Slice(results, func(i, j int) bool {
		if results[i].Scenario != results[j].Scenario {
			return results[i].Scenario < results[j].Scenario
		}
		return results[i].Tier < results[j].Tier
	})

	// deduplicate: keep latest result per scenario+tier (last in sorted order by filename)
	seen := make(map[string]int)
	for i, r := range results {
		key := r.Scenario + "_" + r.Tier
		seen[key] = i
	}
	var deduped []BenchmarkResult
	for _, idx := range seen {
		deduped = append(deduped, results[idx])
	}
	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].Scenario != deduped[j].Scenario {
			return deduped[i].Scenario < deduped[j].Scenario
		}
		return deduped[i].Tier < deduped[j].Tier
	})

	return deduped, nil
}

func PrintResults(results []BenchmarkResult) {
	if len(results) == 0 {
		fmt.Println("No benchmark results found.")
		return
	}

	// group by scenario
	scenarios := make(map[string][]BenchmarkResult)
	var order []string
	for _, r := range results {
		if _, exists := scenarios[r.Scenario]; !exists {
			order = append(order, r.Scenario)
		}
		scenarios[r.Scenario] = append(scenarios[r.Scenario], r)
	}

	for _, scenario := range order {
		runs := scenarios[scenario]
		fmt.Printf("=== %s ===\n\n", scenario)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  TIER\tCONFIG\tBUILD\tTESTS\tCOVERAGE\tCASES\tTOKENS\tCOST\tTIME\tTURNS\n")
		fmt.Fprintf(w, "  ────\t──────\t─────\t─────\t────────\t─────\t──────\t────\t────\t─────\n")

		for _, r := range runs {
			build := r.BuildSuccess
			if build == "" {
				build = "false"
			}
			tests := r.TestPass
			if tests == "" {
				tests = "-"
			}
			coverage := r.CoveragePct
			if coverage == "" || coverage == "0" {
				coverage = "-"
			} else {
				coverage += "%"
			}
			cases := r.TableCaseCount
			if cases == "" || cases == "0" {
				cases = "-"
			}
			tokens := fmt.Sprintf("%dk/%dk", r.TokensIn/1000, r.TokensOut/1000)
			if r.TokensIn == 0 {
				tokens = "-"
			}
			cost := fmt.Sprintf("$%.2f", r.CostUSD)
			if r.CostUSD == 0 {
				cost = "-"
			}
			time := fmt.Sprintf("%ds", r.WallTime)
			turns := fmt.Sprintf("%d", r.NumTurns)

			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.Tier, r.TierName, build, tests, coverage, cases, tokens, cost, time, turns)
		}
		w.Flush()
		fmt.Println()
	}
}
