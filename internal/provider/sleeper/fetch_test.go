package sleeper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

// fakeSleeper serves the testdata fixtures on the same paths the real API
// uses, so Fetch runs its whole path rather than its pure parts.
func fakeSleeper(t *testing.T, fail map[string]bool) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	mux := http.NewServeMux()
	serve := func(pattern, file string) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if fail[file] {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			raw, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Errorf("fixture %s: %v", file, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(raw)
		})
	}
	serve("/v1/league/L1", "league.json")
	serve("/v1/league/L1/rosters", "rosters.json")
	serve("/v1/league/L1/users", "users.json")
	serve("/v1/state/nfl", "state.json")
	serve("/v1/players/nfl", "players.json")
	serve("/projections/nfl/2026/3", "projections.json")

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &calls
}

func fetchFixture(t *testing.T, fail map[string]bool) (callsheet.League, *int) {
	t.Helper()
	srv, calls := fakeSleeper(t, fail)
	p := Provider{
		http:     newHTTPClient(),
		urls:     endpoints{api: srv.URL + "/v1", proj: srv.URL},
		CacheDir: t.TempDir(),
	}
	lg, err := p.Fetch(context.Background(), callsheet.Ref{Platform: "sleeper", Sport: "nfl", ID: "L1"}, callsheet.Options{})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	return lg, calls
}

func TestFetchMapsLeagueMetadata(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	if lg.Name != "Test Superflex League" || lg.Week != 3 || lg.SeasonType != "regular" {
		t.Errorf("metadata: %+v", lg)
	}
	if lg.Waivers.Budget != 100 || lg.Waivers.Type != 2 {
		t.Errorf("waivers: %+v", lg.Waivers)
	}
	if !lg.Projected || lg.PointsKey != "pts_ppr" {
		t.Errorf("projections: projected=%v key=%s", lg.Projected, lg.PointsKey)
	}
	if !lg.InSeason() {
		t.Error("a regular season league should be in season")
	}
}

func TestFetchOrdersTeamsAndResolvesNames(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	if len(lg.Teams) != 2 {
		t.Fatalf("teams: %d", len(lg.Teams))
	}
	// Rosters arrive out of order in the fixture and must come back by ID.
	if lg.Teams[0].ID != "1" || lg.Teams[1].ID != "2" {
		t.Errorf("team order: %s, %s", lg.Teams[0].ID, lg.Teams[1].ID)
	}
	// A team_name wins over the display name, and a missing one falls back.
	if lg.Teams[0].Manager != "Alice's Team" || lg.Teams[1].Manager != "bob" {
		t.Errorf("managers: %q, %q", lg.Teams[0].Manager, lg.Teams[1].Manager)
	}
}

func TestFetchSplitsRosterGroups(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	bob := lg.Teams[1]
	if len(bob.Starters) != 2 || bob.Starters[1].Name != "(empty)" {
		t.Errorf("an empty starting slot should render: %+v", bob.Starters)
	}
	if len(bob.Bench) != 1 || bob.Bench[0].Name != "Rostered Bench" {
		t.Errorf("bench: %+v", bob.Bench)
	}
	if len(bob.Reserve) != 1 || bob.Reserve[0].Injury != "IR" {
		t.Errorf("reserve: %+v", bob.Reserve)
	}
	if len(lg.Teams[0].Taxi) != 1 {
		t.Errorf("taxi: %+v", lg.Teams[0].Taxi)
	}
}

func TestFetchExcludesRosteredFromPool(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	for _, p := range lg.Pool {
		switch p.ID {
		case "p1", "p2", "p3", "p4", "p5":
			t.Errorf("%s is rostered and must not be a free agent", p.Name)
		}
	}
}

func TestFetchDropsPlayersNoClubLists(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	for _, p := range lg.Pool {
		if p.ID == "f6" {
			t.Error("a retired player whose status still reads Active leaked into the pool")
		}
	}
	// A practice squad player with no slot but recent news is kept.
	if !poolHas(lg, "f5") {
		t.Error("a recently signed unslotted player should survive")
	}
}

func TestFetchAttachesProjectionsAndEligibility(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	pool, order := lg.PoolByPosition()

	// The league fields no kicker and no linemen, so an OL must not appear.
	for _, pos := range order {
		if pos == "OL" || pos == "K" {
			t.Errorf("position %s is not fielded by this league", pos)
		}
	}
	rb := pool["RB"]
	if len(rb) == 0 {
		t.Fatal("no running backs in the pool")
	}
	// Projected beats unprojected regardless of preseason rank.
	if rb[0].Name != "Free Projected" || !rb[0].HasPoints || rb[0].Points != 8.5 {
		t.Errorf("ordering or points: %+v", rb[0])
	}
	// A fullback is eligible at running back and belongs in that bucket.
	if !bucketHas(rb, "Free Fullback") {
		t.Error("a fullback eligible at RB should be in the RB bucket")
	}
}

func TestFetchSurvivesProjectionFailure(t *testing.T) {
	lg, _ := fetchFixture(t, map[string]bool{"projections.json": true})
	if lg.Projected {
		t.Error("a failed projection fetch should leave Projected false")
	}
	if len(lg.Pool) == 0 || len(lg.Teams) == 0 {
		t.Error("the league should still render without projections")
	}
	for _, p := range lg.Pool {
		if p.HasPoints {
			t.Errorf("%s carries points after a failed fetch", p.Name)
		}
	}
}

func TestFetchFailsLoudlyOnRequiredEndpoints(t *testing.T) {
	srv, _ := fakeSleeper(t, map[string]bool{"rosters.json": true})
	p := Provider{http: newHTTPClient(), urls: endpoints{api: srv.URL + "/v1", proj: srv.URL}, CacheDir: t.TempDir()}
	_, err := p.Fetch(context.Background(), callsheet.Ref{Sport: "nfl", ID: "L1"}, callsheet.Options{})
	if err == nil {
		t.Fatal("a failed rosters call must not be swallowed")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should name the status: %v", err)
	}
}

func TestPlayerMapIsCachedBetweenRuns(t *testing.T) {
	srv, calls := fakeSleeper(t, nil)
	dir := t.TempDir()
	p := Provider{http: newHTTPClient(), urls: endpoints{api: srv.URL + "/v1", proj: srv.URL}, CacheDir: dir}
	ref := callsheet.Ref{Sport: "nfl", ID: "L1"}
	if _, err := p.Fetch(context.Background(), ref, callsheet.Options{}); err != nil {
		t.Fatal(err)
	}
	first := *calls
	if _, err := p.Fetch(context.Background(), ref, callsheet.Options{}); err != nil {
		t.Fatal(err)
	}
	// The second run makes one fewer call, because the 5MB player map is read
	// from disk rather than refetched.
	if got := *calls - first; got != 5 {
		t.Errorf("second run made %d calls, want 5 with the player map cached", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "players-nfl.json")); err != nil {
		t.Errorf("cache file missing: %v", err)
	}
}

func poolHas(lg callsheet.League, id string) bool {
	for _, p := range lg.Pool {
		if p.ID == id {
			return true
		}
	}
	return false
}

func bucketHas(list []callsheet.Player, name string) bool {
	for _, p := range list {
		if p.Name == name {
			return true
		}
	}
	return false
}

func TestBenchExcludesReserveAndTaxi(t *testing.T) {
	lg, _ := fetchFixture(t, nil)
	for _, team := range lg.Teams {
		seen := map[string]string{}
		for _, group := range []struct {
			label string
			list  []callsheet.Player
		}{
			{"starters", team.Starters}, {"bench", team.Bench},
			{"reserve", team.Reserve}, {"taxi", team.Taxi},
		} {
			for _, p := range group.list {
				if p.ID == "0" {
					continue
				}
				if prev, dup := seen[p.ID]; dup {
					t.Errorf("%s: %s appears in both %s and %s", team.Manager, p.Name, prev, group.label)
				}
				seen[p.ID] = group.label
			}
		}
	}
}
