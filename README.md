# Lotus

<p align="center">
  <img src="https://pub-50dac11dd9ed4bc88adfff4ce0fcef3a.r2.dev/imgs/cards/purity.png" alt="Lotus" width="400" />
  <br/>
  <sub>Art: <b>Purity</b> card from <a href="https://store.steampowered.com/app/2868840/Slay_the_Spire_2/">Slay the Spire 2</a> — "Remove a card from your deck."</sub>
</p>

<p align="center">
  <a href="README.zh-TW.md">繁體中文</a>
</p>

Minimal AI config recommender for coding assistants.

The AI coding tool ecosystem is noisy — thousands of skills, dozens of MCP registries, new frameworks weekly. Lotus answers one question: **given your project, what's the minimum that works?**

## What It Does

1. **Analyzes** your project — detects language, framework, database, CI, and existing AI config
2. **Recommends** the minimum effective set of skills, agents, MCP servers, and hooks from a curated catalog
3. **Compares** competing solutions by token cost, time, and quality
4. **Applies** configs directly to your `.claude/` directory

## Install

```bash
go install github.com/texliao/lotus/cmd/lotus@latest
```

## Usage

```bash
# Detect stack and scan existing AI config
lotus analyze .

# Get recommendations
lotus recommend .

# Preview changes without writing
lotus apply . --dry-run

# Apply recommended config
lotus apply .

# Browse catalog
lotus catalog list
lotus catalog list --kind bundle
lotus catalog list --stack go

# Show entry details
lotus catalog show superpowers
```

## Example

```
$ lotus analyze .

Project: /home/user/my-go-api

Detected stacks:
  LANGUAGE  VERSION  FRAMEWORK  DATABASE  CI
  go        1.24     gin        postgres  github-actions

Existing AI config:
  CLAUDE.md:    true
  Skills:       1 (git-commit)
  Agents:       none
  Hooks:        none
  MCP servers:  none

Inferred use cases: backend-development, code-review, testing, git-workflow

$ lotus recommend .

Recommendations for /home/user/my-go-api
Detected: go

  SCORE  ACTION  KIND    ID           REASON
  -----  ------  ----    --           ------
  22     add     bundle  superpowers  matches use case: code-review; stack-agnostic

1 recommendations. Run `lotus apply .` to apply.
```

## Catalog

Lotus ships with a curated catalog of evaluated AI coding tools:

| Kind | Description | Examples |
|------|-------------|---------|
| `skill` | Single SKILL.md file | minimax-frontend-dev, git-commit |
| `bundle` | Multi-file package (skills + agents + hooks) | superpowers, gstack, D-Team |
| `source` | Large library to cherry-pick from | agency-agents (144 personas) |
| `agent` | Single agent definition | - |
| `mcp-server` | MCP server config | - |
| `hook` | Shell hook for Claude Code | - |

### Included Entries

**Bundles**
- [superpowers](https://github.com/obra/superpowers) — structured dev workflow (brainstorm, plan, TDD, review). Zero deps.
- [gstack](https://github.com/garrytan/gstack) — virtual eng team (plan, build, QA, ship, retro). Requires Bun + Playwright.
- [A-Team](https://github.com/chemistrywow31/A-Team) — meta-agent that generates custom agent teams via interview.

**Skills**
- [MiniMax-AI/skills](https://github.com/MiniMax-AI/skills) — frontend, fullstack, Android, iOS, Flutter, React Native, shader, PDF/PPTX/XLSX/DOCX generation.
- git-commit — conventional commit workflow.

**Sources**
- [agency-agents](https://github.com/msitarzewski/agency-agents) — 144 agent personas across engineering, design, marketing, sales, QA, game dev, and more.

## How Recommendations Work

1. **Stack detection** — scan `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, CI configs, Docker Compose
2. **Use case inference** — map detected stack to use cases (backend-development, frontend-development, etc.)
3. **Catalog matching** — find entries matching your use cases and stack
4. **Scoring** — `base_score = use_case_match + stack_match + tier_bonus - weight_penalty`
5. **Conflict resolution** — if two entries conflict (e.g., superpowers vs gstack), keep the higher-scored one

## Adding Entries

Add a YAML file to `catalogdata/data/<kind>/`:

```yaml
id: my-skill
kind: skill
name: "My Skill"
source:
  registry: github
  repo: "user/repo"
  url: "https://github.com/user/repo"
use_cases:
  - backend-development
stacks:
  - go
requires:
  tools: []
  mcp_servers: []
  runtime: []
lotus:
  tier: recommended
  notes: "Description of what this does and why."
  conflicts_with: []
  pairs_well_with: []
```

## License

MIT
