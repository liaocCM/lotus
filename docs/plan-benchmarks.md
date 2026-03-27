# Plan: Benchmark Scenarios for Lotus

## Context

Current recommender scores entries via use-case/stack matching with hardcoded tier multipliers. No real evaluation data backs the recommendations. We need mock scenarios that simulate real projects, run competing solutions against them, and record structured results so the recommender can use actual benchmark data instead of vibes.

## Scenarios

5 scenarios covering the main competition groups in the catalog:

| # | Scenario | Stack | Competing Solutions | What We Measure |
|---|----------|-------|-------------------|-----------------|
| S1 | Go backend API (CRUD + tests) | go + gin + postgres | superpowers vs gstack vs D-Team | tokens, time, code quality, test coverage |
| S2 | React/Next.js frontend feature | typescript + next | minimax-frontend-dev vs gstack vs superpowers | tokens, time, component quality, accessibility |
| S3 | New project bootstrap (greenfield) | go (empty) | A-Team (generate team) vs D-Team (pre-built) vs bare Claude Code | tokens, time, team quality, completeness |
| S4 | Multi-team pipeline orchestration | any | O-Team vs manual sequential | tokens, time, handoff quality, pipeline completeness |
| S5 | Mobile app feature (Flutter) | dart + flutter | minimax-flutter-dev vs bare Claude Code | tokens, time, code quality |

## What Changes

### 1. Add `Benchmark` field to catalog Entry (entry.go)
```go
type Benchmark struct {
    Scenario string  `yaml:"scenario"`
    TokensIn int     `yaml:"tokens_in"`
    TokensOut int    `yaml:"tokens_out"`
    WallTime int     `yaml:"wall_time_seconds"`
    Quality  float64 `yaml:"quality_score"` // 1-5
}

// Entry gains:
Benchmarks []Benchmark `yaml:"benchmarks,omitempty"`
```

### 2. Add scenario definitions (catalogdata/data/scenarios/)
Each scenario is a YAML file describing:
- ID, name, description
- Stack requirements
- Task prompt (what to ask the AI)
- Acceptance criteria (checklist)
- Quality rubric (what 1-5 means for this task)

### 3. Add benchmark data to catalog entries
After manually running each scenario with each competing solution, record results directly in the catalog YAML:
```yaml
benchmarks:
  - scenario: s1-go-crud-api
    tokens_in: 12400
    tokens_out: 8200
    wall_time_seconds: 45
    quality_score: 4.2
```

### 4. Update recommender scoring (recommender.go)
Current: `score = use_case_match * tier_bonus * weight_penalty`
New: if benchmark data exists for a matching scenario, factor it in:
```
score = base_score * benchmark_factor
benchmark_factor = quality_score / normalized_token_cost
```
Entries with benchmarks get more accurate scores; entries without fall back to current heuristic.

### 5. Add `lotus benchmark` CLI commands
- `lotus benchmark list` — list scenarios
- `lotus benchmark show <id>` — show scenario details + results across solutions
- `lotus benchmark compare <id1> <id2> --scenario <s>` — side-by-side comparison

### 6. Also: add D-Team + O-Team to catalog
New YAML entries needed for the 2 repos the user just evaluated.

## Files to Modify/Create

| File | Action |
|------|--------|
| `internal/catalog/entry.go` | Add Benchmark struct + field to Entry |
| `internal/recommender/recommender.go` | Factor benchmark data into scoring |
| `cmd/lotus/main.go` | Add benchmark subcommands |
| `internal/benchmark/benchmark.go` | NEW — scenario loading, comparison display |
| `catalogdata/data/scenarios/s1-go-crud-api.yaml` | NEW — scenario definition |
| `catalogdata/data/scenarios/s2-react-feature.yaml` | NEW |
| `catalogdata/data/scenarios/s3-greenfield-bootstrap.yaml` | NEW |
| `catalogdata/data/scenarios/s4-multi-team-pipeline.yaml` | NEW |
| `catalogdata/data/scenarios/s5-flutter-feature.yaml` | NEW |
| `catalogdata/data/bundles/d-team.yaml` | NEW — catalog entry |
| `catalogdata/data/bundles/o-team.yaml` | NEW — catalog entry (kind: skill, not bundle) |
| existing catalog YAMLs | Add mock benchmark data to entries that compete in scenarios |

## Mock Benchmark Data

We won't run real evaluations yet — we'll populate realistic mock data based on what we know about each solution's characteristics:

**S1: Go CRUD API**
| Solution | tokens_in | tokens_out | wall_time | quality |
|----------|-----------|------------|-----------|---------|
| superpowers | 8k | 5k | 35s | 4.0 | light, TDD-first, less overhead |
| gstack | 15k | 10k | 60s | 4.2 | heavier setup, but QA/review baked in |
| d-team | 12k | 8k | 50s | 4.5 | full team, best quality but more tokens |

**S2: React Feature**
| Solution | tokens_in | tokens_out | wall_time | quality |
|----------|-----------|------------|-----------|---------|
| minimax-frontend-dev | 6k | 4k | 25s | 4.0 | focused, lightweight |
| gstack | 14k | 9k | 55s | 3.8 | overkill for single feature |
| superpowers | 9k | 6k | 35s | 3.5 | not frontend-specific |

**S3: Greenfield Bootstrap**
| Solution | tokens_in | tokens_out | wall_time | quality |
|----------|-----------|------------|-----------|---------|
| a-team | 20k | 15k | 90s | 4.5 | custom team, best fit |
| d-team | 5k | 3k | 20s | 3.8 | instant but generic |
| bare | 3k | 2k | 15s | 2.5 | no structure |

**S4: Multi-Team Pipeline**
| Solution | tokens_in | tokens_out | wall_time | quality |
|----------|-----------|------------|-----------|---------|
| o-team | 10k | 7k | 45s | 4.3 | purpose-built for this |
| manual | 25k | 18k | 120s | 3.0 | tedious, error-prone handoffs |

**S5: Flutter Feature**
| Solution | tokens_in | tokens_out | wall_time | quality |
|----------|-----------|------------|-----------|---------|
| minimax-flutter-dev | 7k | 5k | 30s | 4.2 | flutter-specific |
| bare | 8k | 6k | 35s | 3.0 | no flutter knowledge |

## Verification
1. `go build ./cmd/lotus/` compiles
2. `lotus benchmark list` shows 5 scenarios
3. `lotus benchmark show s1-go-crud-api` shows scenario + results table
4. `lotus benchmark compare superpowers gstack --scenario s1-go-crud-api` shows side-by-side
5. `lotus recommend .` scores now factor in benchmark data (Go project should rank superpowers higher due to better token efficiency)
6. `lotus catalog show d-team` and `lotus catalog show o-team` work
