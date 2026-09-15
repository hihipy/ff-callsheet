package callsheet

import "sort"

// The slot names below are this package's own vocabulary, not any platform's.
// A provider maps its platform's names into these before returning a League.
// Sleeper happens to use the same strings, which is a coincidence of it having
// been the first provider written, not a contract. Yahoo writes its flex slots
// as W/R/T and Q/W/R/T, and its provider is responsible for translating them.
//
// A slot name that reaches StartablePositions unrecognized is treated as a
// position, so a mistranslation shows up as a strange heading in the output
// rather than as an empty free agent list.

// FlexSlots expands a lineup slot into the positions it accepts. A slot absent
// from here and from StructuralSlots names a position directly, which is what
// keeps an unfamiliar format working.
var FlexSlots = map[string][]string{
	"FLEX":       {"RB", "WR", "TE"},
	"WRRB_FLEX":  {"RB", "WR"},
	"REC_FLEX":   {"WR", "TE"},
	"SUPER_FLEX": {"QB", "RB", "WR", "TE"},
	"IDP_FLEX":   {"DL", "LB", "DB"},
}

// StructuralSlots hold players rather than start them, so they say nothing
// about which positions a league can field.
var StructuralSlots = map[string]bool{"BN": true, "IR": true, "TAXI": true}

// PreferredOrder is the order a reader expects when these positions appear.
// Anything else a league uses still renders, sorted after them.
var PreferredOrder = []string{"QB", "RB", "WR", "TE", "K", "DEF", "DL", "LB", "DB"}

// StartablePositions reads the league's own slots rather than assuming a
// lineup. A league with no kicker slot never sees a kicker, and an IDP league
// sees its defenders.
func StartablePositions(slots []string) map[string]bool {
	out := make(map[string]bool)
	for _, s := range slots {
		if StructuralSlots[s] {
			continue
		}
		if expanded, ok := FlexSlots[s]; ok {
			for _, p := range expanded {
				out[p] = true
			}
			continue
		}
		// Any remaining slot names a position directly. An IDP league's LB
		// slot lands here, and so does a format nobody has written down yet.
		out[s] = true
	}
	return out
}

// OrderedPositions puts a startable set into a stable order, so two runs of
// the same league diff cleanly.
func OrderedPositions(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	seen := make(map[string]bool, len(set))
	for _, p := range PreferredOrder {
		if set[p] {
			out = append(out, p)
			seen[p] = true
		}
	}
	extra := make([]string, 0, len(set))
	for p := range set {
		if !seen[p] {
			extra = append(extra, p)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// Bucket returns the position a player is listed under, preferring the primary
// position and falling back to any eligibility the league fields.
func (p Player) Bucket(startable map[string]bool) (string, bool) {
	if startable[p.Position] {
		return p.Position, true
	}
	for _, e := range p.Eligible {
		if startable[e] {
			return e, true
		}
	}
	return "", false
}

// PoolByPosition groups the available pool into the league's own positions,
// ordered so the best option in each bucket is first. Projected points beat
// preseason rank wherever they exist, because rank is from August.
func (l League) PoolByPosition() (map[string][]Player, []string) {
	startable := StartablePositions(l.Slots)
	out := make(map[string][]Player)
	for _, p := range l.Pool {
		if pos, ok := p.Bucket(startable); ok {
			out[pos] = append(out[pos], p)
		}
	}
	for pos := range out {
		list := out[pos]
		sort.Slice(list, func(i, j int) bool {
			if list[i].HasPoints != list[j].HasPoints {
				return list[i].HasPoints
			}
			if list[i].HasPoints && list[i].Points != list[j].Points {
				return list[i].Points > list[j].Points
			}
			if list[i].Rank != list[j].Rank {
				return list[i].Rank < list[j].Rank
			}
			if list[i].Name != list[j].Name {
				return list[i].Name < list[j].Name
			}
			// The pool is built by ranging a map, and sort.Slice is not
			// stable, so two same-named players tied on every other key would
			// swap between runs. ID is unique and settles it.
			return list[i].ID < list[j].ID
		})
		out[pos] = list
	}
	return out, OrderedPositions(startable)
}

// ByRank orders players by preseason rank alone. Injured players belong here
// rather than in the projection ordering, because an injured stash has no
// projection this week and would otherwise sort last.
func ByRank(list []Player) []Player {
	out := append([]Player(nil), list...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ID < out[j].ID
	})
	return out
}
