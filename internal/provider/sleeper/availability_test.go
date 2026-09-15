package sleeper

import (
	"testing"
	"time"
)

func ptrInt(v int) *int    { return &v }
func ptrMs(v int64) *int64 { return &v }

// These records are taken from a real Sleeper player map, which is why the
// timestamps look arbitrary. now matches the day they were pulled.
var (
	now    = time.UnixMilli(1789392004000)
	window = DefaultNewsWindow
)

func TestAvailableOnRealRecords(t *testing.T) {
	cases := []struct {
		name string
		p    player
		want bool
	}{
		{"Njoku, slotted and fresh", player{Position: "TE", Team: "LAC", DepthChartOrder: ptrInt(1), NewsUpdated: ptrMs(1789392004000)}, true},
		{"Jauan Jennings, third on the chart", player{Position: "WR", Team: "MIN", DepthChartOrder: ptrInt(3), NewsUpdated: ptrMs(1789363561224)}, true},
		{"unslotted but signed five days ago", player{Position: "WR", Team: "LAC", NewsUpdated: ptrMs(1788853848544)}, true},
		{"unslotted and forty-four days stale", player{Position: "OL", Team: "GB", NewsUpdated: ptrMs(1785457226217)}, false},
		{"slotted with news 139 days stale", player{Position: "LB", Team: "NO", DepthChartOrder: ptrInt(2), NewsUpdated: ptrMs(1777271737281)}, true},
		{"retired, still reads Active upstream", player{Position: "QB", Team: "PIT", NewsUpdated: ptrMs(1643296817250)}, false},
		{"retired, no club", player{Position: "RB", NewsUpdated: ptrMs(1512750001181)}, false},
		{"team defense", player{Position: "DEF", Team: "SF"}, true},
	}
	for _, c := range cases {
		if got := available(c.p, now, window); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestToPlayerResolvesOptionals(t *testing.T) {
	p := player{PlayerID: "1", FirstName: "Kyle", LastName: "Juszczyk", Position: "FB",
		FantasyPositions: []string{"RB"}, Team: "SF"}
	got := p.toPlayer()
	if got.Name != "Kyle Juszczyk" || got.Rank == 0 {
		t.Errorf("unexpected flatten: %+v", got)
	}
	if got.Eligible[0] != "RB" {
		t.Errorf("eligibility lost: %v", got.Eligible)
	}
	// An absent fantasy_positions falls back to the primary position rather
	// than leaving the player eligible nowhere.
	bare := player{Position: "K", Team: "DAL"}.toPlayer()
	if len(bare.Eligible) != 1 || bare.Eligible[0] != "K" {
		t.Errorf("fallback eligibility: %v", bare.Eligible)
	}
}

func TestPointsKeyCoversUnusualScoring(t *testing.T) {
	cases := []struct {
		rec   float64
		key   string
		exact bool
	}{
		{1, "pts_ppr", true},
		{0.5, "pts_half_ppr", true},
		{0, "pts_std", true},
		// Sleeper projects only three scoring bases, so anything else takes
		// the nearest and is reported as approximate.
		{1.5, "pts_ppr", false},
		{0.75, "pts_ppr", false},
		{0.25, "pts_half_ppr", false},
		{0.1, "pts_std", false},
	}
	for _, c := range cases {
		key, exact := pointsKey(map[string]float64{"rec": c.rec})
		if key != c.key || exact != c.exact {
			t.Errorf("rec %g: got %s exact=%v, want %s exact=%v", c.rec, key, exact, c.key, c.exact)
		}
	}
}
