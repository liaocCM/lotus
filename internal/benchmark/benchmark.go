package benchmark

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/texliao/lotus/internal/catalog"
	"gopkg.in/yaml.v3"
)

type Scenario struct {
	ID          string        `yaml:"id"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Category    string        `yaml:"category"`
	Difficulty  string        `yaml:"difficulty"`
	Stack       ScenarioStack `yaml:"stack"`
	Prompt      string        `yaml:"prompt"`
	Acceptance  []string      `yaml:"acceptance_criteria"`
	Rubric      map[string]string `yaml:"quality_rubric"`
	Competitors []string      `yaml:"competitors"`
}

type ScenarioStack struct {
	Language  string `yaml:"language"`
	Framework string `yaml:"framework,omitempty"`
	Database  string `yaml:"database,omitempty"`
}

func LoadScenarios(catalogFS fs.FS) ([]Scenario, error) {
	entries, err := fs.ReadDir(catalogFS, "scenarios")
	if err != nil {
		return nil, nil // no scenarios dir is fine
	}

	var scenarios []Scenario
	for _, f := range entries {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile(catalogFS, filepath.Join("scenarios", f.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading scenario %s: %w", f.Name(), err)
		}
		var s Scenario
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parsing scenario %s: %w", f.Name(), err)
		}
		scenarios = append(scenarios, s)
	}
	return scenarios, nil
}

func PrintList(scenarios []Scenario) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tCATEGORY\tDIFFICULTY\tCOMPETITORS\n")
	fmt.Fprintf(w, "──\t────\t────────\t──────────\t───────────\n")
	for _, s := range scenarios {
		comps := strings.Join(s.Competitors, ", ")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.Name, s.Category, s.Difficulty, comps)
	}
	w.Flush()
}

func PrintShow(scenario *Scenario, cat *catalog.Catalog) {
	fmt.Printf("Scenario:    %s\n", scenario.ID)
	fmt.Printf("Name:        %s\n", scenario.Name)
	fmt.Printf("Category:    %s\n", scenario.Category)
	fmt.Printf("Difficulty:  %s\n", scenario.Difficulty)
	fmt.Printf("Stack:       %s", scenario.Stack.Language)
	if scenario.Stack.Framework != "" {
		fmt.Printf(" + %s", scenario.Stack.Framework)
	}
	if scenario.Stack.Database != "" {
		fmt.Printf(" + %s", scenario.Stack.Database)
	}
	fmt.Println()
	fmt.Printf("\nDescription:\n  %s\n", scenario.Description)
	fmt.Printf("\nPrompt:\n  %s\n", scenario.Prompt)

	fmt.Printf("\nAcceptance criteria:\n")
	for _, a := range scenario.Acceptance {
		fmt.Printf("  - %s\n", a)
	}

	fmt.Printf("\nQuality rubric:\n")
	for k, v := range scenario.Rubric {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// show benchmark results from competing entries
	fmt.Printf("\nResults:\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  SOLUTION\tTOKENS IN\tTOKENS OUT\tTIME\tQUALITY\tEFFICIENCY\n")
	fmt.Fprintf(w, "  ────────\t─────────\t──────────\t────\t───────\t──────────\n")

	for _, compID := range scenario.Competitors {
		entry := cat.Get(compID)
		if entry == nil {
			continue
		}
		for _, b := range entry.Benchmarks {
			if b.Scenario == scenario.ID {
				totalTokens := float64(b.TokensIn + b.TokensOut)
				efficiency := b.Quality / (totalTokens / 10000) // quality per 10k tokens
				fmt.Fprintf(w, "  %s\t%dk\t%dk\t%ds\t%.1f/5\t%.2f\n",
					compID, b.TokensIn/1000, b.TokensOut/1000, b.WallTime, b.Quality, efficiency)
			}
		}
	}
	w.Flush()
}

func PrintCompare(id1, id2 string, scenarioID string, cat *catalog.Catalog) error {
	e1 := cat.Get(id1)
	e2 := cat.Get(id2)
	if e1 == nil {
		return fmt.Errorf("entry %q not found", id1)
	}
	if e2 == nil {
		return fmt.Errorf("entry %q not found", id2)
	}

	var b1, b2 *catalog.Benchmark
	for i := range e1.Benchmarks {
		if e1.Benchmarks[i].Scenario == scenarioID {
			b1 = &e1.Benchmarks[i]
			break
		}
	}
	for i := range e2.Benchmarks {
		if e2.Benchmarks[i].Scenario == scenarioID {
			b2 = &e2.Benchmarks[i]
			break
		}
	}

	if b1 == nil {
		return fmt.Errorf("%s has no benchmark data for scenario %s", id1, scenarioID)
	}
	if b2 == nil {
		return fmt.Errorf("%s has no benchmark data for scenario %s", id2, scenarioID)
	}

	fmt.Printf("Comparison: %s vs %s on %s\n\n", id1, id2, scenarioID)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  METRIC\t%s\t%s\tDIFF\n", strings.ToUpper(id1), strings.ToUpper(id2))
	fmt.Fprintf(w, "  ──────\t%s\t%s\t────\n", strings.Repeat("─", len(id1)), strings.Repeat("─", len(id2)))

	fmt.Fprintf(w, "  Tokens in\t%dk\t%dk\t%+dk\n",
		b1.TokensIn/1000, b2.TokensIn/1000, (b1.TokensIn-b2.TokensIn)/1000)
	fmt.Fprintf(w, "  Tokens out\t%dk\t%dk\t%+dk\n",
		b1.TokensOut/1000, b2.TokensOut/1000, (b1.TokensOut-b2.TokensOut)/1000)
	fmt.Fprintf(w, "  Total tokens\t%dk\t%dk\t%+dk\n",
		(b1.TokensIn+b1.TokensOut)/1000, (b2.TokensIn+b2.TokensOut)/1000,
		((b1.TokensIn+b1.TokensOut)-(b2.TokensIn+b2.TokensOut))/1000)
	fmt.Fprintf(w, "  Wall time\t%ds\t%ds\t%+ds\n",
		b1.WallTime, b2.WallTime, b1.WallTime-b2.WallTime)
	fmt.Fprintf(w, "  Quality\t%.1f/5\t%.1f/5\t%+.1f\n",
		b1.Quality, b2.Quality, b1.Quality-b2.Quality)

	eff1 := b1.Quality / (float64(b1.TokensIn+b1.TokensOut) / 10000)
	eff2 := b2.Quality / (float64(b2.TokensIn+b2.TokensOut) / 10000)
	fmt.Fprintf(w, "  Efficiency\t%.2f\t%.2f\t%+.2f\n", eff1, eff2, eff1-eff2)
	w.Flush()

	// verdict
	fmt.Println()
	if eff1 > eff2 {
		fmt.Printf("  %s is more token-efficient (%.2f vs %.2f quality per 10k tokens)\n", id1, eff1, eff2)
	} else {
		fmt.Printf("  %s is more token-efficient (%.2f vs %.2f quality per 10k tokens)\n", id2, eff2, eff1)
	}
	if b1.Quality > b2.Quality {
		fmt.Printf("  %s produces higher quality (%.1f vs %.1f)\n", id1, b1.Quality, b2.Quality)
	} else if b2.Quality > b1.Quality {
		fmt.Printf("  %s produces higher quality (%.1f vs %.1f)\n", id2, b2.Quality, b1.Quality)
	}

	return nil
}

func FindScenario(scenarios []Scenario, id string) *Scenario {
	for i := range scenarios {
		if scenarios[i].ID == id {
			return &scenarios[i]
		}
	}
	return nil
}
