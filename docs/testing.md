# Testing

Everything runs offline. No test touches Sleeper, so the suite is fast and
works on a plane.

```bash
go test ./...            # everything
go test -race ./...      # what CI runs
go test -cover ./...     # coverage per package
```

## What Is Covered

**The provider, end to end.** `internal/provider/sleeper/fetch_test.go` starts
a local HTTP server that answers on the same paths Sleeper uses, backed by the
fixtures in `testdata/`. `Fetch` runs its whole path against it: six requests,
the roster diff, the projection merge, and the cache. The provider's hosts are
fields rather than constants for exactly this reason.

**The failure paths.** A projections call returning 500 must leave the document
rendering without points, and a rosters call returning 500 must fail the run
loudly. Both are asserted, because the difference between optional and required
data is easy to get backwards later.

**The cache.** Two consecutive fetches are counted: the second makes one fewer
request, because the 5MB player map is read from disk.

**The rendered output, byte for byte.** `internal/render/golden_test.go`
compares against `testdata/golden.md`. Regenerate deliberately:

```bash
go test ./internal/render -update
```

Read the diff before committing it. A golden test only earns its place if
changing it is a decision rather than a reflex.

**The rules that took three attempts to get right.** The availability tests use
real records pulled from Sleeper's player map, including a quarterback who
retired after the 2021 season and whose record still reads `active: true` and
`status: "Active"`. Anyone who reaches for those fields again will fail that
test.

## Fixtures

`internal/provider/sleeper/testdata/` holds one small file per endpoint. They
are hand-built rather than captured wholesale, so each one is small enough to
read, and each player exists to cover a case: a rostered starter, an empty
starting slot, a taxi squad player, a fullback eligible at running back, an
injured stash with no projection, a practice squad body with no depth chart
slot, a retired player who must be excluded, and a lineman the league does not
field.

When you add a case, add the player to `players.json` and assert on it. Keep
the file readable: a 5MB capture would make the suite slower and no test
clearer.

## Adding a Provider

Copy the shape of the Sleeper tests. Point the new provider at a local server,
use fixtures taken from real responses, and assert on the neutral `League` it
returns rather than on its internal types. The renderer tests already cover
what happens downstream, so a provider only has to prove its own mapping.
