# Adding a Provider

A provider is one platform: Sleeper, Yahoo, ESPN, or anything else that knows
what a fantasy league is. Adding one touches a new folder and one line of
wiring. Nothing in `internal/render` or `internal/callsheet` changes.

## What a Provider Owes

```go
type Provider interface {
	Name() string
	Fetch(ctx context.Context, ref Ref, opts Options) (League, error)
}
```

One call returns the whole league. The provider owns every quirk of its own
API, including the hardest one: deciding who counts as an available player.
Sleeper has no free agent endpoint, so its provider loads the full player map
and subtracts every rostered ID. Yahoo answers that question directly with an
ownership filter. Neither fact escapes its own package.

## Steps

1. Create `internal/provider/<name>/`.
2. Decode the platform's own shapes into unexported types in that package. Keep
   optional values as pointers there, and resolve them when you flatten.
3. Flatten into `callsheet.Player`, `callsheet.Team`, and `callsheet.League`.
   Resolve every optional value: the renderer never handles a nil.
4. Register the provider in `cmd/callsheet/main.go`, in the `providers` map.
5. Write tests against real records you pulled from the platform, not invented
   ones. The Sleeper tests do this, which is why they caught a retired player
   whose status field still read Active.

## Slot Names Are This Project's Vocabulary

`League.Slots` does not carry the platform's slot names. It carries the ones
`internal/callsheet/positions.go` defines, and translating is the provider's
job. Sleeper happens to use the same strings, which is a coincidence of it
being written first rather than a contract.

Yahoo, for example, writes its flex as `W/R/T` and its superflex as
`Q/W/R/T`. A Yahoo provider maps those to `FLEX` and `SUPER_FLEX` before
returning. An unrecognized slot name is treated as a position, so a
mistranslation surfaces as an odd heading in the output rather than an empty
free agent list.

## What to Get Right

**Rank and points are different things.** `Rank` is preseason ordering where
lower is better, and `callsheet.UnrankedRank` when the platform supplies none.
`Points` is this week, and `HasPoints` says whether it exists at all. An
injured stash has no points and still deserves a rank.

**`Slotted` means a club currently lists the player.** It separates rotation
players from practice squad. If your platform has no equivalent signal, leave
it false for everyone: the renderer then omits the split entirely rather than
reporting that nobody is on a depth chart.

**`Options` holds only what every provider understands.** `StaleWindow` bounds
how old a freshness signal may be, and a provider without one ignores it. A
knob meaningful to a single platform belongs in that provider's own
construction, not in the shared struct.

**`Eligible` can differ from `Position`.** A fullback is listed at FB and
started at RB. Position filtering happens in `internal/callsheet` against the
league's own slots, so populate `Eligible` and let it decide.

**Optional data degrades, it does not fail.** Sleeper's projections live on an
undocumented host. When that call fails the run still produces a file, ordered
by rank instead, and says so in the header. Anything optional you add should
behave the same way.

## Sports Other Than Football

`Ref.Sport` carries through to the provider, and Sleeper's endpoints already
take a sport path segment. What is football-specific is position handling:
`FlexSlots` and `PreferredOrder` in `internal/callsheet/positions.go` describe
an NFL lineup. A basketball league would need its own entries there. The rest
of the pipeline does not care.
