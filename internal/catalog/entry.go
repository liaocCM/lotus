package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

var catalogFS fs.FS

func SetFS(f fs.FS) {
	catalogFS = f
}

type Entry struct {
	ID       string   `yaml:"id"`
	Kind     string   `yaml:"kind"`
	Name     string   `yaml:"name"`
	Source   Source   `yaml:"source"`
	UseCases []string `yaml:"use_cases"`
	Stacks   []string `yaml:"stacks"`
	Weight   string   `yaml:"weight,omitempty"`
	Contains []ContainedItem `yaml:"contains,omitempty"`
	Requires   Requirements    `yaml:"requires"`
	Lotus      LotusMeta       `yaml:"lotus"`
	Benchmarks []Benchmark     `yaml:"benchmarks,omitempty"`
}

type Benchmark struct {
	Scenario string  `yaml:"scenario"`
	TokensIn int     `yaml:"tokens_in"`
	TokensOut int    `yaml:"tokens_out"`
	WallTime int     `yaml:"wall_time_seconds"`
	Quality  float64 `yaml:"quality_score"`
}

type Source struct {
	Registry string `yaml:"registry"`
	Repo     string `yaml:"repo"`
	URL      string `yaml:"url"`
}

type ContainedItem struct {
	Kind  string `yaml:"kind"`
	Count int    `yaml:"count"`
}

type Requirements struct {
	Tools      []string `yaml:"tools"`
	MCPServers []string `yaml:"mcp_servers"`
	Runtime    []string `yaml:"runtime"`
}

type LotusMeta struct {
	Tier          string   `yaml:"tier"`
	Notes         string   `yaml:"notes"`
	ConflictsWith []string `yaml:"conflicts_with"`
	PairsWellWith []string `yaml:"pairs_well_with"`
}

type Catalog struct {
	Entries       []Entry
	byID          map[string]*Entry
	byUseCase     map[string][]*Entry
	byStack       map[string][]*Entry
	byKind        map[string][]*Entry
}

func Load() (*Catalog, error) {
	c := &Catalog{
		byID:      make(map[string]*Entry),
		byUseCase: make(map[string][]*Entry),
		byStack:   make(map[string][]*Entry),
		byKind:    make(map[string][]*Entry),
	}

	dirs := []string{"skills", "agents", "mcp-servers", "hooks", "bundles", "sources"}
	for _, dir := range dirs {
		entries, err := fs.ReadDir(catalogFS, dir)
		if err != nil {
			continue
		}
		for _, f := range entries {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			data, err := fs.ReadFile(catalogFS, filepath.Join(dir, f.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading %s/%s: %w", dir, f.Name(), err)
			}
			var entry Entry
			if err := yaml.Unmarshal(data, &entry); err != nil {
				return nil, fmt.Errorf("parsing %s/%s: %w", dir, f.Name(), err)
			}
			c.Entries = append(c.Entries, entry)
		}
	}

	for i := range c.Entries {
		e := &c.Entries[i]
		c.byID[e.ID] = e
		c.byKind[e.Kind] = append(c.byKind[e.Kind], e)
		for _, uc := range e.UseCases {
			c.byUseCase[uc] = append(c.byUseCase[uc], e)
		}
		for _, s := range e.Stacks {
			c.byStack[s] = append(c.byStack[s], e)
		}
	}

	return c, nil
}

func (c *Catalog) Get(id string) *Entry {
	return c.byID[id]
}

func (c *Catalog) ByUseCase(uc string) []*Entry {
	return c.byUseCase[uc]
}

func (c *Catalog) ByStack(stack string) []*Entry {
	return c.byStack[stack]
}

func (c *Catalog) ByKind(kind string) []*Entry {
	return c.byKind[kind]
}

func (c *Catalog) PrintList(kindFilter, stackFilter string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tKIND\tNAME\tSTACKS\tTIER\n")
	fmt.Fprintf(w, "──\t────\t────\t──────\t────\n")
	for _, e := range c.Entries {
		if kindFilter != "" && e.Kind != kindFilter {
			continue
		}
		if stackFilter != "" && !containsStr(e.Stacks, stackFilter) {
			continue
		}
		stacks := strings.Join(e.Stacks, ", ")
		if len(stacks) == 0 {
			stacks = "any"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.ID, e.Kind, e.Name, stacks, e.Lotus.Tier)
	}
	w.Flush()
}

func (c *Catalog) PrintShow(id string) error {
	e := c.Get(id)
	if e == nil {
		return fmt.Errorf("entry %q not found", id)
	}

	fmt.Printf("ID:        %s\n", e.ID)
	fmt.Printf("Name:      %s\n", e.Name)
	fmt.Printf("Kind:      %s\n", e.Kind)
	fmt.Printf("Tier:      %s\n", e.Lotus.Tier)
	fmt.Printf("Source:    %s\n", e.Source.URL)

	if len(e.Stacks) > 0 {
		fmt.Printf("Stacks:    %s\n", strings.Join(e.Stacks, ", "))
	}
	if len(e.UseCases) > 0 {
		fmt.Printf("Use cases: %s\n", strings.Join(e.UseCases, ", "))
	}
	if e.Weight != "" {
		fmt.Printf("Weight:    %s\n", e.Weight)
	}
	if len(e.Contains) > 0 {
		fmt.Printf("Contains:\n")
		for _, item := range e.Contains {
			fmt.Printf("  - %d %s(s)\n", item.Count, item.Kind)
		}
	}
	if len(e.Requires.Runtime) > 0 {
		fmt.Printf("Requires:  %s\n", strings.Join(e.Requires.Runtime, ", "))
	}
	if len(e.Lotus.ConflictsWith) > 0 {
		fmt.Printf("Conflicts: %s\n", strings.Join(e.Lotus.ConflictsWith, ", "))
	}
	if len(e.Lotus.PairsWellWith) > 0 {
		fmt.Printf("Pairs with: %s\n", strings.Join(e.Lotus.PairsWellWith, ", "))
	}
	if e.Lotus.Notes != "" {
		fmt.Printf("Notes:     %s\n", e.Lotus.Notes)
	}

	if len(e.Benchmarks) > 0 {
		fmt.Printf("\nBenchmarks:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  SCENARIO\tTOKENS IN\tTOKENS OUT\tTIME\tQUALITY\n")
		for _, b := range e.Benchmarks {
			fmt.Fprintf(w, "  %s\t%dk\t%dk\t%ds\t%.1f/5\n",
				b.Scenario, b.TokensIn/1000, b.TokensOut/1000, b.WallTime, b.Quality)
		}
		w.Flush()
	}

	return nil
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
