// Package config reads the optional profile file. The file is a convenience:
// every league can be named on the command line instead, so a fresh clone
// works with no setup.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
)

// DefaultPlatform is assumed when a reference carries no platform prefix, so
// today's bare league IDs keep working once other providers exist.
const DefaultPlatform = "sleeper"

// DefaultSport is assumed when a profile names none.
const DefaultSport = "nfl"

// Config is the whole file. Credentials never live here: this file is the one
// people paste into an issue when asking for help.
type Config struct {
	Leagues  map[string]Entry `json:"leagues"`
	Defaults Defaults         `json:"defaults"`
}

// Entry is one saved league under a short alias.
type Entry struct {
	Platform string `json:"platform"`
	Sport    string `json:"sport,omitempty"`
	ID       string `json:"id"`
}

// Defaults are applied when the matching flag is not given.
type Defaults struct {
	Out string `json:"out,omitempty"`
	Top int    `json:"top,omitempty"`
}

// Path returns where the profile file lives on this platform.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "callsheet", "config.json"), nil
}

// Load reads the profile file. A missing file is not an error, because the
// tool is fully usable without one.
func Load() (Config, error) {
	c := Config{Leagues: map[string]Entry{}}
	path, err := Path()
	if err != nil {
		return c, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if c.Leagues == nil {
		c.Leagues = map[string]Entry{}
	}
	return c, nil
}

// Resolve turns one command line argument into a league reference. An alias in
// the config wins; anything else is read as a reference directly, so a stranger
// who cloned the repo needs no config at all.
//
// Accepted forms: "seam" (an alias), "1399955922058547200" (an ID on the
// default platform), "sleeper:1399955922058547200" (an explicit platform).
func (c Config) Resolve(arg string) (callsheet.Ref, error) {
	if e, ok := c.Leagues[arg]; ok {
		return e.ref(), nil
	}
	platform, id := DefaultPlatform, arg
	if p, rest, found := strings.Cut(arg, ":"); found {
		platform, id = p, rest
	}
	if id == "" {
		return callsheet.Ref{}, fmt.Errorf("%q names no league", arg)
	}
	return callsheet.Ref{Platform: platform, Sport: DefaultSport, ID: id}, nil
}

// All returns every saved league, sorted by alias, which is what a run with no
// arguments uses.
func (c Config) All() []callsheet.Ref {
	names := make([]string, 0, len(c.Leagues))
	for name := range c.Leagues {
		names = append(names, name)
	}
	sortStrings(names)
	out := make([]callsheet.Ref, 0, len(names))
	for _, name := range names {
		out = append(out, c.Leagues[name].ref())
	}
	return out
}

func (e Entry) ref() callsheet.Ref {
	platform := e.Platform
	if platform == "" {
		platform = DefaultPlatform
	}
	sport := e.Sport
	if sport == "" {
		sport = DefaultSport
	}
	return callsheet.Ref{Platform: platform, Sport: sport, ID: e.ID}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
