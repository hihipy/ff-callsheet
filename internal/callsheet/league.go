// Package callsheet holds the platform-neutral view of a fantasy league and
// the position logic that applies to every platform. Nothing here knows about
// Sleeper, Yahoo, or any other provider.
package callsheet

import (
	"context"
	"time"
)

// UnrankedRank sorts a player with no ranking behind every ranked one.
const UnrankedRank = 1 << 30

// Ref names one league on one platform. The platform prefix lets a single run
// mix providers once more than one exists.
type Ref struct {
	Platform string // sleeper, yahoo
	Sport    string // nfl
	ID       string
}

// Provider fetches one league and returns it in neutral form. A provider owns
// every quirk of its own API, including how it decides who is a free agent,
// so the renderer never learns which platform it is reading.
type Provider interface {
	Name() string
	Fetch(ctx context.Context, ref Ref, opts Options) (League, error)
}

// Options carry what a fetch needs from the caller. Anything here applies to
// every provider. A knob that means something to one platform and nothing to
// another belongs in that provider's own construction, not in this struct.
type Options struct {
	// StaleWindow bounds how old a platform's freshness signal may be before a
	// player is treated as no longer active. Providers that have no such signal
	// ignore it. Zero means the provider's own default.
	StaleWindow time.Duration
	// SkipProjections turns off any optional projection lookup.
	SkipProjections bool
}

// League is the whole document's input.
type League struct {
	Ref        Ref
	Name       string
	Season     string
	SeasonType string // pre, regular, post
	Week       int
	TeamCount  int
	// Slots uses this package's slot vocabulary, not the platform's. See the
	// note above FlexSlots in positions.go.
	Slots []string
	// Scoring is the platform's own settings, passed through unchanged. It is
	// read only for display, and a provider whose keys differ should map the
	// handful in render.scoringKeys rather than inventing a schema.
	Scoring   map[string]float64
	Waivers   Waivers
	FetchedAt time.Time

	Teams []Team
	// Pool is every unrostered player the provider considers currently
	// available. Position filtering happens in this package, not the provider.
	Pool []Player

	// Projected reports whether points are present. PointsKey names the scoring
	// basis in the platform's own words and is shown to the reader as-is, so a
	// provider may leave it empty rather than inventing a label.
	Projected bool
	PointsKey string
	// PointsApproximate says the basis is the nearest the platform publishes
	// rather than the league's actual scoring. It reaches the document, because
	// the log does not: a reader pasting this into an assistant would otherwise
	// see an exact-sounding claim, and -quiet removes the warning entirely.
	PointsApproximate bool
}

// Waivers describes how a league acquires players. Budget is zero in a league
// that uses priority order instead.
type Waivers struct {
	Budget int
	Type   int
}

// Team is one manager's roster.
type Team struct {
	ID             string
	Manager        string
	Wins           int
	Losses         int
	Ties           int
	FAABUsed       int
	WaiverPosition int

	Starters []Player
	Bench    []Player
	Reserve  []Player
	Taxi     []Player
}

// Player is one person or team defense, flattened out of whatever shape the
// provider used. Optional values are resolved here so the renderer never deals
// in pointers.
type Player struct {
	ID       string
	Name     string
	Position string
	// Eligible lists every position the player may be started at, which can
	// differ from Position. A fullback is eligible at running back.
	Eligible []string
	Team     string
	Injury   string
	// Rank is a preseason ordering where lower is better, UnrankedRank when
	// the provider supplies none.
	Rank int

	Points    float64
	HasPoints bool
	OnBye     bool
	// Slotted reports that a club currently lists the player, which separates
	// rotation players from practice squad. A provider whose platform exposes
	// no such signal leaves this false for everyone, and the renderer omits the
	// split rather than reporting a wrong one.
	Slotted bool
}

// InSeason reports whether a run should produce anything at all. It is the
// stop condition for an unattended weekly job.
func (l League) InSeason() bool {
	return l.SeasonType == "regular" || l.SeasonType == "post"
}
