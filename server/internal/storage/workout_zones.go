package storage

import (
	"context"
	"fmt"
	"time"
)

// ZoneCount is the number of heart rate zones. Zone 5 is the top one and the
// only one drawn in the accent.
const ZoneCount = 5

// zoneBounds are the zone edges as fractions of the user's maximum heart rate.
var zoneBounds = []float64{0.6, 0.7, 0.8, 0.9}

// WorkoutZones holds one session's share of time per zone, as fractions summing
// to 1. Empty when the session has no heart rate samples.
type WorkoutZones struct {
	WorkoutID string    `json:"workout_id"`
	Shares    []float64 `json:"shares"`
}

// GetMaxHeartRate returns the highest heart rate ever recorded for the user,
// which is what the zone bands are computed from. Returns 0 when no workout
// carries heart rate data.
func (db *DB) GetMaxHeartRate(ctx context.Context, userID int) (float64, error) {
	var max *float64
	err := db.Pool.QueryRow(ctx,
		`SELECT MAX(COALESCE(max_bpm, avg_bpm)) FROM workout_heart_rate WHERE user_id = $1`,
		userID).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("querying max heart rate: %w", err)
	}
	if max == nil {
		return 0, nil
	}
	return *max, nil
}

// GetWorkoutZones returns the per-zone share of each workout in the range.
//
// Shares are counted by sample rather than by elapsed time. Samples arrive at a
// near-constant cadence within a session, so the two agree closely, and
// counting avoids a window function over every heart rate row in the range —
// this runs for a whole workout list, not one session.
func (db *DB) GetWorkoutZones(ctx context.Context, userID int, start, end time.Time, maxHR float64) ([]WorkoutZones, error) {
	if maxHR <= 0 {
		return nil, nil
	}

	edges := make([]float64, len(zoneBounds))
	for i, f := range zoneBounds {
		edges[i] = f * maxHR
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT hr.workout_id,
		        WIDTH_BUCKET(COALESCE(hr.avg_bpm, hr.max_bpm), $4::float8[]) AS zone,
		        COUNT(*)
		 FROM workout_heart_rate hr
		 JOIN workouts w ON w.id = hr.workout_id
		 WHERE hr.user_id = $1 AND w.start_time >= $2 AND w.start_time < $3
		   AND COALESCE(hr.avg_bpm, hr.max_bpm) IS NOT NULL
		 GROUP BY hr.workout_id, zone`,
		userID, start, end, edges)
	if err != nil {
		return nil, fmt.Errorf("querying workout zones: %w", err)
	}
	defer rows.Close()

	counts := make(map[string][]float64)
	for rows.Next() {
		var id string
		var zone int
		var n int64
		if err := rows.Scan(&id, &zone, &n); err != nil {
			return nil, fmt.Errorf("scanning workout zone: %w", err)
		}
		if _, ok := counts[id]; !ok {
			counts[id] = make([]float64, ZoneCount)
		}
		if zone >= 0 && zone < ZoneCount {
			counts[id][zone] += float64(n)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]WorkoutZones, 0, len(counts))
	for id, c := range counts {
		var total float64
		for _, v := range c {
			total += v
		}
		if total == 0 {
			continue
		}
		shares := make([]float64, ZoneCount)
		for i, v := range c {
			shares[i] = v / total
		}
		result = append(result, WorkoutZones{WorkoutID: id, Shares: shares})
	}
	return result, nil
}
