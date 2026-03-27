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

func Recommend(profile *analyzer.ProjectProfile, cat *catalog.Catalog) *Recommendations {
	recs := &Recommendations{Profile: profile}

	useCases := profile.InferUseCases()
	languages := profile.Languages()
	existing := existingSet(profile)

	// score each catalog entry
	type scored struct {
		entry  *catalog.Entry
		score  float64
		reason string
	}
	var candidates []scored

	for i := range cat.Entries {
		e := &cat.Entries[i]

		// skip if already configured
		if existing[e.ID] {
			continue
		}

		// skip "avoid" tier
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
					reasons = append(reasons, fmt.Sprintf("matches use case: %s", uc))
				}
			}
		}

		// stack match
		for _, es := range e.Stacks {
			for _, pl := range languages {
				if es == pl {
					score += 15
					reasons = append(reasons, fmt.Sprintf("matches stack: %s", es))
				}
			}
		}

		// stack-agnostic entries get a small bonus if they match a use case
		if len(e.Stacks) == 0 && score > 0 {
			score += 5
			reasons = append(reasons, "stack-agnostic (works with any stack)")
		}

		// tier bonus
		switch e.Lotus.Tier {
		case "recommended":
			score *= 1.5
		case "alternative":
			score *= 1.0
		}

		// weight penalty for heavy bundles
		switch e.Weight {
		case "heavy":
			score *= 0.7
		case "medium":
			score *= 0.9
		}

		if score > 0 {
			reason := strings.Join(reasons, "; ")
			candidates = append(candidates, scored{entry: e, score: score, reason: reason})
		}
	}

	// sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// resolve conflicts: if two entries conflict, keep the higher-scored one
	selected := make(map[string]bool)
	conflicted := make(map[string]bool)

	for _, c := range candidates {
		if conflicted[c.entry.ID] {
			continue
		}
		selected[c.entry.ID] = true
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
	fmt.Printf("Detected: %s\n\n", strings.Join(r.Profile.Languages(), ", "))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  SCORE\tACTION\tKIND\tID\tREASON\n")
	fmt.Fprintf(w, "  ─────\t──────\t────\t──\t──────\n")
	for _, rec := range r.Items {
		reason := rec.Reason
		if len(reason) > 60 {
			reason = reason[:57] + "..."
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
