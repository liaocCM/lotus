# Plan: Two-Axis Evaluation System

## Context

Current scoring blends everything into one number. Evaluation should separate objective (measurable) from subjective (taste-based), with built-in defaults that users can override.

## Two Axes

### Axis 1: Objective (auto-measurable)
- Token cost (in/out)
- Wall time
- Test pass rate
- Build success
- Code coverage %
- Lint/type error count

Source: Docker benchmarks (Phase: plan-docker-benchmarks.md)
Same for all users. Stored in catalog YAML.

### Axis 2: Subjective (human judgment)
- Code style / readability (1-5)
- Doc quality (1-5)
- UI/UX aesthetics (1-5)
- Architecture taste (1-5)
- Overall feel (1-5)

Source: built-in defaults (curated by us) + user overrides.

## Data Model

### Catalog entry (built-in, shipped with lotus)
```yaml
benchmarks:
  - scenario: s1-go-crud-api
    # objective (auto-measured)
    tokens_in: 8000
    tokens_out: 5000
    wall_time_seconds: 35
    test_pass_rate: 1.0
    build_success: true
    # subjective (our defaults)
    code_style: 4.0
    doc_quality: 3.5
    architecture: 4.0
    overall_feel: 4.0
```

### User preferences (~/.lotus/preferences.yaml, local only)
```yaml
# overrides built-in subjective scores
overrides:
  superpowers:
    code_style: 3.0      # user thinks it's too verbose
    overall_feel: 3.5
  d-team:
    architecture: 5.0     # user loves the team structure
# user's weight preferences (which axis matters more)
weights:
  token_efficiency: 0.3   # default: 0.25
  quality: 0.3            # default: 0.25
  code_style: 0.2         # default: 0.25
  speed: 0.2              # default: 0.25
```

## Scoring Formula

```
objective_score = (
    quality_score * weights.quality +
    token_efficiency * weights.token_efficiency +
    speed_score * weights.speed
)

subjective_score = (
    code_style * weights.code_style  // user override or built-in default
)

final_score = base_match_score * (objective_score + subjective_score)
```

## Implementation

### Phase 1: Built-in subjective defaults (next)
- Extend Benchmark struct with subjective fields
- We rate each competing solution ourselves
- Ship as part of catalog YAML
- Recommender uses combined objective + subjective score

### Phase 2: User preferences file (~/.lotus/)
- `lotus preferences init` — create ~/.lotus/preferences.yaml with defaults
- `lotus preferences set <entry-id> <dimension> <score>` — override a score
- `lotus preferences weights` — adjust axis weights
- Recommender loads user prefs and merges with built-in defaults

### Phase 3: Interactive review UI
- `lotus benchmark review <scenario-id>` — show side-by-side outputs
- User picks winner, rates dimensions
- Scores auto-saved to preferences.yaml

### Phase 4: Community aggregation (future)
- `lotus preferences export` — share your ratings
- Aggregate community ratings (median of N reviewers)
- Built-in defaults evolve from our ratings to community consensus

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/catalog/entry.go` | Extend Benchmark struct with subjective fields |
| `internal/preferences/preferences.go` | NEW — load/save ~/.lotus/preferences.yaml |
| `internal/recommender/recommender.go` | Merge built-in + user prefs into scoring |
| `cmd/lotus/main.go` | Add preferences subcommands |
| catalog YAMLs | Add subjective scores to existing benchmark data |

## Key Principle

Fresh install = useful immediately (our curated data).
Over time = personalized (user overrides what they disagree with).
No rating required to use lotus — defaults are good enough.
