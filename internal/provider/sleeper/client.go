// Package sleeper implements the callsheet Provider against Sleeper's public
// read-only API. Everything Sleeper-specific lives here: its endpoints, its
// player map, and its quirks around who counts as a free agent.
package sleeper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// apiBase is Sleeper's documented read-only API.
	apiBase = "https://api.sleeper.app/v1"

	// projBase hosts projections, which are not in Sleeper's published docs.
	// Every use of it degrades to nothing rather than failing a run.
	projBase = "https://api.sleeper.com"

	// PlayerCacheTTL matches Sleeper's guidance to fetch the player map at
	// most once a day. It is roughly 5MB.
	PlayerCacheTTL = 24 * time.Hour
)

type httpClient struct {
	c *http.Client
}

func newHTTPClient() httpClient {
	return httpClient{c: &http.Client{Timeout: 60 * time.Second}}
}

// endpoints names the two hosts this provider reads. They are fields rather
// than constants so a test can point the whole provider at a local server and
// exercise Fetch for real, rather than only its pure parts.
type endpoints struct {
	api  string
	proj string
}

func defaultEndpoints() endpoints {
	return endpoints{api: apiBase, proj: projBase}
}

func (h httpClient) getJSON(ctx context.Context, url string, into any) error {
	body, err := h.getBytes(ctx, url)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, into)
}

func (h httpClient) getBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
