// Command callsheet exports a fantasy league as a single markdown file: every
// team's roster, the free agent pool, and the league's own settings, sized for
// pasting into an AI assistant.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
	"github.com/hihipy/ff-callsheet/internal/config"
	"github.com/hihipy/ff-callsheet/internal/provider/sleeper"
	"github.com/hihipy/ff-callsheet/internal/render"
)

func main() {
	out := flag.String("out", "", "file or directory to write; defaults to standard output")
	top := flag.Int("top", 0, "healthy free agents to list per position (default 25)")
	staleDays := flag.Int("stale-days", 0, "how many days of inactivity before a player leaves the pool, where the platform reports it")
	skipProj := flag.Bool("no-projections", false, "skip the projections lookup")
	quiet := flag.Bool("quiet", false, "suppress progress and diagnostics")
	anySeason := flag.Bool("any-season", false, "run even outside the regular and post season")
	flag.Usage = usage
	flag.Parse()

	if err := run(*out, *top, *staleDays, *skipProj, *quiet, *anySeason, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), `callsheet exports a fantasy league as markdown.

Usage:
  callsheet [flags] [league ...]

A league is an alias from your config file, a bare league ID, or a
platform-prefixed reference such as sleeper:1399955922058547200. With no
league given, every league in the config file is exported.

Flags:
`)
	flag.PrintDefaults()
}

func run(outPath string, top, staleDays int, skipProj, quiet, anySeason bool, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if top == 0 {
		top = cfg.Defaults.Top
	}
	if outPath == "" {
		outPath = cfg.Defaults.Out
	}

	refs, err := resolveAll(cfg, args)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		path, _ := config.Path()
		return fmt.Errorf("no league given and none saved in %s", path)
	}

	var logw *os.File
	if !quiet {
		logw = os.Stderr
	}
	providers := map[string]callsheet.Provider{
		"sleeper": sleeper.New(logw),
	}

	opts := callsheet.Options{SkipProjections: skipProj}
	if staleDays > 0 {
		opts.StaleWindow = time.Duration(staleDays) * 24 * time.Hour
	}

	// One league failing must not lose the others, so failures are collected
	// and reported at the end rather than ending the run.
	var failed []string
	for _, ref := range refs {
		p, ok := providers[ref.Platform]
		if !ok {
			failed = append(failed, fmt.Sprintf("%s: no provider for platform %q", ref.ID, ref.Platform))
			continue
		}
		lg, err := p.Fetch(context.Background(), ref, opts)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", ref.ID, err))
			continue
		}
		if !lg.InSeason() && !anySeason {
			warn(logw, "%s: %s season, nothing to export", lg.Name, lg.SeasonType)
			continue
		}
		if err := write(outPath, lg, render.Options{Top: top}, len(refs) > 1); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", ref.ID, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("%d of %d leagues failed:\n  %s",
			len(failed), len(refs), strings.Join(failed, "\n  "))
	}
	return nil
}

// resolveAll turns the arguments into references, falling back to every saved
// league when none are named.
func resolveAll(cfg config.Config, args []string) ([]callsheet.Ref, error) {
	if len(args) == 0 {
		return cfg.All(), nil
	}
	out := make([]callsheet.Ref, 0, len(args))
	for _, a := range args {
		ref, err := cfg.Resolve(a)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

// write sends one league to stdout, to a named file, or to a file inside a
// directory. Several leagues always get their own files, since two leagues
// rarely belong in one prompt.
func write(outPath string, lg callsheet.League, opts render.Options, many bool) error {
	md := render.Markdown(lg, opts)

	if outPath == "" {
		if many {
			// Concatenating leagues on stdout would silently merge them.
			return fmt.Errorf("several leagues need -out to be a directory")
		}
		_, err := os.Stdout.WriteString(md)
		return err
	}

	path := outPath
	if many || isDir(outPath) {
		if err := os.MkdirAll(outPath, 0o755); err != nil {
			return err
		}
		path = filepath.Join(outPath, slug(lg)+".md")
	}
	// Written from Go rather than redirected by the shell, because Windows
	// PowerShell 5.1 redirection produces UTF-16 with a byte order mark.
	return os.WriteFile(path, []byte(md), 0o644)
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// slug builds a filename that stays stable across runs and sorts by week.
func slug(lg callsheet.League) string {
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		case r == ' ' || r == '-' || r == '_':
			return '-'
		default:
			return -1
		}
	}, lg.Name)
	// Dropped punctuation can leave runs of separators behind, so "A !!! / B"
	// does not become "a----b".
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	if name == "" {
		name = lg.Ref.ID
	}
	return fmt.Sprintf("%s-week-%02d", name, lg.Week)
}

func warn(f *os.File, format string, args ...any) {
	if f == nil {
		return
	}
	fmt.Fprintf(f, format+"\n", args...)
}
