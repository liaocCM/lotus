package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/texliao/lotus/internal/recommender"
)

func Apply(root string, recs *recommender.Recommendations, dryRun bool) error {
	if len(recs.Items) == 0 {
		fmt.Println("Nothing to apply.")
		return nil
	}

	if dryRun {
		fmt.Println("Dry run — no changes will be made.\n")
	}

	claudeDir := filepath.Join(root, ".claude")

	for _, rec := range recs.Items {
		if rec.Action != "add" {
			continue
		}

		switch rec.Entry.Kind {
		case "skill":
			if err := applySkill(claudeDir, rec, dryRun); err != nil {
				return err
			}
		case "bundle":
			if err := applyBundle(claudeDir, rec, dryRun); err != nil {
				return err
			}
		case "mcp-server":
			if err := applyMCPServer(claudeDir, rec, dryRun); err != nil {
				return err
			}
		case "hook":
			if err := applyHook(claudeDir, rec, dryRun); err != nil {
				return err
			}
		case "source":
			printSourceInfo(rec)
		case "agent":
			if err := applyAgent(claudeDir, rec, dryRun); err != nil {
				return err
			}
		}
	}

	if !dryRun {
		fmt.Printf("\nApplied %d recommendations to %s\n", len(recs.Items), claudeDir)
	}

	return nil
}

func applySkill(claudeDir string, rec recommender.Recommendation, dryRun bool) error {
	skillDir := filepath.Join(claudeDir, "skills", rec.Entry.ID)
	readmePath := filepath.Join(skillDir, "README.md")

	if dryRun {
		fmt.Printf("  [skill] Would create %s\n", skillDir)
		fmt.Printf("          Install from: %s\n", rec.Entry.Source.URL)
		return nil
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# %s

Recommended by Lotus on %s.

Install from: %s

## Setup

`+"```bash"+`
git clone %s
# Follow the repo's installation instructions
`+"```"+`
`, rec.Entry.Name, time.Now().Format("2006-01-02"), rec.Entry.Source.URL, rec.Entry.Source.URL)

	return os.WriteFile(readmePath, []byte(content), 0644)
}

func applyBundle(claudeDir string, rec recommender.Recommendation, dryRun bool) error {
	if dryRun {
		fmt.Printf("  [bundle] Would install %s (%s)\n", rec.Entry.ID, rec.Entry.Weight)
		fmt.Printf("           Source: %s\n", rec.Entry.Source.URL)
		if len(rec.Entry.Requires.Runtime) > 0 {
			fmt.Printf("           Requires: %s\n", rec.Entry.Requires.Runtime)
		}
		return nil
	}

	// bundles are installed via git clone; create a pointer file
	bundleDir := filepath.Join(claudeDir, "bundles")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return err
	}

	pointer := fmt.Sprintf(`# %s

Installed by Lotus on %s.

Source: %s
Weight: %s

## Installation

`+"```bash"+`
git clone %s
# Follow the repo's setup instructions
`+"```"+`
`, rec.Entry.Name, time.Now().Format("2006-01-02"), rec.Entry.Source.URL, rec.Entry.Weight, rec.Entry.Source.URL)

	return os.WriteFile(filepath.Join(bundleDir, rec.Entry.ID+".md"), []byte(pointer), 0644)
}

func applyMCPServer(claudeDir string, rec recommender.Recommendation, dryRun bool) error {
	settingsPath := filepath.Join(claudeDir, "settings.json")

	if dryRun {
		fmt.Printf("  [mcp-server] Would add %s to %s\n", rec.Entry.ID, settingsPath)
		return nil
	}

	settings := loadOrCreateSettings(settingsPath)

	if settings["mcpServers"] == nil {
		settings["mcpServers"] = make(map[string]any)
	}
	servers := settings["mcpServers"].(map[string]any)
	servers[rec.Entry.ID] = map[string]any{
		"_lotus_note": fmt.Sprintf("Added by Lotus. Install from: %s", rec.Entry.Source.URL),
	}

	return writeSettings(settingsPath, settings)
}

func applyHook(claudeDir string, rec recommender.Recommendation, dryRun bool) error {
	if dryRun {
		fmt.Printf("  [hook] Would add hook %s\n", rec.Entry.ID)
		return nil
	}
	// hooks require manual setup — just print instructions
	fmt.Printf("  [hook] %s — manual setup required\n", rec.Entry.ID)
	fmt.Printf("         See: %s\n", rec.Entry.Source.URL)
	return nil
}

func applyAgent(claudeDir string, rec recommender.Recommendation, dryRun bool) error {
	if dryRun {
		fmt.Printf("  [agent] Would add agent %s\n", rec.Entry.ID)
		return nil
	}
	agentDir := filepath.Join(claudeDir, "agents")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return err
	}
	content := fmt.Sprintf(`# %s

Recommended by Lotus on %s.
Source: %s
`, rec.Entry.Name, time.Now().Format("2006-01-02"), rec.Entry.Source.URL)
	return os.WriteFile(filepath.Join(agentDir, rec.Entry.ID+".md"), []byte(content), 0644)
}

func printSourceInfo(rec recommender.Recommendation) {
	fmt.Printf("  [source] %s — browse and cherry-pick from:\n", rec.Entry.Name)
	fmt.Printf("           %s\n", rec.Entry.Source.URL)
}

func loadOrCreateSettings(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]any)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return make(map[string]any)
	}
	return settings
}

func writeSettings(path string, settings map[string]any) error {
	// backup
	if _, err := os.Stat(path); err == nil {
		backup := path + ".lotus-backup"
		data, _ := os.ReadFile(path)
		os.WriteFile(backup, data, 0644)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
