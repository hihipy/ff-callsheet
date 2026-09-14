package sleeper

// The structs below decode only the fields the tool uses. Pointers mark values
// Sleeper leaves null, where absent and zero mean different things.

type league struct {
	Name            string   `json:"name"`
	Season          string   `json:"season"`
	Status          string   `json:"status"`
	TotalRosters    int      `json:"total_rosters"`
	RosterPositions []string `json:"roster_positions"`
	Settings        struct {
		WaiverBudget     int `json:"waiver_budget"`
		WaiverType       int `json:"waiver_type"`
		PlayoffWeekStart int `json:"playoff_week_start"`
	} `json:"settings"`
	ScoringSettings map[string]float64 `json:"scoring_settings"`
}

type roster struct {
	RosterID int      `json:"roster_id"`
	OwnerID  string   `json:"owner_id"`
	Players  []string `json:"players"`
	Starters []string `json:"starters"`
	Reserve  []string `json:"reserve"`
	Taxi     []string `json:"taxi"`
	Settings struct {
		Wins             int `json:"wins"`
		Losses           int `json:"losses"`
		Ties             int `json:"ties"`
		WaiverBudgetUsed int `json:"waiver_budget_used"`
		WaiverPosition   int `json:"waiver_position"`
	} `json:"settings"`
}

type user struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Metadata    struct {
		TeamName string `json:"team_name"`
	} `json:"metadata"`
}

type state struct {
	Week       int    `json:"week"`
	SeasonType string `json:"season_type"`
	Season     string `json:"season"`
}

type player struct {
	PlayerID         string   `json:"player_id"`
	FirstName        string   `json:"first_name"`
	LastName         string   `json:"last_name"`
	Position         string   `json:"position"`
	FantasyPositions []string `json:"fantasy_positions"`
	Team             string   `json:"team"`
	Injury           string   `json:"injury_status"`

	// SearchRank is null for players Sleeper does not rank. DepthChartOrder is
	// null for anyone no club currently lists, which is the only reliable
	// signal that a player is still playing: active and status both read
	// Active for players who retired years ago.
	SearchRank      *int   `json:"search_rank"`
	DepthChartOrder *int   `json:"depth_chart_order"`
	NewsUpdated     *int64 `json:"news_updated"`
}
