package sleeper

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

// Provider implements callsheet.Provider against Sleeper.
type Provider struct {
	http httpClient
	urls endpoints
	// Log receives progress and warnings. Nil discards them.
	Log io.Writer
	// CacheDir overrides where the player map is cached. Empty uses the
	// user cache directory, which is what a real run wants.
	CacheDir string
}

// New returns a Provider writing diagnostics to log, which may be nil.
func New(log io.Writer) Provider {
	return Provider{http: newHTTPClient(), urls: defaultEndpoints(), Log: log}
}

func (p Provider) Name() string { return "sleeper" }

func (p Provider) logf(format string, args ...any) {
	if p.Log == nil {
		return
	}
	fmt.Fprintf(p.Log, format+"\n", args...)
}

// Fetch pulls one league and returns it in neutral form. Free agency on
// Sleeper is a set difference rather than an endpoint, so the whole player map
// is loaded and every rostered ID subtracted from it.
func (p Provider) Fetch(ctx context.Context, ref callsheet.Ref, opts callsheet.Options) (callsheet.League, error) {
	sport := ref.Sport
	if sport == "" {
		sport = "nfl"
	}
	window := opts.StaleWindow
	if window == 0 {
		window = DefaultNewsWindow
	}

	var (
		lg      league
		rosters []roster
		users   []user
		st      state
	)
	base := fmt.Sprintf("%s/league/%s", p.urls.api, ref.ID)
	if err := p.http.getJSON(ctx, base, &lg); err != nil {
		return callsheet.League{}, err
	}
	if err := p.http.getJSON(ctx, base+"/rosters", &rosters); err != nil {
		return callsheet.League{}, err
	}
	if err := p.http.getJSON(ctx, base+"/users", &users); err != nil {
		return callsheet.League{}, err
	}
	if err := p.http.getJSON(ctx, fmt.Sprintf("%s/state/%s", p.urls.api, sport), &st); err != nil {
		return callsheet.League{}, err
	}
	players, err := p.loadPlayers(ctx, sport)
	if err != nil {
		return callsheet.League{}, err
	}

	out := callsheet.League{
		Ref:        callsheet.Ref{Platform: p.Name(), Sport: sport, ID: ref.ID},
		Name:       lg.Name,
		Season:     lg.Season,
		SeasonType: st.SeasonType,
		Week:       st.Week,
		TeamCount:  lg.TotalRosters,
		Slots:      lg.RosterPositions,
		Scoring:    lg.ScoringSettings,
		Waivers:    callsheet.Waivers{Budget: lg.Settings.WaiverBudget, Type: lg.Settings.WaiverType},
		FetchedAt:  time.Now(),
	}

	// Projections are optional and carry byes with them, so they are fetched
	// before rendering but never block it.
	var proj projections
	if !opts.SkipProjections {
		playerTeam := make(map[string]string, len(players))
		for id, pl := range players {
			playerTeam[id] = pl.Team
		}
		positions := callsheet.OrderedPositions(callsheet.StartablePositions(lg.RosterPositions))
		proj = p.fetchProjections(ctx, sport, st.Season, st.Week, lg.ScoringSettings, positions, playerTeam)
		out.Projected = proj.fetched
		out.PointsKey = proj.key
	}

	// Rosters carry owner_id only, so manager names come from the users call.
	names := make(map[string]string, len(users))
	for _, u := range users {
		label := u.Metadata.TeamName
		if label == "" {
			label = u.DisplayName
		}
		names[u.UserID] = label
	}

	rostered := make(map[string]bool)
	sort.Slice(rosters, func(i, j int) bool { return rosters[i].RosterID < rosters[j].RosterID })
	for _, r := range rosters {
		for _, group := range [][]string{r.Players, r.Reserve, r.Taxi} {
			for _, id := range group {
				rostered[id] = true
			}
		}
		manager := names[r.OwnerID]
		if manager == "" {
			manager = "(unclaimed team)"
		}
		starting := make(map[string]bool, len(r.Starters))
		for _, id := range r.Starters {
			starting[id] = true
		}
		var bench []string
		for _, id := range r.Players {
			if !starting[id] {
				bench = append(bench, id)
			}
		}
		out.Teams = append(out.Teams, callsheet.Team{
			ID:             fmt.Sprint(r.RosterID),
			Manager:        manager,
			Wins:           r.Settings.Wins,
			Losses:         r.Settings.Losses,
			Ties:           r.Settings.Ties,
			FAABUsed:       r.Settings.WaiverBudgetUsed,
			WaiverPosition: r.Settings.WaiverPosition,
			Starters:       resolve(r.Starters, players, proj),
			Bench:          resolve(bench, players, proj),
			Reserve:        resolve(r.Reserve, players, proj),
			Taxi:           resolve(r.Taxi, players, proj),
		})
	}

	now := time.Now()
	var slotted, dropped int
	for id, pl := range players {
		if rostered[id] {
			continue
		}
		if !available(pl, now, window) {
			dropped++
			continue
		}
		out.Pool = append(out.Pool, withProjection(pl.toPlayer(), proj))
		if pl.onDepthChart() {
			slotted++
		}
	}
	// This count spans every position Sleeper tracks, including linemen and
	// kickers a league may not field. The document's own count is smaller,
	// because position filtering happens against the league's slots.
	p.logf("pool: %d unrostered and currently listed at any position, %d on a club depth chart, %d excluded",
		len(out.Pool), slotted, dropped)

	return out, nil
}

// resolve turns a list of Sleeper IDs into neutral players. An ID with no
// record still renders, because a silent gap is worse than an odd line.
func resolve(ids []string, players map[string]player, proj projections) []callsheet.Player {
	out := make([]callsheet.Player, 0, len(ids))
	for _, id := range ids {
		pl, ok := players[id]
		if !ok {
			// An empty starting slot arrives as "0" rather than a player ID.
			name := id
			if id == "0" {
				name = "(empty)"
			}
			out = append(out, callsheet.Player{ID: id, Name: name, Rank: callsheet.UnrankedRank})
			continue
		}
		out = append(out, withProjection(pl.toPlayer(), proj))
	}
	return out
}

func withProjection(p callsheet.Player, proj projections) callsheet.Player {
	p.OnBye = proj.onBye[p.Team]
	if v, ok := proj.points[p.ID]; ok {
		p.Points = v
		p.HasPoints = true
	}
	return p
}
