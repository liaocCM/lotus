package analyzer

import (
	"os"
	"path/filepath"
	"strings"
)

func Analyze(root string) (*ProjectProfile, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	profile := &ProjectProfile{
		Path:   absRoot,
		Stacks: detectStacks(absRoot),
	}

	profile.ExistingConfig = scanExistingConfig(absRoot)

	return profile, nil
}

func scanExistingConfig(root string) ExistingConfig {
	cfg := ExistingConfig{}

	// check CLAUDE.md
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); err == nil {
		cfg.HasClaudeMD = true
	}

	claudeDir := filepath.Join(root, ".claude")

	// scan skills
	cfg.Skills = listSubdirs(filepath.Join(claudeDir, "skills"))

	// scan agents
	cfg.Agents = listSubdirs(filepath.Join(claudeDir, "agents"))
	// also check for agent .md files directly
	cfg.Agents = append(cfg.Agents, listMDFiles(filepath.Join(claudeDir, "agents"))...)

	// scan rules
	cfg.Rules = listMDFiles(filepath.Join(claudeDir, "rules"))

	// scan settings.json for hooks and MCP servers
	cfg.Hooks, cfg.MCPServers = parseSettings(filepath.Join(claudeDir, "settings.json"))
	localHooks, localMCP := parseSettings(filepath.Join(claudeDir, "settings.local.json"))
	cfg.Hooks = append(cfg.Hooks, localHooks...)
	cfg.MCPServers = append(cfg.MCPServers, localMCP...)

	return cfg
}

func listSubdirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func listMDFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			name := strings.TrimSuffix(e.Name(), ".md")
			names = append(names, name)
		}
	}
	return names
}
