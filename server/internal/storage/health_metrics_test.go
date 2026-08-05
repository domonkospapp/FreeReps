package storage

import (
	"strings"
	"testing"
)

// TestSourcePriorityCaseSQL verifies that the SQL CASE expression correctly
// maps source names to priority numbers, ensuring higher-priority sources
// win during deduplication.
func TestSourcePriorityCaseSQL(t *testing.T) {
	tests := []struct {
		name       string
		priorities []string
		wantSQL    string
	}{
		{
			name:       "empty priorities returns constant 1 (no-op dedup)",
			priorities: nil,
			wantSQL:    "1",
		},
		{
			name:       "single named source",
			priorities: []string{"Oura"},
			wantSQL:    "CASE WHEN source LIKE 'Oura%' THEN 1 ELSE 2 END",
		},
		{
			name:       "oura then empty string",
			priorities: []string{"Oura", ""},
			wantSQL:    "CASE WHEN source LIKE 'Oura%' THEN 1 WHEN source = '' THEN 2 ELSE 3 END",
		},
		{
			name:       "three sources with prefix matching",
			priorities: []string{"Oura", "Apple Watch", ""},
			wantSQL:    "CASE WHEN source LIKE 'Oura%' THEN 1 WHEN source LIKE 'Apple Watch%' THEN 2 WHEN source = '' THEN 3 ELSE 4 END",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sourcePriorityCaseSQL(tt.priorities)
			if got != tt.wantSQL {
				t.Errorf("sourcePriorityCaseSQL() =\n  %q\nwant:\n  %q", got, tt.wantSQL)
			}
		})
	}
}

// TestDedupCTE verifies that the generated CTE has the correct structure:
// a WITH clause using time_bucket, ROW_NUMBER, and the right parameter placeholders.
func TestDedupCTE(t *testing.T) {
	cte := dedupCTE([]string{"Oura", ""}, "$2", "$3", "$4", "$5")

	checks := []string{
		"WITH deduped AS",
		"time_bucket('5 minutes', time)",
		"ROW_NUMBER()",
		"LIKE 'Oura%' THEN 1",
		"source = '' THEN 2",
		"metric_name = $2",
		"time >= $3",
		"time < $4",
		"user_id = $5",
	}

	for _, check := range checks {
		if !strings.Contains(cte, check) {
			t.Errorf("dedupCTE missing %q in:\n%s", check, cte)
		}
	}
}

// TestLatestMetricsQueryDedupesBySourcePriority exists because the latest value
// and the series drawn beside it used to be resolved differently: the series
// deduplicated by source priority while the latest row was picked by timestamp
// alone. A lower-priority device writing a minute later then decided both the
// value and the source name shown next to a sparkline computed from the other
// device.
func TestLatestMetricsQueryDedupesBySourcePriority(t *testing.T) {
	query := latestMetricsQuery([]string{"Oura", ""})

	checks := []string{
		"WITH deduped AS",
		"PARTITION BY metric_name, time_bucket('5 minutes', time)",
		// Priority decides inside a bucket, recency decides between buckets.
		"WHEN source LIKE 'Oura%' THEN 1",
		"WHERE rn = 1",
		"ORDER BY metric_name, time DESC",
	}

	for _, check := range checks {
		if !strings.Contains(query, check) {
			t.Errorf("latestMetricsQuery missing %q in:\n%s", check, query)
		}
	}
}

// TestLatestMetricsQueryWithoutPrioritiesIsANoOp verifies the query still
// resolves when no priority is configured, rather than emitting an empty CASE.
func TestLatestMetricsQueryWithoutPrioritiesIsANoOp(t *testing.T) {
	query := latestMetricsQuery(nil)

	if !strings.Contains(query, "ORDER BY 1") {
		t.Errorf("expected the no-op ordering, got:\n%s", query)
	}
}

// TestDedupCTEMultiMetric verifies the multi-metric CTE partitions by both
// metric_name and time bucket, preventing cross-metric deduplication.
func TestDedupCTEMultiMetric(t *testing.T) {
	cte := dedupCTEMultiMetric([]string{"Oura", ""}, "$1", "$2,$3")

	checks := []string{
		"WITH deduped AS",
		"PARTITION BY metric_name, time_bucket('5 minutes', time)",
		"user_id = $1",
		"metric_name IN ($2,$3)",
	}

	for _, check := range checks {
		if !strings.Contains(cte, check) {
			t.Errorf("dedupCTEMultiMetric missing %q in:\n%s", check, cte)
		}
	}
}
