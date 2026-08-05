package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// WithingsToken holds per-user Withings application credentials and OAuth2 tokens.
// ClientID/ClientSecret are the user's app credentials from the Withings Partner Hub.
// AccessToken/RefreshToken are populated after OAuth2 authorization.
type WithingsToken struct {
	UserID         int
	ClientID       string
	ClientSecret   string
	AccessToken    string
	RefreshToken   string
	TokenType      string
	ExpiresAt      time.Time
	WithingsUserID string
	UpdatedAt      time.Time
}

// WithingsSyncState tracks the server-reported update timestamp of the last sync
// for a data type. Unix seconds, because the API filters to the second.
type WithingsSyncState struct {
	UserID     int
	DataType   string
	LastUpdate int64
	UpdatedAt  time.Time
}

// UpsertWithingsCredentials stores or updates a user's Withings app credentials
// (before OAuth2 authorization). Does not overwrite existing tokens.
func (db *DB) UpsertWithingsCredentials(ctx context.Context, userID int, clientID, clientSecret string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO withings_tokens (user_id, client_id, client_secret, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   client_id = EXCLUDED.client_id,
		   client_secret = EXCLUDED.client_secret,
		   updated_at = NOW()`,
		userID, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("upserting withings credentials: %w", err)
	}
	return nil
}

// UpsertWithingsToken stores or updates OAuth2 tokens for a user, preserving
// client_id/client_secret.
//
// This is an INSERT ... ON CONFLICT rather than the plain UPDATE the Oura
// equivalent uses. Withings rotates the refresh token on every refresh and the
// previous one stops working within hours, so an UPDATE that matches no row
// would discard the only usable token and the connection would be lost at the
// next cycle with nothing in the logs.
func (db *DB) UpsertWithingsToken(ctx context.Context, tok WithingsToken) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO withings_tokens
		   (user_id, access_token, refresh_token, token_type, expires_at, withings_user_id, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   access_token = EXCLUDED.access_token,
		   refresh_token = EXCLUDED.refresh_token,
		   token_type = EXCLUDED.token_type,
		   expires_at = EXCLUDED.expires_at,
		   withings_user_id = EXCLUDED.withings_user_id,
		   updated_at = NOW()`,
		tok.UserID, tok.AccessToken, tok.RefreshToken, tok.TokenType, tok.ExpiresAt, tok.WithingsUserID)
	if err != nil {
		return fmt.Errorf("upserting withings token: %w", err)
	}
	return nil
}

// GetWithingsToken retrieves the Withings credentials and tokens for a user.
// Returns nil if not found.
func (db *DB) GetWithingsToken(ctx context.Context, userID int) (*WithingsToken, error) {
	var tok WithingsToken
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, client_id, client_secret, access_token, refresh_token,
		        token_type, expires_at, withings_user_id, updated_at
		 FROM withings_tokens WHERE user_id = $1`, userID).
		Scan(&tok.UserID, &tok.ClientID, &tok.ClientSecret, &tok.AccessToken, &tok.RefreshToken,
			&tok.TokenType, &tok.ExpiresAt, &tok.WithingsUserID, &tok.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting withings token: %w", err)
	}
	return &tok, nil
}

// DeleteWithingsToken removes the credentials and tokens for a user.
func (db *DB) DeleteWithingsToken(ctx context.Context, userID int) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM withings_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("deleting withings token: %w", err)
	}
	return nil
}

// ListWithingsAuthorizedUsers returns the user IDs that have completed OAuth2
// authorization. Rows holding credentials but no access token are excluded: the
// sync loop cannot do anything with them, and including them would write an
// error row to import_logs on every cycle.
func (db *DB) ListWithingsAuthorizedUsers(ctx context.Context) ([]int, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id FROM withings_tokens WHERE access_token <> '' ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("listing withings token users: %w", err)
	}
	defer rows.Close()

	var users []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scanning withings token user: %w", err)
		}
		users = append(users, uid)
	}
	return users, rows.Err()
}

// GetWithingsSyncState retrieves the last update timestamp for a data type.
// Returns nil if no state exists (first sync).
func (db *DB) GetWithingsSyncState(ctx context.Context, userID int, dataType string) (*WithingsSyncState, error) {
	var s WithingsSyncState
	err := db.Pool.QueryRow(ctx,
		`SELECT user_id, data_type, last_update, updated_at
		 FROM withings_sync_state WHERE user_id = $1 AND data_type = $2`,
		userID, dataType).
		Scan(&s.UserID, &s.DataType, &s.LastUpdate, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting withings sync state: %w", err)
	}
	return &s, nil
}

// UpsertWithingsSyncState updates the last update timestamp for a data type.
func (db *DB) UpsertWithingsSyncState(ctx context.Context, userID int, dataType string, lastUpdate int64) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO withings_sync_state (user_id, data_type, last_update, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id, data_type) DO UPDATE SET
		   last_update = EXCLUDED.last_update,
		   updated_at = NOW()`,
		userID, dataType, lastUpdate)
	if err != nil {
		return fmt.Errorf("upserting withings sync state: %w", err)
	}
	return nil
}

// DeleteWithingsSyncStates removes all sync state for a user (used on disconnect).
func (db *DB) DeleteWithingsSyncStates(ctx context.Context, userID int) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM withings_sync_state WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("deleting withings sync states: %w", err)
	}
	return nil
}

// ListWithingsSyncStates retrieves all sync states for a user.
func (db *DB) ListWithingsSyncStates(ctx context.Context, userID int) ([]WithingsSyncState, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id, data_type, last_update, updated_at
		 FROM withings_sync_state WHERE user_id = $1 ORDER BY data_type`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("listing withings sync states: %w", err)
	}
	defer rows.Close()

	var states []WithingsSyncState
	for rows.Next() {
		var s WithingsSyncState
		if err := rows.Scan(&s.UserID, &s.DataType, &s.LastUpdate, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning withings sync state: %w", err)
		}
		states = append(states, s)
	}
	return states, rows.Err()
}
