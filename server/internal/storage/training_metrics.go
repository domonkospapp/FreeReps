package storage

import (
	"context"
	"fmt"
	"time"
)

// TrainingTonnageMetric is the metric name under which daily strength training
// volume is written into health_metrics.
//
// It lives there rather than being computed on demand because GetCorrelation
// reads health_metrics exclusively. Without a row in that table, training volume
// cannot be plotted against HRV, sleep or resting heart rate — which is the one
// question this server exists to answer that no training app can.
const TrainingTonnageMetric = "strength_tonnage"

// trainingMetricSource marks rows this server computed rather than received.
const trainingMetricSource = "FreeReps"

// RebuildTrainingMetrics recomputes the daily tonnage series for a date range.
//
// The range is deleted before it is rewritten, so a session corrected in the
// training app corrects the figure instead of adding to it. Both bounds are
// treated as whole days.
func (db *DB) RebuildTrainingMetrics(ctx context.Context, userID int, from, to time.Time) (int64, error) {
	from = from.UTC().Truncate(24 * time.Hour)
	to = to.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)

	if _, err := db.Pool.Exec(ctx,
		`DELETE FROM health_metrics
		 WHERE user_id = $1 AND metric_name = $2 AND source = $3
		   AND time >= $4 AND time < $5`,
		userID, TrainingTonnageMetric, trainingMetricSource, from, to); err != nil {
		return 0, fmt.Errorf("clearing training metrics: %w", err)
	}

	// Noon UTC, the same choice insertSleepAnalysis makes: a fixed within-day
	// timestamp keeps the 5-minute dedup window falling identically across runs.
	tag, err := db.Pool.Exec(ctx,
		`INSERT INTO health_metrics (time, user_id, metric_name, source, units, qty)
		 SELECT date_trunc('day', session_date) + INTERVAL '12 hours',
		        $1, $2, $3, 'kg',
		        SUM(weight_kg * reps)
		 FROM workout_sets
		 WHERE user_id = $1
		   AND NOT is_warmup
		   AND weight_kg > 0
		   AND session_date >= $4 AND session_date < $5
		 GROUP BY 1
		 HAVING SUM(weight_kg * reps) > 0
		 ON CONFLICT DO NOTHING`,
		userID, TrainingTonnageMetric, trainingMetricSource, from, to)
	if err != nil {
		return 0, fmt.Errorf("writing training metrics: %w", err)
	}

	db.InvalidateAvailableMetrics(userID)
	return tag.RowsAffected(), nil
}

// TrainingMetricsRange returns the first and last day with strength data, for a
// full rebuild. Returns ok=false when the user has no sets at all.
func (db *DB) TrainingMetricsRange(ctx context.Context, userID int) (from, to time.Time, ok bool, err error) {
	var minT, maxT *time.Time
	if err = db.Pool.QueryRow(ctx,
		`SELECT MIN(session_date), MAX(session_date) FROM workout_sets WHERE user_id = $1`,
		userID).Scan(&minT, &maxT); err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("reading training range: %w", err)
	}
	if minT == nil || maxT == nil {
		return time.Time{}, time.Time{}, false, nil
	}
	return *minT, *maxT, true, nil
}
