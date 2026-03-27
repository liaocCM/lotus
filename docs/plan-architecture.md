# Lotus — Minimal AI Config Recommender

## Context

The AI coding tool ecosystem is noisy — 13k+ skills on ClawHub, dozens of MCP registries, new frameworks weekly. All reshuffling the same primitives (prompts, skills, MCP, hooks, agents). No tool answers: "given YOUR project, what's the minimum that works?" and "which option costs the fewest tokens for the same result?"

Lotus fills this gap: **analyze project → match to curated catalog → recommend minimal config → benchmark competing options**.

## Architecture Decisions

- **Go CLI** — single binary, fast startup, no runtime deps, matches your stack
- **CLI + Claude Code skill** — skill is a thin SKILL.md wrapper that calls `lotus` via Bash
- **YAML catalog in git** — human-readable, diffable, PR-contributable, embedded via `go:embed`
- **Semi-automated evaluation** — token/time measured automatically, quality scored by human rubric (1-5)

## Data Model

### Catalog Entry (`catalog/<kind>/<id>.yaml`)
```yaml
id: string
kind: skill | agent | mcp-server | hook | claude-md-snippet
name: string
source: { registry, url, version }
use_cases: [backend-development, testing, ...]
stacks: [go, node, python, rust]
requires: { tools, mcp_servers, runtime }
benchmarks:
  tasks:
    - task_id, tokens_in, tokens_out, wall_time_seconds
    - quality_score (1-5), quality_dimensions: {correctness, completeness, style, tests, efficiency}
lotus: { tier: recommended|alternative|avoid, notes, conflicts_with, pairs_well_with }
```

### Benchmark Task (`benchmarks/tasks/<id>.yaml`)
```yaml
id, name, category, difficulty, stack
description, setup: { repo, branch }
acceptance_criteria: []
quality_rubric: { correctness, completeness, style, tests, efficiency }
```

## Project Structure
```
lotus/
├── cmd/lotus/main.go              # cobra CLI
├── internal/
│   ├── analyzer/                # stack detection + existing config parsing
│   ├── catalog/                 # load/query catalog entries
│   ├── recommender/             # match analysis → catalog → ranked recommendations
│   ├── evaluator/               # benchmark runner + metrics + reports
│   ├── generator/               # output .claude/ configs, settings.json, CLAUDE.md
│   └── registry/                # ClawHub/Smithery API clients
├── catalog/                     # embedded YAML catalog (~30 entries MVP)
│   ├── skills/ agents/ mcp-servers/ hooks/
├── benchmarks/tasks/            # evaluation task definitions
├── skill/SKILL.md               # Claude Code skill wrapper
```

## MVP Commands
```bash
lotus analyze [path]       # detect stack, scan existing .claude/ config
lotus recommend [path]     # analyze + match catalog + output recommendations
lotus apply [path]         # generate/modify .claude/ files (with --dry-run)
lotus catalog list         # browse catalog
lotus catalog show <id>    # entry details + benchmark data
```

## MVP Scope
- Stack detection: Go, TypeScript, Python, Rust
- Parse existing .claude/ configs (skills, agents, hooks, MCP, CLAUDE.md)
- 20-30 hand-curated catalog entries
- Scoring: stack_match * quality / token_cost - existing_overlap
- Config generation: SKILL.md, agent .md, settings.json merge, CLAUDE.md augment

## Roadmap

### Phase 1: Foundation + MVP
- cobra CLI scaffold
- Stack detectors (go.mod, package.json, Cargo.toml, pyproject.toml, CI)
- Existing config parser (.claude/ dir, settings.json)
- Catalog schema + go:embed loading + in-memory index
- Recommender (intersection of use_case + stack indexes, scored)
- Config generator (settings.json deep merge, backup originals)
- SKILL.md wrapper
- Curate initial 20-30 catalog entries

### Phase 2: Evaluation Framework
- 10 benchmark tasks across stacks
- Evaluator runner (isolated worktree, headless Claude Code, capture tokens/time)
- Human scoring CLI: `lotus eval score <run-id>`
- `lotus eval run <id> --task <task-id>` / `lotus eval compare <id1> <id2>`

### Phase 3: Registry Integration
- ClawHub + Smithery API clients
- `lotus search <query>` across registries
- `lotus catalog add <url>` import to local catalog
- File-based cache with TTL

### Phase 4: Community + Polish
- `lotus catalog contribute` (PR-ready YAML gen)
- Remote catalog sync from central repo
- Multi-editor output (Cursor, Codex)
- `lotus diff` / `lotus upgrade`
- Homebrew formula

## Resolved Questions

### 1. Catalog hosting → bundled in lotus repo (MVP), split later if needed
Separate repo adds friction for contributors and complicates the build. Start with `catalog/` embedded via `go:embed`. If the catalog grows past ~100 entries or needs independent release cadence, extract to `lotus-catalog` then.

### 2. Benchmark reproducibility → pin model + context in metadata
Each benchmark run records: model ID, context window, date, lotus version. Temperature is always default (not configurable in Claude Code anyway). Re-run benchmarks when model versions change. Compare across models is a feature, not a bug.

### 3. Quality scoring → you score the initial 20-30, open to community after
Bootstrap the catalog yourself with hands-on evaluation. After launch, accept community scores via `lotus catalog contribute` with a structured rubric. Aggregate scores (median of 3+ reviewers) replace single-reviewer scores.

### 4. Monetization → pure OSS, monetize later via hosted eval service if demand exists
Don't overthink this now. The catalog and CLI are MIT-licensed. If there's demand, a hosted "lotus eval" service (run benchmarks in cloud, get reports) is a natural SaaS extension. But MVP is 100% OSS.

### 5. Scope → configs AND claude-md snippets, but clearly separated
Lotus recommends structural configs (skills/MCP/hooks/agents) as primary output. CLAUDE.md content snippets are a separate `kind: claude-md-snippet` in the catalog — opt-in, not forced. This avoids overwriting opinionated project rules while still offering useful defaults.

## Data Model Update: Bundle Kind

Informed by real-world projects (superpowers, gstack, A-Team), the catalog needs a `bundle` kind:

```yaml
kind: bundle
contains:
  - kind: skill
    count: 14
  - kind: agent
    count: 1
  - kind: hook
    count: 1
weight: light | medium | heavy    # dependency footprint
requires:
  runtime: []                      # e.g., ["bun>=1.0", "playwright"]
```

Bundles compete with each other. Lotus should surface comparisons:
- superpowers (light, 0 deps) vs gstack (heavy, Bun+Playwright) for "structured dev workflow"
- A-Team is meta-level (generates teams) — different use case, no direct conflict

## Data Model Update: Source Kind

Large prompt libraries (agency-agents: 144 personas) are too big to be a single bundle.
Lotus treats them as a `source` — a registry it can cherry-pick from:

```yaml
kind: source
repo: msitarzewski/agency-agents
entry_count: 144
categories: [engineering, design, marketing, sales, qa, game-dev, spatial, academic]
install_method: scripts/install.sh   # or conversion scripts per platform
```

Lotus indexes individual entries from sources and recommends specific ones:
"for your Go backend, grab `engineering/backend-engineer.md` from agency-agents"

## Catalog Kind Summary

| Kind | Example | When to use |
|------|---------|-------------|
| `skill` | minimax-frontend-dev | Single SKILL.md |
| `agent` | backend-engineer persona | Single agent .md |
| `mcp-server` | postgres MCP | Single MCP server config |
| `hook` | block-force-push | Single hook entry |
| `claude-md-snippet` | error-handling rules | CLAUDE.md content block |
| `bundle` | superpowers, gstack | Multi-file package (skills+agents+hooks) |
| `source` | agency-agents | Large library lotus cherry-picks from |

## Initial Catalog Entries (seed list)

### Bundles
| ID | Source | Use Case | Weight |
|----|--------|----------|--------|
| superpowers | obra/superpowers | structured dev workflow (brainstorm→plan→TDD→review) | light (0 deps) |
| gstack | garrytan/gstack | virtual eng team (plan→build→QA→ship→retro) + browser QA | heavy (Bun+Playwright) |
| a-team | chemistrywow31/A-Team | meta: generates custom agent teams via interview | light (0 deps) |

### Sources
| ID | Source | Scale | Focus |
|----|--------|-------|-------|
| agency-agents | msitarzewski/agency-agents | 144 personas | 12+ divisions: eng, design, marketing, sales, QA, game-dev, spatial, academic |

### Individual Skills
| ID | Source | Use Case | Stack |
|----|--------|----------|-------|
| minimax-frontend-dev | MiniMax-AI/skills | frontend development | React/Next.js |
| minimax-fullstack-dev | MiniMax-AI/skills | fullstack development | React + APIs |
| minimax-android-dev | MiniMax-AI/skills | Android native | Kotlin/Compose |
| minimax-ios-dev | MiniMax-AI/skills | iOS native | SwiftUI/UIKit |
| minimax-flutter-dev | MiniMax-AI/skills | cross-platform mobile | Flutter |
| minimax-rn-dev | MiniMax-AI/skills | cross-platform mobile | React Native/Expo |
| minimax-shader-dev | MiniMax-AI/skills | creative/graphics | GLSL |
| minimax-pdf | MiniMax-AI/skills | document generation | PDF |
| minimax-pptx | MiniMax-AI/skills | document generation | PowerPoint |
| minimax-xlsx | MiniMax-AI/skills | document generation | Excel (C#/OpenXML) |
| minimax-docx | MiniMax-AI/skills | document generation | Word (C#/OpenXML) |
| git-commit | luxray | git commit workflow | stack-agnostic |
| backend-dev-go | TBD | backend development | Go |
| backend-dev-node | TBD | backend development | Node/TypeScript |
| code-review | TBD | code review | stack-agnostic |
| security-audit | TBD | security | stack-agnostic |

### MCP Servers (to curate)
- database (postgres, mongodb, sqlite)
- git/github/gitlab
- filesystem, docker, browser (playwright)

### Hooks (to curate)
- pre-commit lint, notification on stop
- safety guards (block rm -rf, block force push)
