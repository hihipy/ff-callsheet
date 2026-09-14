# FF Callsheet

[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)

**Built with**

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![Markdown](https://img.shields.io/badge/Markdown-000000?style=flat&logo=markdown&logoColor=white)](https://commonmark.org)

**Your whole fantasy league in one file, ready to hand to an AI.**

FF Callsheet takes a league ID and writes a single document holding every team's roster, every available free agent, and the league's own rules. You can read it yourself, or paste it into ChatGPT or Claude and ask what to do this week.

---

## What problem this solves

Ask an AI for fantasy advice and it will happily invent things. It does not know who is on your bench, who your rivals are carrying, what your league does with kickers, or how much waiver money anyone has left. You end up typing all of that out, badly, every week.

This writes it out for you. One command produces a file about nine kilobytes long, small enough to paste anywhere, that answers all of those questions at once. What you get back stops being generic advice and starts being about your actual league.

It works on any football league Sleeper supports. Standard, PPR, half PPR, two quarterback, superflex, dynasty with taxi squads, leagues with defensive players, leagues with no kicker. The tool reads your league's own settings rather than assuming a format.

---

## Installing it

### Step 1: install Go

Go is the programming language this is written in. You need it once, to turn the source code into a program.

- **macOS**: `brew install go`, or download the installer from [go.dev/dl](https://go.dev/dl/)
- **Windows**: download the MSI installer from [go.dev/dl](https://go.dev/dl/) and run it
- **Linux**: your package manager, or the tarball from [go.dev/dl](https://go.dev/dl/)

Open a new terminal window afterward and type `go version`. If it prints a version number, you are set.

### Step 2: build the tool

```bash
git clone https://github.com/hihipy/ff-callsheet.git
cd ff-callsheet
go build -o callsheet ./cmd/callsheet
```

That leaves a single file called `callsheet` in the folder. It has no dependencies and needs no installer. Move it anywhere you like, or run it from where it sits.

---

## Using it

You need your league ID, which sits in the address bar when you open your league on Sleeper. In `https://sleeper.com/leagues/1399955922058547200/team` the ID is the long number.

```bash
./callsheet -out league.md 1399955922058547200
```

That writes `league.md`. Open it, or paste the contents into an AI and ask it something:

> Here is my fantasy league. I am 2-3 and my running backs keep getting hurt. Who should I target on waivers, and what should I bid?

The first run downloads a list of every NFL player, which takes a few seconds. After that it is cached for a day and runs are quick.

### Saving your leagues

Typing a twenty digit number gets old. You can save short names for your leagues instead, in a file the tool looks for on its own.

On macOS that file is `~/Library/Application Support/callsheet/config.json`, on Linux `~/.config/callsheet/config.json`, and on Windows `%AppData%\callsheet\config.json`.

```json
{
  "leagues": {
    "seam": { "platform": "sleeper", "id": "1399955922058547200" },
    "bourbon": { "platform": "sleeper", "id": "1389704249549529088" }
  },
  "defaults": { "top": 25 }
}
```

Then:

```bash
./callsheet -out league.md seam
./callsheet -out reports/ seam bourbon
./callsheet -out reports/
```

The last one exports every league you have saved. When you export more than one, `-out` needs to be a folder, and each league gets its own file named after itself and the week.

The config file is optional. Everything works with a bare league ID, so nothing has to be set up before the first run.

### Running it every week

The tool knows when the season is on. Outside the regular season and playoffs it writes nothing and exits quietly, so a scheduled job is safe to leave running all year. Point a weekly cron job or scheduled task at it and it will start producing files again in September on its own.

---

## What the finished file looks like

```markdown
# The Busted Seam PPR League

2026 regular, week 1 | 12 teams | sleeper | fetched 2026-09-14 15:53 EDT

Slots: QB RB WR TE FLEX FLEX SUPER_FLEX DEF BN BN BN BN BN BN BN BN

FAAB budget: $100 | waiver_type 2

Scoring: rec 1 | bonus_rec_te 0.5 | pass_td 4 | pass_int -2

## Rosters

### cstinnett19

0-0-0 | $100 FAAB left of $100

Starters: Joe Burrow QB CIN (20.9) | Chase Brown RB CIN (17.1) | ...
Bench: Jonathon Brooks RB CAR (9.9) | Brock Bowers TE LV (Out) | ...

## Free agents

443 unrostered, 410 of them on a club depth chart. Top 25 healthy per
position, ordered by projected pts_ppr for this week.

### RB (81 available, 76 on a depth chart)

Keaton Mitchell LAC (6.7) | Emari Demercado DAL (6.1) | ...

Injured: James Conner ARI (IR) | Trey Benson ARI (IR) | ...
```

The number in parentheses is this week's projection, in your league's own scoring. `BYE` replaces it when a player's team is off that week. Injured players are listed separately, ordered by how good they were before they got hurt, because a stash is worth knowing about even with no projection.

---

## All the options

| Flag | What it does |
| --- | --- |
| `-out` | Where to write. A filename for one league, a folder for several. Prints to the screen if left out |
| `-top` | How many free agents to list per position. Defaults to 25 |
| `-stale-days` | How many days without news before an unlisted player drops out of the pool. Defaults to 7 |
| `-no-projections` | Skip projections. Free agents fall back to preseason ranking |
| `-any-season` | Run even in the offseason, when the tool would otherwise write nothing |
| `-quiet` | Stop printing progress |

Flags go before the league names, not after.

---

## Things it cannot do

**It cannot make moves for you.** Sleeper's public interface is read only. This can tell you to bid $14 on a running back, and you still place the bid yourself in the app. That is a limit of the platform, not a feature waiting to be written.

**Projections are borrowed.** They come from an endpoint Sleeper publishes but does not document, so it may change without warning. When it does, the file still gets written and says at the top that it fell back to preseason ranking.

**It is Sleeper only today.** Yahoo support is planned, and the code is already split so that a second platform is a new folder rather than a rewrite. If you want to try it yourself, `docs/adding-a-provider.md` explains what is involved.

**Free agent counts include practice squad players.** The tool separates them out and tells you how many of each, but the total is larger than the set of players who realistically matter.

---

## For developers

Written in Go with nothing outside the standard library.

```
cmd/callsheet/          flags and wiring
internal/callsheet/     the platform-neutral league, and position logic
internal/provider/      one folder per platform
internal/render/        markdown output
internal/config/        the saved leagues file
```

Free agency is the interesting part. Sleeper has no endpoint for it, so the tool loads every player it knows about, subtracts everyone on a roster, and filters what remains. That filter is harder than it sounds: Sleeper marks players as active long after they retire, and a quarterback who last played in 2021 will still tell you he is on his old team. What actually separates him from a real free agent is that no club lists him on a depth chart and nothing has been written about him in years.

```bash
go test ./...
go test -race ./...
```

Everything runs offline against fixtures, including the network paths. `docs/testing.md` covers how that works and how to regenerate the golden output file.

---

## License

This project is licensed under [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/).

- **Attribution.** Credit the original work.
- **NonCommercial.** No commercial use.
- **ShareAlike.** Derivatives must use the same license.
