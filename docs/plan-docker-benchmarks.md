# Plan: Real Benchmarks via Docker

## Context

Current benchmark data is mock estimates. This plan covers running real Claude Code sessions in isolated Docker containers to get actual token usage, wall time, and quality scores.

## 5 Configuration Tiers

Every scenario is tested across these 5 tiers to measure the value-add of each layer:

| Tier | Config | What it tests |
|------|--------|---------------|
| A | **Bare** — default Claude Code, no config | Baseline |
| B | **Native subagents** — bare + agent teams enabled | Does Claude's built-in multi-agent help? |
| C | **Single skill** — one focused skill for the task | Is minimal config enough? |
| D | **Light bundle** — superpowers (14 skills, 0 deps) | Does structured workflow add value? |
| E | **Heavy bundle** — gstack or d-team (20+ skills, external deps) | Does full team setup justify the cost? |

## Task Complexity Dimension

Scenarios are tagged by complexity. Expected pattern:

| Complexity | Token budget | Expected best tier |
|---|---|---|
| **trivial** | <5k tokens | A or C — overhead not worth it |
| **small** | 5-15k tokens | C or D — one skill or light workflow |
| **medium** | 15-50k tokens | D — structured workflow pays off |
| **large** | 50k+ tokens | E — full team justified |

This lets lotus recommend lighter configs for small tasks and heavier ones for large tasks.

## Pilot Benchmark: Go Slugify (trivial)

First end-to-end test of the pipeline.

**Task:** Write `stringutil.Slugify(s string) string` + 8 table-driven tests.

**Test repo** (`benchmarks/repos/go-slugify/`):
```
go.mod                  (module example.com/testbed; go 1.22)
stringutil/.gitkeep     (empty package)
```

**Prompt:**
> Write a `stringutil.Slugify(s string) string` function that converts a string to a URL-safe slug (lowercase, hyphens for spaces, strip non-alphanumeric). Include table-driven tests with at least 8 cases covering unicode, leading/trailing spaces, consecutive special chars, and empty string.

**5 runs:**

| Run | Tier | Config details |
|-----|------|---------------|
| A | Bare | `claude -p "<prompt>"` |
| B | Native subagents | `claude -p "<prompt>"` with CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1 |
| C | Single skill | Go backend dev skill in .claude/skills/ |
| D | Light bundle | superpowers installed |
| E | Heavy bundle | d-team installed |

**Verification script:**
```bash
go vet ./...                           # compiles?
go test ./stringutil/... -v -count=1   # tests pass?
go test ./stringutil/... -cover        # coverage %
grep -c "func Test" stringutil/*_test.go  # test case count
# spot checks:
# Slugify("Hello World!") == "hello-world"
# Slugify("") == ""
# Slugify("  Café  ") == "cafe"
```

**Metrics captured per run:**
- tokens_in, tokens_out (from Claude Code JSON output)
- wall_time_seconds
- build_success (bool)
- test_pass (bool)
- coverage_pct (float)
- test_case_count (int)
- first_try_compile (bool) — did it compile without Claude retrying?

## Prerequisites

- `~/.claude/.env` contains OAuth key for Claude Code
- Docker installed
- Go test repos in `benchmarks/repos/`

## Architecture

```
lotus benchmark run <scenario-id> --tier <A|B|C|D|E>
  └── spins up Docker container
       ├── base image: golang:1.22 + node:20 (for claude CLI)
       ├── copies test repo for scenario
       ├── installs config for the tier (nothing / skill / bundle)
       ├── injects ~/.claude/.env for auth
       ├── runs claude -p "<prompt>" --output-format json
       ├── runs verification script
       ├── captures all metrics
       └── outputs structured result JSON

lotus benchmark run-all <scenario-id>
  └── runs all 5 tiers sequentially, outputs comparison table
```

## Dockerfile

```dockerfile
FROM golang:1.22

# Install Node.js + Claude Code
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @anthropic-ai/claude-code

WORKDIR /workspace
COPY repo/ .
COPY config/ .claude/
COPY .env /root/.claude/.env
COPY verify.sh .

# Run benchmark
CMD ["bash", "-c", "claude -p \"$(cat prompt.txt)\" --output-format json > result.json 2>&1 && bash verify.sh"]
```

## Result JSON

```json
{
  "scenario": "s0-go-slugify",
  "tier": "D",
  "config": "superpowers",
  "metrics": {
    "tokens_in": 3200,
    "tokens_out": 1800,
    "wall_time_seconds": 22,
    "build_success": true,
    "test_pass": true,
    "coverage_pct": 95.2,
    "test_case_count": 10,
    "first_try_compile": true
  }
}
```

## Test Repos (full list)

| Scenario | Complexity | Repo | Contents |
|----------|-----------|------|----------|
| s0-go-slugify | trivial | `benchmarks/repos/go-slugify/` | go.mod + empty stringutil/ |
| s1-go-crud-api | medium | `benchmarks/repos/go-gin-api/` | Go + Gin + PostgreSQL skeleton |
| s2-react-feature | medium | `benchmarks/repos/nextjs-app/` | Next.js + TS + Tailwind skeleton |
| s3-greenfield-bootstrap | small | `benchmarks/repos/go-empty/` | Empty Go module |
| s4-multi-team-pipeline | large | `benchmarks/repos/multi-team/` | 3 pre-defined team folders |
| s5-flutter-feature | medium | `benchmarks/repos/flutter-app/` | Flutter skeleton |

## Implementation Steps

1. Create `benchmarks/repos/go-slugify/` test repo (2 files)
2. Create `benchmarks/Dockerfile` and `benchmarks/verify-go-slugify.sh`
3. Create `internal/evaluator/` package (docker.go, runner.go, metrics.go)
4. Add `lotus benchmark run` and `lotus benchmark run-all` CLI commands
5. Run pilot: 5 tiers on go-slugify
6. Analyze results, update catalog YAML with real data
7. Expand to remaining scenarios

## Cost Estimate

Pilot (go-slugify): 5 runs x ~3-5k tokens = ~15-25k tokens
Full suite: 6 scenarios x 5 tiers = 30 runs x ~10k avg = ~300k tokens

## Open Questions

1. How to handle solutions that need MCP servers (e.g., Playwright for gstack)?
2. Pin model version in Dockerfile or use current?
3. Network access — some bundles need git clone during setup
4. Tier B (native subagents) — does the trivial slugify task even trigger subagent spawning?
5. Tier C — which single skill to use per scenario? Need a mapping.
