package config

import "testing"

func TestResolveForms(t *testing.T) {
	c := Config{Leagues: map[string]Entry{
		"seam": {Platform: "sleeper", ID: "1399955922058547200"},
		"yaho": {Platform: "yahoo", Sport: "nfl", ID: "nfl.l.123456"},
	}}
	cases := []struct{ arg, platform, id string }{
		{"seam", "sleeper", "1399955922058547200"},
		{"yaho", "yahoo", "nfl.l.123456"},
		{"1389704249549529088", "sleeper", "1389704249549529088"},
		{"yahoo:nfl.l.999", "yahoo", "nfl.l.999"},
	}
	for _, c2 := range cases {
		got, err := c.Resolve(c2.arg)
		if err != nil {
			t.Fatalf("%s: %v", c2.arg, err)
		}
		if got.Platform != c2.platform || got.ID != c2.id {
			t.Errorf("%s: got %+v", c2.arg, got)
		}
	}
	if _, err := c.Resolve("sleeper:"); err == nil {
		t.Error("an empty id should be an error")
	}
}

func TestAllIsSorted(t *testing.T) {
	c := Config{Leagues: map[string]Entry{
		"zulu": {ID: "3"}, "alpha": {ID: "1"}, "mike": {ID: "2"},
	}}
	got := c.All()
	want := []string{"1", "2", "3"}
	for i, r := range got {
		if r.ID != want[i] {
			t.Errorf("position %d: got %s want %s", i, r.ID, want[i])
		}
		if r.Platform != DefaultPlatform || r.Sport != DefaultSport {
			t.Errorf("defaults not applied: %+v", r)
		}
	}
}
