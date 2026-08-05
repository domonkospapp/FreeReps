package storage

import (
	"math"
	"testing"
)

// The RIR term is the point of this formula. Without it every set stopped short
// of failure is credited only with the repetitions performed, which understates
// roughly half the sets in this data set and makes a trend line look flat when
// the lifter was simply leaving more in reserve.
func TestEpleyE1RM(t *testing.T) {
	tests := []struct {
		name      string
		weight    float64
		reps      int
		effortRIR float64
		want      float64
	}{
		{"single rep to failure is the weight itself, plus Epley's linear term", 100, 1, 0, 103.333},
		{"8 reps to failure", 100, 8, 0, 126.667},
		{"8 reps at RIR 2 counts as 10 effective reps", 100, 8, 2, 133.333},
		{"and matches 10 reps taken to failure", 100, 10, 0, 133.333},
		{"half a rep in reserve is honoured", 100, 8, 0.5, 128.333},
		{"zero weight yields no estimate", 0, 8, 1, 0},
		{"zero reps yields no estimate", 100, 0, 1, 0},
		{"a bodyweight set without load yields no estimate", 0, 12, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := epleyE1RM(tt.weight, tt.reps, tt.effortRIR)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("epleyE1RM(%v, %d, %v) = %.3f, want %.3f",
					tt.weight, tt.reps, tt.effortRIR, got, tt.want)
			}
		})
	}
}

// A set at RIR 2 must be credited above the same set taken to failure at the
// same weight and reps, because it demonstrates more strength.
func TestEpleyE1RMRewardsRepsInReserve(t *testing.T) {
	toFailure := epleyE1RM(100, 8, 0)
	withReserve := epleyE1RM(100, 8, 2)
	if withReserve <= toFailure {
		t.Errorf("RIR 2 estimate %.2f should exceed the to-failure estimate %.2f",
			withReserve, toFailure)
	}
}

// The weighting of assisting muscles is a convention, not a measurement. Pinning
// it in a test makes a change to it deliberate rather than incidental.
func TestSecondaryMuscleWeight(t *testing.T) {
	if secondaryMuscleWeight != 0.5 {
		t.Errorf("secondaryMuscleWeight = %v; changing it changes every weighted "+
			"volume figure and needs to be a stated decision", secondaryMuscleWeight)
	}
}
