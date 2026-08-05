package storage

import (
	"math"
	"testing"
	"time"
)

func day(start time.Time, n int) time.Time {
	return start.AddDate(0, 0, n)
}

func TestBuildDashboardMetricPlacesPointsByDay(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	points := []DailyPoint{
		{Day: day(start, 0), Value: 10},
		{Day: day(start, 2), Value: 30},
	}

	series, _, _, _, _ := BuildDashboardMetric(points, start, 5, 5)

	if len(series) != 5 {
		t.Fatalf("series length = %d, want 5", len(series))
	}
	if series[0] == nil || *series[0] != 10 {
		t.Errorf("series[0] = %v, want 10", series[0])
	}
	// A day with no samples stays nil rather than being filled in, so the
	// sparkline shows the gap.
	if series[1] != nil {
		t.Errorf("series[1] = %v, want nil for the day without samples", *series[1])
	}
	if series[2] == nil || *series[2] != 30 {
		t.Errorf("series[2] = %v, want 30", series[2])
	}
}

func TestBuildDashboardMetricDeltaComparesTwoWeeks(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	var points []DailyPoint
	// Days 16-22 average 10, days 23-29 average 14.
	for i := 16; i < 23; i++ {
		points = append(points, DailyPoint{Day: day(start, i), Value: 10})
	}
	for i := 23; i < 30; i++ {
		points = append(points, DailyPoint{Day: day(start, i), Value: 14})
	}

	_, delta, deltaPct, _, _ := BuildDashboardMetric(points, start, 30, 30)

	if delta == nil {
		t.Fatal("delta = nil, want 4")
	}
	if math.Abs(*delta-4) > 1e-9 {
		t.Errorf("delta = %v, want 4", *delta)
	}
	if deltaPct == nil {
		t.Fatal("deltaPct = nil, want 0.4")
	}
	if math.Abs(*deltaPct-0.4) > 1e-9 {
		t.Errorf("deltaPct = %v, want 0.4", *deltaPct)
	}
}

func TestBuildDashboardMetricWithoutPriorWindowReportsNoDelta(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	// Only the recent window carries data; the prior one is empty.
	var points []DailyPoint
	for i := 25; i < 30; i++ {
		points = append(points, DailyPoint{Day: day(start, i), Value: 7})
	}

	_, delta, deltaPct, low, high := BuildDashboardMetric(points, start, 30, 30)

	if delta != nil {
		t.Errorf("delta = %v, want nil when the prior window has no samples", *delta)
	}
	if deltaPct != nil {
		t.Errorf("deltaPct = %v, want nil", *deltaPct)
	}
	// The range is still computable from what is there.
	if low == nil || high == nil {
		t.Fatal("range should be computable from the samples present")
	}
	if *low != 7 || *high != 7 {
		t.Errorf("range = %v..%v, want 7..7", *low, *high)
	}
}

func TestBuildDashboardMetricShortSelectionStillHasADelta(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	// A 7-day selection queries 14 days, so both delta windows are covered.
	var points []DailyPoint
	for i := 0; i < 7; i++ {
		points = append(points, DailyPoint{Day: day(start, i), Value: 10})
	}
	for i := 7; i < 14; i++ {
		points = append(points, DailyPoint{Day: day(start, i), Value: 12})
	}

	series, delta, _, low, high := BuildDashboardMetric(points, start, 14, 7)

	if len(series) != 7 {
		t.Fatalf("series length = %d, want the 7 shown days", len(series))
	}
	// The series is the tail of the buffer: the second, higher week.
	if series[0] == nil || *series[0] != 12 {
		t.Errorf("series[0] = %v, want 12", series[0])
	}
	if delta == nil || math.Abs(*delta-2) > 1e-9 {
		t.Errorf("delta = %v, want 2 from the full 14-day buffer", delta)
	}
	// The stated range describes the shown week only.
	if low == nil || high == nil || *low != 12 || *high != 12 {
		t.Errorf("range should cover the shown series only, got %v..%v", low, high)
	}
}

func TestBuildDashboardMetricEmpty(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	series, delta, deltaPct, low, high := BuildDashboardMetric(nil, start, 30, 30)

	if len(series) != 30 {
		t.Errorf("series length = %d, want 30 slots even with no data", len(series))
	}
	for i, v := range series {
		if v != nil {
			t.Errorf("series[%d] = %v, want nil", i, *v)
		}
	}
	if delta != nil || deltaPct != nil || low != nil || high != nil {
		t.Error("derived values should all be nil for an empty series")
	}
}

func TestPercentileRangeIgnoresASingleOutlier(t *testing.T) {
	values := make([]float64, 0, 100)
	for i := 0; i < 99; i++ {
		values = append(values, 50)
	}
	// One dropped sensor reading. min/max would state 0..50; p05/p95 should not.
	values = append(values, 0)

	low, high := percentileRange(values, 0.05, 0.95)

	if low == nil || high == nil {
		t.Fatal("percentileRange returned nil")
	}
	if *low != 50 || *high != 50 {
		t.Errorf("range = %v..%v, want 50..50", *low, *high)
	}
}

func TestPercentileInterpolates(t *testing.T) {
	sorted := []float64{0, 10, 20, 30, 40}

	if got := percentile(sorted, 0.5); got != 20 {
		t.Errorf("p50 = %v, want 20", got)
	}
	if got := percentile(sorted, 0.25); got != 10 {
		t.Errorf("p25 = %v, want 10", got)
	}
	// 0.125 * 4 = 0.5, halfway between 0 and 10.
	if got := percentile(sorted, 0.125); got != 5 {
		t.Errorf("p12.5 = %v, want 5", got)
	}
}
