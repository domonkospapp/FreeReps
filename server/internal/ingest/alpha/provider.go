package alpha

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/claude/freereps/internal/ingest"
	"github.com/claude/freereps/internal/models"
	"github.com/claude/freereps/internal/storage"
)

// SourceName identifies rows in workout_sets that came from an Alpha Progression
// CSV export. It is part of the table's natural key.
const SourceName = "Alpha Progression"

// Provider processes Alpha Progression CSV exports.
type Provider struct {
	db  *storage.DB
	log *slog.Logger
}

// NewProvider creates a new Alpha Progression ingest provider.
func NewProvider(db *storage.DB, log *slog.Logger) *Provider {
	return &Provider{db: db, log: log}
}

// Ingest parses a CSV export and stores the workout set data.
func (p *Provider) Ingest(ctx context.Context, r io.Reader, userID int) (*ingest.Result, error) {
	sessions, err := Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	result := &ingest.Result{}
	var allRows []models.WorkoutSetRow

	for _, s := range sessions {
		for _, ex := range s.Exercises {
			for _, set := range ex.Sets {
				// Alpha always reports RIR, using -1 for an unrated set; the
				// column is nullable because Hevy leaves it empty.
				rir := set.RIR
				allRows = append(allRows, models.WorkoutSetRow{
					UserID:           userID,
					Source:           SourceName,
					SessionName:      s.Name,
					SessionDate:      s.Date,
					SessionDuration:  s.Duration,
					ExerciseNumber:   ex.Number,
					ExerciseName:     ex.Name,
					Equipment:        ex.Equipment,
					TargetReps:       ex.TargetReps,
					IsWarmup:         set.IsWarmup,
					SetNumber:        set.Number,
					WeightKg:         set.WeightKg,
					IsBodyweightPlus: set.IsBodyweightPlus,
					Reps:             set.Reps,
					RIR:              &rir,
				})
			}
		}
	}

	if len(allRows) > 0 {
		inserted, err := p.db.InsertWorkoutSets(ctx, allRows)
		if err != nil {
			return nil, fmt.Errorf("inserting sets: %w", err)
		}
		result.SetsReceived = len(allRows)
		result.SetsInserted = inserted
	}

	return result, nil
}
