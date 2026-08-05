package storage

import "time"

// sqlTimestamp renders a time as a SQL literal for embedding directly in a
// query, rather than passing it as a bind parameter.
//
// This exists for one reason, measured on the production instance: a bind
// parameter hides the value from the planner, so it cannot exclude chunks while
// planning and must build a plan covering all of them. health_metrics spans
// twelve years in 7-day chunks — 514 of them — and planning a query whose time
// bound is a parameter took 584ms against 12ms for the same query with a
// literal. The execution was never the expensive part.
//
// Embedding a value in SQL is normally how injection happens. It is safe here
// and only here: the input is a time.Time, and the output of Format is always
// digits, hyphens, colons and a zone offset. No caller-supplied string reaches
// it. Do not extend this pattern to any other type.
func sqlTimestamp(t time.Time) string {
	return "TIMESTAMPTZ '" + t.UTC().Format("2006-01-02 15:04:05.000000-07") + "'"
}
