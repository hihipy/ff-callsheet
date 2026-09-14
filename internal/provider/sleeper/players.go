package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// loadPlayers reads the cached player map, refetching only when it is missing
// or stale. The cache lives with other throwaway data rather than with config,
// because losing it costs a refetch and nothing else.
func (p Provider) loadPlayers(ctx context.Context, sport string) (map[string]player, error) {
	path, err := cachePath(sport)
	if err != nil {
		return nil, err
	}

	if fi, err := os.Stat(path); err == nil && time.Since(fi.ModTime()) < PlayerCacheTTL {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var m map[string]player
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	}

	p.logf("fetching the %s player map, roughly 5MB, once a day", sport)
	raw, err := p.http.getBytes(ctx, fmt.Sprintf("%s/players/%s", apiBase, sport))
	if err != nil {
		return nil, err
	}
	var m map[string]player
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// A failed cache write costs a refetch, not a run.
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		p.logf("warning: could not cache players: %v", err)
	}
	return m, nil
}

func cachePath(sport string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "callsheet", "players-"+sport+".json"), nil
}
