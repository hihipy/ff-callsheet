package main

import (
	"testing"

	"github.com/hihipy/ff-callsheet/internal/callsheet"
	"github.com/hihipy/ff-callsheet/internal/config"
)

func TestSlugIsFilesystemSafeAndStable(t *testing.T) {
	cases := []struct {
		name string
		lg   callsheet.League
		want string
	}{
		{"spaces and case", callsheet.League{Name: "The Busted Seam", Week: 1}, "the-busted-seam-week-01"},
		{"punctuation stripped", callsheet.League{Name: "CraccKillas!!! / 2026", Week: 12}, "cracckillas-2026-week-12"},
		{"emoji and accents stripped", callsheet.League{Name: "Liga Ñ 🏈", Week: 3}, "liga-week-03"},
		{"falls back to the id", callsheet.League{Name: "!!!", Week: 9, Ref: callsheet.Ref{ID: "12345"}}, "12345-week-09"},
	}
	for _, c := range cases {
		if got := slug(c.lg); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestResolveAllFallsBackToEverySavedLeague(t *testing.T) {
	cfg := config.Config{Leagues: map[string]config.Entry{
		"a": {ID: "1"}, "b": {Platform: "yahoo", ID: "nfl.l.2"},
	}}
	got, err := resolveAll(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("no arguments should mean every saved league, got %d", len(got))
	}
	named, err := resolveAll(cfg, []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 1 || named[0].Platform != "yahoo" {
		t.Errorf("named league: %+v", named)
	}
	if _, err := resolveAll(cfg, []string{"sleeper:"}); err == nil {
		t.Error("an empty id should fail rather than fetch nothing")
	}
}
