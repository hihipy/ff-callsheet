package render

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

// update rewrites the golden file instead of comparing against it:
//
//	go test ./internal/render -update
//
// Review the diff before committing it. A golden test is only worth having if
// changing it is a deliberate act.
var update = flag.Bool("update", false, "rewrite golden files")

// goldenLeague exercises every branch the renderer has: both waiver kinds via
// two cases, byes, injuries, unprojected players, an empty starting slot, and
// a position the league does not field.
func goldenLeague() callsheet.League {
	return callsheet.League{
		Ref:        callsheet.Ref{Platform: "sleeper", Sport: "nfl", ID: "L1"},
		Name:       "Golden League",
		Season:     "2026",
		SeasonType: "regular",
		Week:       7,
		TeamCount:  2,
		Slots:      []string{"QB", "RB", "FLEX", "DEF", "BN", "IR"},
		Scoring:    map[string]float64{"rec": 1, "bonus_rec_te": 0.5, "pass_td": 4, "pass_int": -2},
		Waivers:    callsheet.Waivers{Budget: 100, Type: 2},
		FetchedAt:  time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC),
		Projected:  true,
		PointsKey:  "pts_ppr",
		Teams: []callsheet.Team{
			{
				ID: "1", Manager: "Alice", Wins: 4, Losses: 2, FAABUsed: 35,
				Starters: []callsheet.Player{
					{Name: "Anna Arm", Position: "QB", Team: "KC", Points: 22.4, HasPoints: true},
					{Name: "(empty)", Position: "", Team: ""},
				},
				Bench: []callsheet.Player{
					{Name: "Bye Back", Position: "RB", Team: "MIA", Points: 10, HasPoints: true, OnBye: true},
					{Name: "Hurt Hands", Position: "WR", Team: "NYJ", Injury: "Questionable", Points: 7.2, HasPoints: true},
				},
				Reserve: []callsheet.Player{
					{Name: "Stashed Star", Position: "WR", Team: "ARI", Injury: "IR"},
				},
			},
			{
				ID: "2", Manager: "Bob", Wins: 2, Losses: 4, FAABUsed: 0,
				Starters: []callsheet.Player{
					{Name: "Cal Carry", Position: "RB", Team: "SF", Points: 14.8, HasPoints: true},
				},
				Taxi: []callsheet.Player{
					{Name: "Rookie Wheels", Position: "RB", Team: "GB", Rank: 400},
				},
			},
		},
		Pool: []callsheet.Player{
			{ID: "1", Name: "Top Option", Position: "RB", Team: "DAL", Points: 9.9, HasPoints: true, Slotted: true, Rank: 120},
			{ID: "2", Name: "Second Option", Position: "RB", Team: "CHI", Points: 6.25, HasPoints: true, Slotted: true, Rank: 80},
			{ID: "3", Name: "No Projection", Position: "RB", Team: "NO", Slotted: true, Rank: 60},
			{ID: "4", Name: "Practice Body", Position: "RB", Team: "LV", Rank: callsheet.UnrankedRank},
			{ID: "5", Name: "Injured Stash", Position: "RB", Team: "ARI", Injury: "IR", Slotted: true, Rank: 30},
			{ID: "6", Name: "Bye Option", Position: "QB", Team: "MIA", Points: 18, HasPoints: true, OnBye: true, Slotted: true, Rank: 90},
			{ID: "7", Name: "Flex Fullback", Position: "FB", Eligible: []string{"RB"}, Team: "SF", Slotted: true, Rank: 500},
			{ID: "8", Name: "Not Fielded", Position: "OL", Eligible: []string{"OL"}, Team: "GB", Slotted: true, Rank: 700},
			{ID: "DAL", Name: "Dallas Cowboys", Position: "DEF", Team: "DAL", Points: 8.1, HasPoints: true, Rank: callsheet.UnrankedRank},
		},
	}
}

func TestGoldenMarkdown(t *testing.T) {
	got := Markdown(goldenLeague(), Options{Top: 3})
	path := filepath.Join("testdata", "golden.md")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden file rewritten")
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run: go test ./internal/render -update)", err)
	}
	if got != string(want) {
		t.Errorf("output drifted from testdata/golden.md\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestGoldenWaiverPriority covers the other waiver branch, which the main
// golden league cannot show at the same time.
func TestGoldenWaiverPriority(t *testing.T) {
	l := goldenLeague()
	l.Waivers = callsheet.Waivers{Budget: 0, Type: 0}
	l.Teams[0].WaiverPosition = 6
	out := Markdown(l, Options{Top: 3})
	if !contains(out, "Waivers: priority order") || !contains(out, "waiver position 6") {
		t.Error("priority waiver rendering missing")
	}
	if contains(out, "FAAB") {
		t.Error("FAAB language leaked into a priority league")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
