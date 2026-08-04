package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// HevyCredentials holds a user's Hevy API key and the ingest cutoff.
// Hevy issues one account-wide key with no scoping; it grants read and write
// access to every endpoint.
type HevyCredentials struct {
	UserID    int
	APIKey    string
	SyncFrom  time.Time
	UpdatedAt time.Time
}

// HevySyncState tracks the timestamp passed as `since` to the next
// GET /v1/workouts/events call.
type HevySyncState struct {
	UserID      int
	LastEventAt time.Time
	UpdatedAt   time.Time
}

// UpsertHevyCredentials stores or replaces a user's Hevy API key and cutoff date.
func (db *DB) UpsertHevyCredentials(ctx context.Context, userID int, apiKey string, syncFrom time.Time) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO hevy_credentials (user_id, api_key, sync_from, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   api_key = EXCLUDED.api_key,
		   sync_from = EXCLUDED.sync_from,
		   updated_at = NOW()`,
		userID, apiKey, syncFrom)
	if err != nil {
		return fmt.Errorf("upserting hevy credentials: %w", err)
	}
	return nil
}

// GetHevyCredentials returns a user's credentials, or (nil, nil) if none are stored.
func (db *DB) GetHevyCredentials(ctx context.Context, userID int) (*HevyCredentials, error) {
	var c HevyCredentials
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, api_key, sync_from, updated_at
		 FROM hevy_credentials WHERE user_id = $1`, userID).
		Scan(&c.UserID, &c.APIKey, &c.SyncFrom, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting hevy credentials: %w", err)
	}
	return &c, nil
}

// DeleteHevyCredentials removes a user's credentials and their sync state.
func (db *DB) DeleteHevyCredentials(ctx context.Context, userID int) error {
	if _, err := db.Pool.Exec(ctx, `DELETE FROM hevy_sync_state WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting hevy sync state: %w", err)
	}
	if _, err := db.Pool.Exec(ctx, `DELETE FROM hevy_credentials WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting hevy credentials: %w", err)
	}
	return nil
}

// ListHevyCredentialUsers returns the IDs of all users with a stored API key.
func (db *DB) ListHevyCredentialUsers(ctx context.Context) ([]int, error) {
	rows, err := db.Pool.Query(ctx, `SELECT user_id FROM hevy_credentials ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("listing hevy credential users: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning hevy credential user: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetHevySyncState returns a user's sync state, or (nil, nil) if the user has
// never synced — in which case the caller fetches without a `since` bound.
func (db *DB) GetHevySyncState(ctx context.Context, userID int) (*HevySyncState, error) {
	var s HevySyncState
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, last_event_at, updated_at
		 FROM hevy_sync_state WHERE user_id = $1`, userID).
		Scan(&s.UserID, &s.LastEventAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting hevy sync state: %w", err)
	}
	return &s, nil
}

// UpsertHevySyncState records the point the next sync resumes from.
func (db *DB) UpsertHevySyncState(ctx context.Context, userID int, lastEventAt time.Time) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO hevy_sync_state (user_id, last_event_at, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   last_event_at = EXCLUDED.last_event_at,
		   updated_at = NOW()`,
		userID, lastEventAt)
	if err != nil {
		return fmt.Errorf("upserting hevy sync state: %w", err)
	}
	return nil
}
