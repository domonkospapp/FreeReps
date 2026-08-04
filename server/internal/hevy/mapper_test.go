package hevy

import (
	"testing"
	"time"
)

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

// RPE is stored on the scale Hevy delivers it. An earlier version converted it
// to RIR while writing, which normalised a source at ingest time — the database
// derives effort_rir from whichever scale is present instead. RIR must stay
// empty on Hevy rows, because a zero there would read as training to failure.
func TestMapWorkoutKeepsRPEUnconverted(t *testing.T) {
	rows, err := MapWorkout(sampleWorkout(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name    string
		row     int
		wantRPE *float64
	}{
		{"unrated warmup carries no RPE", 0, nil},
		{"RPE 9 is stored as 9, not converted to RIR 1", 1, floatPtr(9)},
		{"RPE 10 is stored as 10, not converted to RIR 0", 2, floatPtr(10)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rows[tt.row].RPE
			switch {
			case tt.wantRPE == nil && got != nil:
				t.Errorf("RPE = %v, want nil", *got)
			case tt.wantRPE != nil && got == nil:
				t.Errorf("RPE = nil, want %v", *tt.wantRPE)
			case tt.wantRPE != nil && *got != *tt.wantRPE:
				t.Errorf("RPE = %v, want %v", *got, *tt.wantRPE)
			}
			if rows[tt.row].RIR != nil {
				t.Errorf("RIR = %v, want nil — Hevy rows leave the RIR column empty", *rows[tt.row].RIR)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{75 * time.Minute, "1:15 hr"},
		{62 * time.Minute, "1:02 hr"},
		{45 * time.Minute, "0:45 hr"},
		{0, ""},
		{-time.Minute, ""},
	}
	for _, tt := range tests {
		if got := formatDuration(tt.d); got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func sampleWorkout() Workout {
	return Workout{
		ID:        "w-123",
		Title:     "Push Day",
		RoutineID: "r-456",
		StartTime: "2026-08-04T09:00:00Z",
		EndTime:   "2026-08-04T10:15:00Z",
		Exercises: []Exercise{
			{
				Index:              0,
				Title:              "Bench Press (Barbell)",
				Notes:              "felt strong",
				ExerciseTemplateID: "05293BCA",
				SupersetsID:        intPtr(2),
				Sets: []Set{
					{Index: 0, Type: "warmup", WeightKg: floatPtr(40), Reps: intPtr(10)},
					{Index: 1, Type: "normal", WeightKg: floatPtr(80), Reps: intPtr(8), RPE: floatPtr(9)},
					{Index: 2, Type: "failure", WeightKg: floatPtr(80), Reps: intPtr(6), RPE: floatPtr(10)},
				},
			},
			{
				Index:              1,
				Title:              "Plank",
				ExerciseTemplateID: "AAAA1111",
				Sets: []Set{
					{Index: 0, Type: "normal", DurationSeconds: floatPtr(60)},
				},
			},
		},
	}
}

func TestMapWorkout(t *testing.T) {
	rows, err := MapWorkout(sampleWorkout(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}

	first := rows[0]
	if first.Source != SourceName {
		t.Errorf("Source = %q, want %q", first.Source, SourceName)
	}
	if first.ExternalID != "w-123" || first.RoutineID != "r-456" {
		t.Errorf("external/routine id = %q/%q, want w-123/r-456", first.ExternalID, first.RoutineID)
	}
	if first.UserID != 7 {
		t.Errorf("UserID = %d, want 7", first.UserID)
	}
	if first.SessionName != "Push Day" {
		t.Errorf("SessionName = %q, want Push Day", first.SessionName)
	}
	if first.SessionDuration != "1:15 hr" {
		t.Errorf("SessionDuration = %q, want 1:15 hr", first.SessionDuration)
	}
	if first.SessionEnd == nil {
		t.Fatal("SessionEnd must be set when the workout has an end time")
	}

	// Hevy indexes from zero; workout_sets counts from one, like Alpha.
	if first.ExerciseNumber != 1 || first.SetNumber != 1 {
		t.Errorf("first row numbering = %d/%d, want 1/1", first.ExerciseNumber, first.SetNumber)
	}
	if !first.IsWarmup {
		t.Error("a set of type warmup must set IsWarmup")
	}
	if first.RPE != nil {
		t.Errorf("unrated warmup RPE = %v, want nil", *first.RPE)
	}
	if first.SupersetID == nil || *first.SupersetID != 2 {
		t.Errorf("SupersetID = %v, want 2", first.SupersetID)
	}

	working := rows[1]
	if working.IsWarmup {
		t.Error("a normal set must not set IsWarmup")
	}
	if working.RPE == nil || *working.RPE != 9 {
		t.Errorf("RPE = %v, want 9", working.RPE)
	}
	if working.WeightKg != 80 || working.Reps != 8 {
		t.Errorf("weight/reps = %v/%d, want 80/8", working.WeightKg, working.Reps)
	}

	failure := rows[2]
	if failure.SetType != "failure" {
		t.Errorf("SetType = %q, want failure", failure.SetType)
	}
	if failure.IsWarmup {
		t.Error("a failure set must not count as warmup")
	}

	plank := rows[3]
	if plank.ExerciseNumber != 2 {
		t.Errorf("second exercise number = %d, want 2", plank.ExerciseNumber)
	}
	if plank.DurationSec == nil || *plank.DurationSec != 60 {
		t.Errorf("DurationSec = %v, want 60", plank.DurationSec)
	}
	if plank.Reps != 0 || plank.WeightKg != 0 {
		t.Errorf("a duration-only set should leave reps and weight at zero, got %d/%v", plank.Reps, plank.WeightKg)
	}
	if plank.SupersetID != nil {
		t.Errorf("SupersetID = %v, want nil for an exercise outside a superset", plank.SupersetID)
	}
}

func TestMapWorkoutRejectsUnparsableStart(t *testing.T) {
	w := sampleWorkout()
	w.StartTime = "not a timestamp"
	if _, err := MapWorkout(w, 1); err == nil {
		t.Fatal("expected an error rather than a zero-valued session date")
	}
}

func TestMapWorkoutWithoutEndTime(t *testing.T) {
	w := sampleWorkout()
	w.EndTime = ""
	rows, err := MapWorkout(w, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].SessionEnd != nil {
		t.Error("SessionEnd must stay nil when the workout has no end time")
	}
	if rows[0].SessionDuration != "" {
		t.Errorf("SessionDuration = %q, want empty", rows[0].SessionDuration)
	}
}

func TestMapWorkoutDefaultsSetType(t *testing.T) {
	w := sampleWorkout()
	w.Exercises = []Exercise{{Index: 0, Title: "X", Sets: []Set{{Index: 0}}}}
	rows, err := MapWorkout(w, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].SetType != "normal" {
		t.Errorf("SetType = %q, want normal", rows[0].SetType)
	}
}
