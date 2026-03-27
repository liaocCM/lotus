package analyzer

import (
	"fmt"
	"os"
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
	fmt.Printf("Inferred use cases: %s\n", strings.Join(p.InferUseCases(), ", "))
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return fmt.Sprintf("%d (%s)", len(items), strings.Join(items, ", "))
}
