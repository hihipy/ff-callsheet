// Package render turns a neutral league into the document a person or a model
// reads. It knows nothing about where the league came from.
package render

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

// Options control how much of the pool the document carries.
type Options struct {
	// Top is how many healthy free agents to list per position.
	Top int
}

// scoringKeys are the settings that change a lineup decision. The full blob
// runs to sixty keys, most of them zero, and costs more than it tells.
var scoringKeys = []string{"rec", "bonus_rec_te", "pass_td", "pass_int", "rush_td", "rec_td", "fum_lost"}

// Markdown renders the league. Headings give a model landmarks to navigate by,
// and rows stay unpadded because no human reads this in a terminal.
func Markdown(l callsheet.League, opts Options) string {
	if opts.Top <= 0 {
		opts.Top = 25
	}
	var b bytes.Buffer

	fmt.Fprintf(&b, "# %s\n\n", l.Name)
	fmt.Fprintf(&b, "%s %s, week %d | %d teams | %s | fetched %s\n\n",
		l.Season, l.SeasonType, l.Week, l.TeamCount, l.Ref.Platform,
		l.FetchedAt.Format("2006-01-02 15:04 MST"))
	fmt.Fprintf(&b, "Slots: %s\n\n", strings.Join(l.Slots, " "))

	if l.Waivers.Budget > 0 {
		fmt.Fprintf(&b, "FAAB budget: $%d | waiver_type %d\n\n", l.Waivers.Budget, l.Waivers.Type)
	} else {
		fmt.Fprintf(&b, "Waivers: priority order | waiver_type %d\n\n", l.Waivers.Type)
	}
	writeScoring(&b, l.Scoring)

	b.WriteString("## Rosters\n\n")
	for _, t := range l.Teams {
		writeTeam(&b, t, l.Waivers.Budget)
	}

	writeFreeAgents(&b, l, opts.Top)
	return b.String()
}

func writeScoring(b *bytes.Buffer, s map[string]float64) {
	var parts []string
	for _, k := range scoringKeys {
		if v, ok := s[k]; ok {
			parts = append(parts, fmt.Sprintf("%s %g", k, v))
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(b, "Scoring: %s\n\n", strings.Join(parts, " | "))
	}
}

func writeTeam(b *bytes.Buffer, t callsheet.Team, budget int) {
	fmt.Fprintf(b, "### %s\n\n", t.Manager)
	fmt.Fprintf(b, "%d-%d-%d", t.Wins, t.Losses, t.Ties)
	if budget > 0 {
		fmt.Fprintf(b, " | $%d FAAB left of $%d", budget-t.FAABUsed, budget)
	} else {
		fmt.Fprintf(b, " | waiver position %d", t.WaiverPosition)
	}
	b.WriteString("\n\n")

	writeGroup(b, "Starters", t.Starters)
	writeGroup(b, "Bench", t.Bench)
	writeGroup(b, "Reserve", t.Reserve)
	writeGroup(b, "Taxi", t.Taxi)
	b.WriteString("\n")
}

// writeGroup renders a roster group on one line. A table per group would cost a
// header row once per team for information three fields wide.
func writeGroup(b *bytes.Buffer, label string, list []callsheet.Player) {
	if len(list) == 0 {
		return
	}
	parts := make([]string, 0, len(list))
	for _, p := range list {
		parts = append(parts, strings.TrimSpace(
			fmt.Sprintf("%s %s %s", p.Name, p.Position, p.Team))+suffix(p))
	}
	fmt.Fprintf(b, "%s: %s\n", label, strings.Join(parts, " | "))
}

// suffix renders the trailing detail a player line carries. A bye comes first
// and suppresses the points, because a bye overrides anything a reader would
// otherwise conclude from them.
func suffix(p callsheet.Player) string {
	var bits []string
	if p.OnBye {
		bits = append(bits, "BYE")
	}
	if p.HasPoints && !p.OnBye {
		bits = append(bits, fmt.Sprintf("%.1f", p.Points))
	}
	if p.Injury != "" {
		bits = append(bits, p.Injury)
	}
	if len(bits) == 0 {
		return ""
	}
	return " (" + strings.Join(bits, ", ") + ")"
}

func writeFreeAgents(b *bytes.Buffer, l callsheet.League, top int) {
	pool, order := l.PoolByPosition()

	var total, slotted int
	for _, list := range pool {
		for _, p := range list {
			total++
			if p.Slotted {
				slotted++
			}
		}
	}

	ordering := "preseason rank"
	if l.Projected {
		ordering = "projected " + l.PointsKey + " for this week"
		if l.PointsApproximate {
			// The platform does not publish a basis matching this league's
			// scoring, and a reader handing the file to an assistant should
			// not read the number as exact.
			ordering += ", the nearest basis available rather than this league's exact scoring"
		}
	}
	// A provider whose platform exposes no roster-slot signal leaves Slotted
	// false for everyone. Reporting "0 on a depth chart" would read as a fact
	// about the league rather than a gap in the source, so the split is omitted.
	// The count is of the pool after position filtering, so the label has to
	// say so. A provider logs a wider number under the same word, and in a
	// league with no kicker slot the two differ by every available kicker.
	if slotted > 0 {
		fmt.Fprintf(b, "## Free agents\n\n%d unrostered at positions this league fields, %d of them on a club depth chart and %d on the practice squad or newly signed. Top %d healthy per position, ordered by %s.\n\n",
			total, slotted, total-slotted, top, ordering)
	} else {
		fmt.Fprintf(b, "## Free agents\n\n%d unrostered at positions this league fields. Top %d healthy per position, ordered by %s.\n\n",
			total, top, ordering)
	}

	for _, pos := range order {
		list := pool[pos]
		if len(list) == 0 {
			continue
		}
		if n := countSlotted(list); n > 0 {
			fmt.Fprintf(b, "### %s (%d available, %d on a depth chart)\n\n", pos, len(list), n)
		} else {
			fmt.Fprintf(b, "### %s (%d available)\n\n", pos, len(list))
		}

		healthy := make([]string, 0, top)
		for _, p := range list {
			if len(healthy) == top {
				break
			}
			if p.Injury == "" {
				healthy = append(healthy, p.Name+" "+p.Team+suffix(p))
			}
		}
		fmt.Fprintf(b, "%s\n", strings.Join(healthy, " | "))

		// Injured players get their own pass, ordered by preseason rank. An IR
		// stash has no projection this week, so the projection ordering buries
		// exactly the names worth stashing.
		var injuredAll []callsheet.Player
		for _, p := range list {
			if p.Injury != "" {
				injuredAll = append(injuredAll, p)
			}
		}
		cap := top/3 + 1
		var injured []string
		for _, p := range callsheet.ByRank(injuredAll) {
			if len(injured) == cap {
				break
			}
			injured = append(injured, p.Name+" "+p.Team+suffix(p))
		}
		if len(injured) > 0 {
			fmt.Fprintf(b, "\nInjured: %s\n", strings.Join(injured, " | "))
		}
		b.WriteString("\n")
	}
}

func countSlotted(list []callsheet.Player) int {
	n := 0
	for _, p := range list {
		if p.Slotted {
			n++
		}
	}
	return n
}
