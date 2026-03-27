package analyzer

import (
	"encoding/json"
	"os"
)

type settingsJSON struct {
	Hooks      map[string][]hookEntry     `json:"hooks"`
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

type hookEntry struct {
	Command string `json:"command"`
}

func parseSettings(path string) (hooks []string, mcpServers []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	var settings settingsJSON
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, nil
	}

	for event, entries := range settings.Hooks {
		for range entries {
			hooks = append(hooks, event)
		}
	}

	for name := range settings.MCPServers {
		mcpServers = append(mcpServers, name)
	}

	return hooks, mcpServers
}
