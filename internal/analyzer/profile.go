package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

type Stack struct {
	Language  string `yaml:"language"`
	Version   string `yaml:"version,omitempty"`
	Framework string `yaml:"framework,omitempty"`
	Database  string `yaml:"database,omitempty"`
	Build     string `yaml:"build,omitempty"`
	CI        string `yaml:"ci,omitempty"`
}

type ExistingConfig struct {
	HasClaudeMD bool     `yaml:"claude_md"`
	Skills      []string `yaml:"skills"`
	Agents      []string `yaml:"agents"`
	Hooks       []string `yaml:"hooks"`
	MCPServers  []string `yaml:"mcp_servers"`
	Rules       []string `yaml:"rules"`
}

type ProjectProfile struct {
	Path           string         `yaml:"path"`
	Stacks         []Stack        `yaml:"stacks"`
	ExistingConfig ExistingConfig `yaml:"existing_config"`
	Complexity     Complexity     `yaml:"complexity"`
}

// Complexity represents the estimated project complexity.
// Derived from file count, directory depth, dependency count, etc.
type Complexity struct {
	Level      string `yaml:"level"`       // trivial, small, medium, large
	FileCount  int    `yaml:"file_count"`
	DirCount   int    `yaml:"dir_count"`
	DepCount   int    `yaml:"dep_count"`   // dependencies in go.mod/package.json
	HasTests   bool   `yaml:"has_tests"`
	IsMonorepo bool   `yaml:"is_monorepo"`
}

func (p *ProjectProfile) PrimaryLanguage() string {
	if len(p.Stacks) > 0 {
		return p.Stacks[0].Language
	}
	return ""
}

func (p *ProjectProfile) Languages() []string {
	var langs []string
	for _, s := range p.Stacks {
		langs = append(langs, s.Language)
	}
	return langs
}

func (p *ProjectProfile) InferUseCases() []string {
	var useCases []string
	seen := make(map[string]bool)

	add := func(uc string) {
		if !seen[uc] {
			seen[uc] = true
			useCases = append(useCases, uc)
		}
	}

	for _, s := range p.Stacks {
		switch s.Language {
		case "go", "python", "rust":
			add("backend-development")
		case "typescript", "javascript":
			if s.Framework == "react" || s.Framework == "next" || s.Framework == "vue" || s.Framework == "angular" || s.Framework == "svelte" {
				add("frontend-development")
			}
			if s.Framework == "express" || s.Framework == "nest" || s.Framework == "fastify" || s.Framework == "hono" {
				add("backend-development")
			}
			if s.Framework == "next" || s.Framework == "" {
				add("fullstack-development")
			}
			if s.Framework == "react-native" || s.Framework == "expo" {
				add("mobile-development")
			}
		case "kotlin":
			add("mobile-development")
		case "swift":
			add("mobile-development")
		case "dart":
			add("mobile-development")
		}
		if s.Database != "" {
			add("backend-development")
		}
		if s.CI != "" {
			add("devops")
		}
	}

	// universal use cases
	add("code-review")
	add("testing")
	add("git-workflow")

	return useCases
}

func (p *ProjectProfile) Print() {
	fmt.Printf("Project: %s\n\n", p.Path)

	if len(p.Stacks) > 0 {
		fmt.Println("Detected stacks:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  LANGUAGE\tVERSION\tFRAMEWORK\tDATABASE\tCI\n")
		for _, s := range p.Stacks {
			ver := s.Version
			if ver == "" {
				ver = "-"
			}
			fw := s.Framework
			if fw == "" {
				fw = "-"
			}
			db := s.Database
			if db == "" {
				db = "-"
			}
			ci := s.CI
			if ci == "" {
				ci = "-"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", s.Language, ver, fw, db, ci)
		}
		w.Flush()
	} else {
		fmt.Println("No stacks detected.")
	}

	fmt.Println()
	fmt.Println("Existing AI config:")
	fmt.Printf("  CLAUDE.md:    %v\n", p.ExistingConfig.HasClaudeMD)
	fmt.Printf("  Skills:       %s\n", formatList(p.ExistingConfig.Skills))
	fmt.Printf("  Agents:       %s\n", formatList(p.ExistingConfig.Agents))
	fmt.Printf("  Hooks:        %s\n", formatList(p.ExistingConfig.Hooks))
	fmt.Printf("  MCP servers:  %s\n", formatList(p.ExistingConfig.MCPServers))
	fmt.Printf("  Rules:        %s\n", formatList(p.ExistingConfig.Rules))

	fmt.Println()
	fmt.Printf("Complexity:     %s (%d files, %d dirs, %d deps)\n",
		p.Complexity.Level, p.Complexity.FileCount, p.Complexity.DirCount, p.Complexity.DepCount)
	fmt.Printf("Inferred use cases: %s\n", strings.Join(p.InferUseCases(), ", "))
}

// EstimateComplexity scans the project to determine its complexity level.
func EstimateComplexity(root string) Complexity {
	c := Complexity{}

	// count source files and directories
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// skip hidden dirs, vendor, node_modules
		name := info.Name()
		if info.IsDir() && (strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			c.DirCount++
			return nil
		}
		// count source files by extension
		ext := filepath.Ext(name)
		switch ext {
		case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".dart", ".kt", ".swift":
			c.FileCount++
		}
		return nil
	})

	// count dependencies
	c.DepCount = countDeps(root)

	// check for tests
	c.HasTests = hasTestFiles(root)

	// check for monorepo indicators
	c.IsMonorepo = isMonorepo(root)

	// classify
	switch {
	case c.FileCount <= 5 && c.DepCount <= 3:
		c.Level = "trivial"
	case c.FileCount <= 20 && c.DepCount <= 10:
		c.Level = "small"
	case c.FileCount <= 100 && c.DepCount <= 50:
		c.Level = "medium"
	default:
		c.Level = "large"
	}

	// monorepos bump up one level
	if c.IsMonorepo && c.Level != "large" {
		switch c.Level {
		case "trivial":
			c.Level = "small"
		case "small":
			c.Level = "medium"
		case "medium":
			c.Level = "large"
		}
	}

	return c
}

func countDeps(root string) int {
	count := 0
	// go.mod
	if data, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "require") || strings.Contains(line, "//") || line == "" || line == ")" {
				continue
			}
			if strings.Contains(line, "/") {
				count++
			}
		}
	}
	// package.json
	if data, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
		// rough count: each line with ":" in dependencies sections
		content := string(data)
		for _, section := range []string{"dependencies", "devDependencies"} {
			idx := strings.Index(content, `"`+section+`"`)
			if idx < 0 {
				continue
			}
			block := content[idx:]
			end := strings.Index(block, "}")
			if end > 0 {
				lines := strings.Split(block[:end], "\n")
				for _, l := range lines {
					if strings.Contains(l, ":") && strings.Contains(l, `"`) {
						count++
					}
				}
			}
		}
	}
	return count
}

func hasTestFiles(root string) bool {
	found := false
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.ts") ||
			strings.HasSuffix(name, ".test.tsx") || strings.HasSuffix(name, ".spec.ts") ||
			strings.Contains(name, "test_") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func isMonorepo(root string) bool {
	// check for multiple go.mod, lerna.json, pnpm-workspace.yaml, etc.
	indicators := []string{"lerna.json", "pnpm-workspace.yaml", "nx.json", "turbo.json"}
	for _, f := range indicators {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	// multiple go.mod files
	goModCount := 0
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "go.mod" {
			goModCount++
			if goModCount > 1 {
				return filepath.SkipAll
			}
		}
		return nil
	})
	return goModCount > 1
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return fmt.Sprintf("%d (%s)", len(items), strings.Join(items, ", "))
}
