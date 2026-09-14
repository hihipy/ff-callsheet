package sleeper

import (
	"strings"
	"time"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

// DefaultNewsWindow keeps a player who holds no depth chart slot but whose news
// is recent, which covers someone signed too lately to be listed. It is narrow
// on purpose: at thirty days it admitted 186 players with no slot at all, which
// is the practice squad rather than the pool.
const DefaultNewsWindow = 7 * 24 * time.Hour

func (p player) name() string {
	n := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if n == "" {
		return "(unnamed)"
	}
	return n
}

func (p player) rank() int {
	if p.SearchRank == nil {
		return callsheet.UnrankedRank
	}
	return *p.SearchRank
}

func (p player) onDepthChart() bool { return p.DepthChartOrder != nil }

func (p player) newsFresh(now time.Time, window time.Duration) bool {
	if p.NewsUpdated == nil {
		return false
	}
	return now.Sub(time.UnixMilli(*p.NewsUpdated)) < window
}

// available decides whether an unrostered player belongs in the pool. Sleeper's
// active and status fields cannot answer this: a quarterback who retired after
// the 2021 season still reads active true and status Active. What separates him
// from a real free agent is that no club lists him and his last news is years
// old.
func available(p player, now time.Time, window time.Duration) bool {
	if p.Team == "" {
		return false
	}
	// A team defense is an entity rather than a person, so it has no depth
	// chart row and no news of its own.
	if p.Position == "DEF" {
		return true
	}
	return p.onDepthChart() || p.newsFresh(now, window)
}

// toPlayer flattens a Sleeper record into the neutral type, resolving every
// optional field so nothing downstream handles pointers.
func (p player) toPlayer() callsheet.Player {
	eligible := p.FantasyPositions
	if len(eligible) == 0 {
		eligible = []string{p.Position}
	}
	return callsheet.Player{
		ID:       p.PlayerID,
		Name:     p.name(),
		Position: p.Position,
		Eligible: eligible,
		Team:     p.Team,
		Injury:   p.Injury,
		Rank:     p.rank(),
		Slotted:  p.onDepthChart(),
	}
}
