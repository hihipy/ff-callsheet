package callsheet

import (
	"reflect"
	"testing"
)

func TestStartablePositionsAcrossLeagueTypes(t *testing.T) {
	cases := []struct {
		name  string
		slots []string
		want  []string
	}{
		{
			"superflex, no kicker",
			[]string{"QB", "RB", "WR", "TE", "FLEX", "FLEX", "SUPER_FLEX", "DEF", "BN", "BN"},
			[]string{"QB", "RB", "WR", "TE", "DEF"},
		},
		{
			"standard with kicker",
			[]string{"QB", "RB", "RB", "WR", "WR", "TE", "FLEX", "K", "DEF", "BN"},
			[]string{"QB", "RB", "WR", "TE", "K", "DEF"},
		},
		{
			"IDP with explicit slots",
			[]string{"QB", "RB", "WR", "TE", "FLEX", "DL", "LB", "DB", "BN", "IR"},
			[]string{"QB", "RB", "WR", "TE", "DL", "LB", "DB"},
		},
		{
			"IDP flex",
			[]string{"QB", "RB", "WR", "TE", "IDP_FLEX", "BN"},
			[]string{"QB", "RB", "WR", "TE", "DL", "LB", "DB"},
		},
		{
			"a slot nobody has written down yet",
			[]string{"QB", "RB", "WR", "TE", "EDGE", "BN", "TAXI"},
			[]string{"QB", "RB", "WR", "TE", "EDGE"},
		},
	}
	for _, c := range cases {
		got := OrderedPositions(StartablePositions(c.slots))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}

func TestStructuralSlotsFieldNothing(t *testing.T) {
	if got := StartablePositions([]string{"BN", "IR", "TAXI"}); len(got) != 0 {
		t.Errorf("structural slots should field nothing, got %v", got)
	}
}

func TestBucketUsesEligibility(t *testing.T) {
	league := map[string]bool{"QB": true, "RB": true}
	// A fullback is listed at FB and eligible at RB, which is where Sleeper
	// will let you start him.
	fb := Player{Position: "FB", Eligible: []string{"RB"}}
	if got, ok := fb.Bucket(league); !ok || got != "RB" {
		t.Errorf("fullback: got %q %v, want RB true", got, ok)
	}
	ol := Player{Position: "OL", Eligible: []string{"OL"}}
	if _, ok := ol.Bucket(league); ok {
		t.Error("an offensive lineman should not enter this pool")
	}
}

func TestPoolOrderingPrefersProjections(t *testing.T) {
	l := League{
		Slots: []string{"RB", "BN"},
		Pool: []Player{
			{Name: "unranked no points", Position: "RB", Rank: UnrankedRank},
			{Name: "ranked no points", Position: "RB", Rank: 50},
			{Name: "low points", Position: "RB", Rank: 10, Points: 3, HasPoints: true},
			{Name: "high points", Position: "RB", Rank: 900, Points: 12, HasPoints: true},
		},
	}
	pool, order := l.PoolByPosition()
	if !reflect.DeepEqual(order, []string{"RB"}) {
		t.Fatalf("order: got %v", order)
	}
	want := []string{"high points", "low points", "ranked no points", "unranked no points"}
	for i, p := range pool["RB"] {
		if p.Name != want[i] {
			t.Errorf("position %d: got %q want %q", i, p.Name, want[i])
		}
	}
}

func TestInSeasonStopCondition(t *testing.T) {
	for _, c := range []struct {
		season string
		want   bool
	}{{"regular", true}, {"post", true}, {"pre", false}, {"off", false}} {
		if got := (League{SeasonType: c.season}).InSeason(); got != c.want {
			t.Errorf("%s: got %v want %v", c.season, got, c.want)
		}
	}
}

func TestOrderingIsDeterministicForTiedPlayers(t *testing.T) {
	// Two different players share a name, and neither is ranked or projected.
	// Every sort key ties except the ID, which has to settle it the same way
	// on every run.
	l := League{
		Slots: []string{"RB", "BN"},
		Pool: []Player{
			{ID: "b", Name: "Same Name", Position: "RB", Rank: UnrankedRank},
			{ID: "a", Name: "Same Name", Position: "RB", Rank: UnrankedRank},
		},
	}
	for i := 0; i < 50; i++ {
		pool, _ := l.PoolByPosition()
		if pool["RB"][0].ID != "a" {
			t.Fatalf("run %d put %s first", i, pool["RB"][0].ID)
		}
		ranked := ByRank(l.Pool)
		if ranked[0].ID != "a" {
			t.Fatalf("ByRank run %d put %s first", i, ranked[0].ID)
		}
	}
}
