package hevy

import (
	"fmt"
	"time"

	"github.com/claude/freereps/internal/models"
)

// SourceName identifies rows in workout_sets that came from Hevy. It is part of
// the table's natural key and must match the value used in the source priority
// configuration.
const SourceName = "Hevy"

// formatDuration renders a session length in the "H:MM hr" form that Alpha
// Progression writes, so both sources display identically.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d hr", int(d.Hours()), int(d.Minutes())%60)
}

// parseTime parses an ISO 8601 timestamp as delivered by the Hevy API.
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// MapWorkout converts one Hevy workout into workout_sets rows.
//
// Fields Hevy does not carry stay empty: Equipment lives on the exercise
// template rather than on the logged set, and TargetReps belongs to the routine
// — a completed workout records what happened, not what was prescribed.
//
// RPE is stored as delivered. RIR stays empty for Hevy rows; the database
// derives effort_rir from whichever scale the source used, so the conversion
// happens once in the read path instead of at every write.
func MapWorkout(w Workout, userID int) ([]models.WorkoutSetRow, error) {
	start, err := parseTime(w.StartTime)
	if err != nil {
		return nil, fmt.Errorf("parsing start_time of workout %s: %w", w.ID, err)
	}

	var endPtr *time.Time
	var duration string
	if end, err := parseTime(w.EndTime); err == nil {
		endPtr = &end
		duration = formatDuration(end.Sub(start))
	}

	title := w.Title
	if title == "" {
		title = "Workout"
	}

	var rows []models.WorkoutSetRow
	for _, ex := range w.Exercises {
		for _, set := range ex.Sets {
			row := models.WorkoutSetRow{
				UserID:             userID,
				Source:             SourceName,
				ExternalID:         w.ID,
				RoutineID:          w.RoutineID,
				SessionName:        title,
				SessionDate:        start,
				SessionEnd:         endPtr,
				SessionDuration:    duration,
				ExerciseNumber:     ex.Index + 1,
				ExerciseName:       ex.Title,
				ExerciseTemplateID: ex.ExerciseTemplateID,
				ExerciseNotes:      ex.Notes,
				IsWarmup:           set.Type == "warmup",
				SetType:            set.Type,
				SetNumber:          set.Index + 1,
				SupersetID:         ex.SupersetsID,
				RPE:                set.RPE,
				DistanceM:          set.DistanceMeters,
				DurationSec:        set.DurationSeconds,
				CustomMetric:       set.CustomMetric,
			}
			if row.SetType == "" {
				row.SetType = "normal"
			}
			if set.WeightKg != nil {
				row.WeightKg = *set.WeightKg
			}
			if set.Reps != nil {
				row.Reps = *set.Reps
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// WorkoutStart returns the start time of a workout, for the sync_from cutoff
// check that runs before the workout is mapped.
func WorkoutStart(w Workout) (time.Time, error) {
	return parseTime(w.StartTime)
}
