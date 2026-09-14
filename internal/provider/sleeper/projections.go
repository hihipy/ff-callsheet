package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// projRow is one player's projection for a week. Only the fields the renderer
// needs are decoded, because the payload carries far more and its shape is not
// contractual.
type projRow struct {
	PlayerID string             `json:"player_id"`
	Team     string             `json:"team"`
	Stats    map[string]float64 `json:"stats"`
}

// projections holds a week's projected points by player ID, plus the clubs that
// appear at all. A club with no rows is on bye, which is how bye weeks are
// derived without a schedule source.
type projections struct {
	points  map[string]float64
	onBye   map[string]bool
	key     string
	fetched bool
}

// nflTeams is the fixed set of club abbreviations Sleeper uses. It changes only
// when a franchise relocates, so a literal is cheaper than another request.
var nflTeams = []string{
	"ARI", "ATL", "BAL", "BUF", "CAR", "CHI", "CIN", "CLE", "DAL", "DEN",
	"DET", "GB", "HOU", "IND", "JAX", "KC", "LAC", "LAR", "LV", "MIA",
	"MIN", "NE", "NO", "NYG", "NYJ", "PHI", "PIT", "SEA", "SF", "TB",
	"TEN", "WAS",
}

// pointsKey picks the projection field matching the league's own scoring, so a
// full PPR league is never handed half PPR numbers.
func pointsKey(scoring map[string]float64) string {
	switch scoring["rec"] {
	case 1:
		return "pts_ppr"
	case 0.5:
		return "pts_half_ppr"
	default:
		return "pts_std"
	}
}

// fetchProjections asks for one week. A failure here is reported and survived,
// because the endpoint is undocumented and may change without notice.
func (p Provider) fetchProjections(ctx context.Context, sport, season string, week int,
	scoring map[string]float64, positions []string, playerTeam map[string]string) projections {

	pr := projections{
		points: map[string]float64{},
		onBye:  map[string]bool{},
		key:    pointsKey(scoring),
	}

	var q strings.Builder
	for _, pos := range positions {
		q.WriteString("&position[]=")
		q.WriteString(pos)
	}
	url := fmt.Sprintf("%s/projections/%s/%s/%d?season_type=regular%s&order_by=%s",
		p.urls.proj, sport, season, week, q.String(), pr.key)

	raw, err := p.http.getBytes(ctx, url)
	if err != nil {
		p.logf("warning: projections unavailable, continuing without them: %v", err)
		return pr
	}

	// The payload has been seen as an array and could equally be keyed by
	// player ID, so both are accepted rather than assumed.
	var rows []projRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		var byID map[string]projRow
		if err2 := json.Unmarshal(raw, &byID); err2 != nil {
			p.logf("warning: projections in an unrecognized shape, continuing without them")
			return pr
		}
		for id, r := range byID {
			if r.PlayerID == "" {
				r.PlayerID = id
			}
			rows = append(rows, r)
		}
	}

	playing := map[string]bool{}
	for _, r := range rows {
		if r.PlayerID == "" {
			continue
		}
		if v, ok := r.Stats[pr.key]; ok {
			pr.points[r.PlayerID] = v
		}
		team := r.Team
		if team == "" {
			team = playerTeam[r.PlayerID]
		}
		if team != "" {
			playing[team] = true
		}
	}

	// Bye derivation is only trusted when enough clubs reported. A thin
	// response would otherwise mark most of the league as on bye.
	if len(playing) >= 20 {
		for _, t := range nflTeams {
			if !playing[t] {
				pr.onBye[t] = true
			}
		}
	}
	pr.fetched = len(pr.points) > 0
	p.logf("projections: %d players, %d clubs playing, %d on bye, scoring key %s",
		len(pr.points), len(playing), len(pr.onBye), pr.key)
	return pr
}
