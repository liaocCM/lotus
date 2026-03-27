# Plan: Real Benchmarks via Docker

## Context

Current benchmark data is mock estimates based on project characteristics. This plan covers running real Claude Code sessions in isolated Docker containers to get actual token usage, wall time, and quality scores for each competing solution.

## Prerequisites

- `~/.claude/.env` contains OAuth key for Claude Code
- Docker installed
- Each scenario needs a baseline test repo

## Architecture

```
lotus benchmark run <scenario-id> <solution-id>
  └── spins up Docker container
       ├── copies test repo for scenario
       ├── installs solution (skill/bundle)
       ├── injects ~/.claude/.env for auth
       ├── runs `claude -p "<scenario prompt>"` headless
       ├── captures: token usage (from claude output), wall time, generated files
       └── outputs structured result JSON
```

## Test Repos (to create)

| Scenario | Repo | Contents |
|----------|------|----------|
| s1-go-crud-api | `benchmarks/repos/go-gin-api/` | Go + Gin + PostgreSQL skeleton, existing models, router |
| s2-react-feature | `benchmarks/repos/nextjs-app/` | Next.js + TypeScript + Tailwind skeleton, existing pages |
| s3-greenfield-bootstrap | `benchmarks/repos/go-empty/` | Empty Go module, just go.mod |
| s4-multi-team-pipeline | `benchmarks/repos/multi-team/` | 3 pre-defined team folders |
| s5-flutter-feature | `benchmarks/repos/flutter-app/` | Flutter skeleton with pubspec.yaml |

## Dockerfile

```dockerfile
FROM node:20-slim
RUN npm install -g @anthropic-ai/claude-code
WORKDIR /workspace
COPY .env /root/.claude/.env
COPY repo/ .
COPY solution/ .claude/
CMD ["claude", "-p", "--output-format", "json"]
```

## Implementation Steps

1. Create `internal/evaluator/` package
   - `docker.go` — build/run Docker containers
   - `runner.go` — orchestrate benchmark runs
   - `metrics.go` — parse Claude Code JSON output for token counts
   - `report.go` — generate result JSON
2. Create test repo skeletons in `benchmarks/repos/`
3. Add `lotus benchmark run <scenario> <solution>` command
   - Builds Docker image with solution installed
   - Runs scenario prompt
   - Captures metrics
   - Updates catalog YAML with real numbers
4. Add `lotus benchmark run-all` — runs all scenario/solution combinations
5. Quality scoring: after each run, `lotus benchmark score <run-id>` opens results for human 1-5 rating

## Token Capture

Claude Code CLI outputs token usage in its JSON output format:
```json
{
  "usage": {
    "input_tokens": 12400,
    "output_tokens": 8200
  }
}
```

Parse this from the container's stdout.

## Cost Estimate

~15 runs (5 scenarios x ~3 competitors each)
~10-20k tokens per run
Total: ~150-300k tokens per full benchmark suite

## Open Questions

1. How to handle solutions that need MCP servers (e.g., Playwright for gstack)?
2. Should we pin model version in the Dockerfile or use whatever's current?
3. Network access in containers — some solutions may need git clone during setup
