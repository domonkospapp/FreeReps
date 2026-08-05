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
		// Step one: the newest timestamp per metric, straight off the index.
		"SELECT DISTINCT ON (metric_name) metric_name, time AS peak",
		// Step two: only the five minutes priority is defined over.
		"h.time > n.peak - interval '5 minutes'",
		// Priority decides within that window, recency breaks the tie.
		"WHEN source LIKE 'Oura%' THEN 1",
		"ORDER BY h.metric_name,",
		"h.time DESC",
	}

	for _, check := range checks {
		if !strings.Contains(query, check) {
			t.Errorf("latestMetricsQuery missing %q in:\n%s", check, query)
		}
	}
}

// TestLatestMetricsQueryDoesNotWindowTheWholeTable exists because the obvious
// way to apply source priority — ROW_NUMBER over every row, then DISTINCT ON —
// numbered 4.5 million rows to return seventeen and cost the front page five
// seconds. The shape, not the result, is what regresses.
func TestLatestMetricsQueryDoesNotWindowTheWholeTable(t *testing.T) {
	query := latestMetricsQuery([]string{"Oura", ""})

	if strings.Contains(query, "ROW_NUMBER") {
		t.Errorf("latestMetricsQuery numbers rows again:\n%s", query)
	}
	// Every scan of health_metrics has to be bounded by a time predicate.
	if strings.Contains(query, "PARTITION BY") {
		t.Errorf("latestMetricsQuery partitions again:\n%s", query)
	}
}

// TestLatestMetricsRecentQueryIsBoundedInTime exists because health_metrics is
// a hypertable: without a lower bound on time, TimescaleDB cannot exclude
// chunks and each per-metric lookup walks back through all of them. The bound
// has to sit inside the LATERAL subquery, where the chunk scan happens — not in
// the outer join.
func TestLatestMetricsRecentQueryIsBoundedInTime(t *testing.T) {
	query := latestMetricsForNamesRecentQuery([]string{"Oura", ""})

	lateral := strings.Index(query, "CROSS JOIN LATERAL")
	closing := strings.Index(query[lateral:], ") l")
	if lateral < 0 || closing < 0 {
		t.Fatalf("unexpected query shape:\n%s", query)
	}
	inner := query[lateral : lateral+closing]

	if !strings.Contains(inner, "h.time >= $3") {
		t.Errorf("the lower bound is outside the LATERAL subquery:\n%s", query)
	}
	if !strings.Contains(inner, "LIMIT 1") {
		t.Errorf("expected a LIMIT 1 lookup per metric:\n%s", query)
	}
}

// TestDedupCTEMultiMetricRangeFiltersInsideTheCTE exists because the same
// filter one level out — in the caller's WHERE — makes Postgres number the
// user's whole history before narrowing to the window, which is why the front
// page took the same time for 30 days as for a year.
func TestDedupCTEMultiMetricRangeFiltersInsideTheCTE(t *testing.T) {
	cte := dedupCTEMultiMetricRange([]string{"Oura", ""}, "$1", "$2,$3", "$4", "$5")

	openParen := strings.Index(cte, "(")
	closeParen := strings.LastIndex(cte, ")")
	if openParen < 0 || closeParen < 0 {
		t.Fatalf("unexpected CTE shape:\n%s", cte)
	}
	inner := cte[openParen:closeParen]

	for _, check := range []string{"time >= $4", "time < $5"} {
		if !strings.Contains(inner, check) {
			t.Errorf("range predicate %q is outside the CTE body:\n%s", check, cte)
		}
	}
}

// TestLatestMetricsQueryWithoutPrioritiesIsANoOp verifies the query still
// resolves when no priority is configured, rather than emitting an empty CASE.
func TestLatestMetricsQueryWithoutPrioritiesIsANoOp(t *testing.T) {
	query := latestMetricsQuery(nil)

	// sourcePriorityCaseSQL collapses to the constant 1, leaving recency as the
	// only tiebreaker rather than emitting an empty CASE.
	if !strings.Contains(query, "ORDER BY h.metric_name, 1, h.time DESC") {
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
