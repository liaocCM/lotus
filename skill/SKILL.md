---
name: lotus
description: Analyze project and recommend optimal AI coding assistant configuration. Use when setting up a new project, auditing existing config, or comparing skills/agents/MCP servers.
---

# Lotus — Minimal AI Config Recommender

Run lotus commands to analyze and configure this project.

## Commands

- `lotus analyze .` — Detect stack and current AI config
- `lotus recommend .` — Get recommendations for optimal config
- `lotus apply . --dry-run` — Preview changes
- `lotus apply .` — Apply recommended config
- `lotus catalog list` — Browse all catalog entries
- `lotus catalog list --kind bundle` — Filter by kind
- `lotus catalog list --stack go` — Filter by stack
- `lotus catalog show <id>` — Show entry details

## Usage

When the user asks to optimize their Claude Code setup, analyze their project,
or compare skills/agents, run the appropriate lotus command via Bash.

Always run `lotus analyze .` first to understand the project before recommending.
