package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PrefFrontPageHeroes names the four metrics the dashboard shows as hero
// numbers. Stored as a JSON array of metric names.
const PrefFrontPageHeroes = "front_page_heroes"

// DefaultFrontPageHeroes is used until the user picks their own. Readiness,
// sleep, HRV and resting heart rate answer "how am I doing today" without
// needing a chart.
var DefaultFrontPageHeroes = []string{
	"oura_readiness_score",
	"sleep_analysis",
	"heart_rate_variability",
	"resting_heart_rate",
}

// GetPreference reads one preference into dest. Returns false when the user has
// not set it, leaving dest untouched.
func (db *DB) GetPreference(ctx context.Context, userID int, key string, dest any) (bool, error) {
	var raw []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT value FROM user_preferences WHERE user_id = $1 AND key = $2`,
		userID, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading preference %s: %w", key, err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, fmt.Errorf("decoding preference %s: %w", key, err)
	}
	return true, nil
}

// SetPreference writes one preference, replacing any previous value.
func (db *DB) SetPreference(ctx context.Context, userID int, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encoding preference %s: %w", key, err)
	}
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO user_preferences (user_id, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = EXCLUDED.value`,
		userID, key, raw)
	if err != nil {
		return fmt.Errorf("saving preference %s: %w", key, err)
	}
	return nil
}

// GetFrontPageHeroes returns the user's hero metric selection, falling back to
// the default set.
func (db *DB) GetFrontPageHeroes(ctx context.Context, userID int) ([]string, error) {
	var heroes []string
	found, err := db.GetPreference(ctx, userID, PrefFrontPageHeroes, &heroes)
	if err != nil {
		return nil, err
	}
	if !found || len(heroes) == 0 {
		return DefaultFrontPageHeroes, nil
	}
	return heroes, nil
}
