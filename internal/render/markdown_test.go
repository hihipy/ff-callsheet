package render

import (
	"strings"
	"testing"
	"time"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

func sample() callsheet.League {
	return callsheet.League{
		Ref:        callsheet.Ref{Platform: "sleeper", Sport: "nfl", ID: "1"},
		Name:       "Test League",
		Season:     "2026",
		SeasonType: "regular",
		Week:       3,
		TeamCount:  2,
		Slots:      []string{"QB", "RB", "FLEX", "BN"},
		Scoring:    map[string]float64{"rec": 1, "pass_td": 4},
		Waivers:    callsheet.Waivers{Budget: 100, Type: 2},
		FetchedAt:  time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC),
		Projected:  true,
		PointsKey:  "pts_ppr",
		Teams: []callsheet.Team{{
			Manager: "alice", Wins: 2, Losses: 1, FAABUsed: 25,
			Starters: []callsheet.Player{
				{Name: "A Quarterback", Position: "QB", Team: "KC", Points: 20.5, HasPoints: true},
			},
			Bench: []callsheet.Player{
				{Name: "A Bye Back", Position: "RB", Team: "MIA", Points: 9, HasPoints: true, OnBye: true},
			},
		}},
		Pool: []callsheet.Player{
			{Name: "Healthy Back", Position: "RB", Team: "SF", Points: 8, HasPoints: true, Slotted: true, Rank: 100},
			{Name: "Stashed Back", Position: "RB", Team: "ARI", Injury: "IR", Slotted: true, Rank: 40},
			{Name: "Deep Back", Position: "RB", Team: "NYJ", Rank: callsheet.UnrankedRank},
		},
	}
}

func TestMarkdownStructure(t *testing.T) {
	out := Markdown(sample(), Options{Top: 5})
	for _, want := range []string{
		"# Test League",
		"FAAB budget: $100",
		"$75 FAAB left of $100",
		"Scoring: rec 1 | pass_td 4",
		"## Rosters",
		"### alice",
		"## Free agents",
		"ordered by projected pts_ppr for this week",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestByeSuppressesPoints(t *testing.T) {
	out := Markdown(sample(), Options{Top: 5})
	if !strings.Contains(out, "A Bye Back RB MIA (BYE)") {
		t.Error("bye marker missing or points not suppressed")
	}
	if strings.Contains(out, "(BYE, 9.0)") {
		t.Error("points rendered alongside a bye")
	}
}

func TestInjuredListedSeparatelyAndByRank(t *testing.T) {
	out := Markdown(sample(), Options{Top: 5})
	// The stash outranks the healthy back but must not appear in the healthy
	// line, and must appear in the injured line.
	healthyLine := lineContaining(out, "Healthy Back")
	if strings.Contains(healthyLine, "Stashed Back") {
		t.Error("an injured player leaked into the healthy line")
	}
	if !strings.Contains(out, "Injured: Stashed Back ARI (IR)") {
		t.Error("injured line missing the stash")
	}
}

func TestUnprojectedSortsAfterProjected(t *testing.T) {
	out := Markdown(sample(), Options{Top: 5})
	line := lineContaining(out, "Healthy Back")
	if strings.Index(line, "Healthy Back") > strings.Index(line, "Deep Back") {
		t.Errorf("projected player should lead: %q", line)
	}
}

func lineContaining(s, want string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, want) {
			return l
		}
	}
	return ""
}

func TestSlotSplitOmittedWhenProviderHasNoSignal(t *testing.T) {
	l := sample()
	// A provider whose platform exposes no depth chart leaves Slotted false.
	for i := range l.Pool {
		l.Pool[i].Slotted = false
	}
	out := Markdown(l, Options{Top: 5})
	if strings.Contains(out, "on a club depth chart") {
		t.Error("reported a depth chart split the provider never supplied")
	}
	if strings.Contains(out, "on a depth chart)") {
		t.Error("position heading claimed a depth chart count of zero")
	}
	if !strings.Contains(out, "unrostered. Top 5 healthy per position") {
		t.Error("fallback header missing")
	}
}

func TestSlotSplitShownWhenProviderSuppliesIt(t *testing.T) {
	out := Markdown(sample(), Options{Top: 5})
	if !strings.Contains(out, "on a club depth chart") {
		t.Error("split missing when the provider supplied it")
	}
}
