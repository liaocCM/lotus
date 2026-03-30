package recommender

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/texliao/lotus/internal/analyzer"
	"github.com/texliao/lotus/internal/catalog"
)

type Recommendation struct {
	Entry  *catalog.Entry
	Score  float64
	Reason string
	Action string // "add", "remove", "upgrade"
}

type Recommendations struct {
	Profile *analyzer.ProjectProfile
	Items   []Recommendation
}

// Scoring weights derived from 20 benchmark runs across 4 scenarios:
//
// Key findings:
//   - Trivial tasks: all tiers equal quality. Single skill cheapest ($0.19 vs $0.32 for heavy).
//   - Medium Go: single skill best value ($0.36, 91.1% cov). Heavy adds cost, marginal quality.
//   - Medium React: heavy bundles catch TS errors (build pass). Light tiers fail TS compilation.
//   - Greenfield/setup: bare cheapest ($0.82). Superpowers produces richest config (19 skills, $1.66).
//   - Heavy bundles (d-team) only justify cost on medium+ frontend/complex tasks.
//
// Weight penalty by complexity (from benchmark cost/quality ratios):
//   heavy + trivial: 0.3x (2.5x cost, 0x quality gain)
//   heavy + small:   0.5x
//   heavy + medium:  0.9x (justified for frontend: catches TS errors, 68 vs 38 test cases)
//   heavy + large:   1.1x (bonus — full team structure helps)
//
// Single skills consistently perform well across all complexities.
// Bundles only differentiate on medium+ tasks, especially frontend.

func Recommend(profile *analyzer.ProjectProfile, cat *catalog.Catalog) *Recommendations {
	recs := &Recommendations{Profile: profile}

	useCases := profile.InferUseCases()
	languages := profile.Languages()
	existing := existingSet(profile)
	complexity := profile.Complexity.Level

	type scored struct {
		entry  *catalog.Entry
		score  float64
		reason string
	}
	var candidates []scored

	for i := range cat.Entries {
		e := &cat.Entries[i]

		if existing[e.ID] {
			continue
		}
		if e.Lotus.Tier == "avoid" {
			continue
		}

		score := 0.0
		var reasons []string

		// use case match
		for _, uc := range e.UseCases {
			for _, puc := range useCases {
				if uc == puc {
					score += 10
					reasons = append(reasons, fmt.Sprintf("matches: %s", uc))
				}
			}
		}

		// stack match — stronger signal than use case
		for _, es := range e.Stacks {
			for _, pl := range languages {
				if es == pl {
					score += 15
					reasons = append(reasons, fmt.Sprintf("stack: %s", es))
				}
			}
		}

		// stack-agnostic bonus
		if len(e.Stacks) == 0 && score > 0 {
			score += 5
		}

		if score == 0 {
			continue
		}

		// tier multiplier
		switch e.Lotus.Tier {
		case "recommended":
			score *= 1.5
		case "alternative":
			score *= 1.0
		}

		// complexity-aware weight adjustment (data-driven from benchmarks)
		switch e.Weight {
		case "heavy":
			switch complexity {
			case "trivial":
				score *= 0.3
				reasons = append(reasons, "heavy penalized: trivial project")
			case "small":
				score *= 0.5
				reasons = append(reasons, "heavy penalized: small project")
			case "medium":
				score *= 0.9
			default: // large
				score *= 1.1
				reasons = append(reasons, "heavy justified: large project")
			}
		case "medium":
			switch complexity {
			case "trivial":
				score *= 0.5
			case "small":
				score *= 0.7
			default:
				score *= 0.95
			}
		}

		// bundle complexity bonus
		// benchmarks show bundles produce more test cases and catch type errors on medium+ projects
		if e.Kind == "bundle" {
			switch complexity {
			case "medium":
				score *= 1.15
				reasons = append(reasons, "bundle helps medium project")
			case "large":
				score *= 1.3
				reasons = append(reasons, "bundle helps large project")
			}
		}

		// single skill bonus on trivial/small projects
		// benchmarks: single skill consistently cheapest with equal quality on simple tasks
		if e.Kind == "skill" && (complexity == "trivial" || complexity == "small") {
			score *= 1.2
			reasons = append(reasons, "skill ideal for "+complexity+" project")
		}

		// benchmark efficiency factor
		if len(e.Benchmarks) > 0 {
			var totalQuality float64
			var totalTokens float64
			for _, b := range e.Benchmarks {
				totalQuality += b.Quality
				totalTokens += float64(b.TokensIn + b.TokensOut)
			}
			avgQuality := totalQuality / float64(len(e.Benchmarks))
			avgTokens := totalTokens / float64(len(e.Benchmarks))
			if avgTokens > 0 {
				efficiency := avgQuality / (avgTokens / 10000)
				// normalize: efficiency ~0.03-0.5 from our data, scale to 0.8-1.2 multiplier
				benchFactor := 0.8 + (efficiency * 0.8)
				if benchFactor > 1.5 {
					benchFactor = 1.5
				}
				score *= benchFactor
				reasons = append(reasons, fmt.Sprintf("eff: %.1f qual, %dk tok", avgQuality, int(avgTokens)/1000))
			}
		}

		reason := strings.Join(reasons, "; ")
		candidates = append(candidates, scored{entry: e, score: score, reason: reason})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// resolve conflicts
	conflicted := make(map[string]bool)
	for _, c := range candidates {
		if conflicted[c.entry.ID] {
			continue
		}
		for _, conf := range c.entry.Lotus.ConflictsWith {
			conflicted[conf] = true
		}
		recs.Items = append(recs.Items, Recommendation{
			Entry:  c.entry,
			Score:  c.score,
			Reason: c.reason,
			Action: "add",
		})
	}

	return recs
}

func (r *Recommendations) Print() {
	if len(r.Items) == 0 {
		fmt.Println("No recommendations. Your project config looks complete.")
		return
	}

	fmt.Printf("Recommendations for %s\n", r.Profile.Path)
	fmt.Printf("Detected: %s | Complexity: %s (%d files, %d deps)\n\n",
		strings.Join(r.Profile.Languages(), ", "),
		r.Profile.Complexity.Level,
		r.Profile.Complexity.FileCount,
		r.Profile.Complexity.DepCount)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  SCORE\tACTION\tKIND\tID\tREASON\n")
	fmt.Fprintf(w, "  ─────\t──────\t────\t──\t──────\n")
	for _, rec := range r.Items {
		reason := rec.Reason
		if len(reason) > 70 {
			reason = reason[:67] + "..."
		}
		fmt.Fprintf(w, "  %.0f\t%s\t%s\t%s\t%s\n",
			rec.Score, rec.Action, rec.Entry.Kind, rec.Entry.ID, reason)
	}
	w.Flush()

	fmt.Printf("\n%d recommendations. Run `lotus apply .` to apply.\n", len(r.Items))
}

func existingSet(profile *analyzer.ProjectProfile) map[string]bool {
	set := make(map[string]bool)
	for _, s := range profile.ExistingConfig.Skills {
		set[s] = true
	}
	for _, a := range profile.ExistingConfig.Agents {
		set[a] = true
	}
	for _, m := range profile.ExistingConfig.MCPServers {
		set[m] = true
	}
	return set
}
