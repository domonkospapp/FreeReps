//go:build integration

// Package alpha's integration tests run against a real PostgreSQL server,
// because the property they check — that a second import of an unchanged file
// changes nothing — is enforced by a unique constraint, not by Go code. A unit
// test with a fake store would assert the fake's behaviour and prove nothing
// about workout_sets_source_natural_key.
//
// Run with:
//
//	FREEREPS_TEST_DSN=postgres://user:pass@host:port/db?sslmode=disable \
//	  go test -tags integration ./internal/ingest/alpha/
package alpha

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	// Resolving Europe/Berlin by name must not depend on the host carrying
	// /usr/share/zoneinfo.
	_ "time/tzdata"

	"github.com/claude/freereps/internal/storage"
)

const importCSV = `"Legs · Day 2 · Week 4 · Push-Pull-Legs";"2026-02-19 4:54 h";"1:02 hr"
"1. Hack Squats · Machine · 8 reps";"WU1 · 37,5 kg · 9 reps"
#;KG;REPS;RIR
1;115;8;1
2;115;10;1

"Push · Day 1 · Week 4 · Push-Pull-Legs";"2026-02-17 5:04 h";"1:12 hr"
"1. Bench Press · Barbell · 6 reps";"WU1 · 22,5 kg · 10 reps"
#;KG;REPS;RIR
1;102,5;6;0
2;100;6;0
`

// testDB connects to FREEREPS_TEST_DSN, applies the migrations and empties the
// tables this test writes to. It refuses to run against the deployed database
// name, because the test truncates.
func testDB(t *testing.T) *storage.DB {
	t.Helper()
	dsn := os.Getenv("FREEREPS_TEST_DSN")
	if dsn == "" {
		t.Skip("FREEREPS_TEST_DSN not set")
	}

	migrations, err := filepath.Abs("../../../migrations")
	if err != nil {
		t.Fatalf("resolving migrations path: %v", err)
	}
	if err := storage.RunMigrations(dsn, migrations); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	ctx := context.Background()
	db, err := storage.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	t.Cleanup(db.Close)

	var dbName string
	if err := db.Pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		t.Fatalf("reading database name: %v", err)
	}
	if dbName == "freereps" {
		t.Fatalf("refusing to truncate the database named %q — point FREEREPS_TEST_DSN at a scratch database", dbName)
	}

	if _, err := db.Pool.Exec(ctx, `TRUNCATE workout_sets`); err != nil {
		t.Fatalf("truncating workout_sets: %v", err)
	}
	return db
}

func countSets(t *testing.T, db *storage.DB) (rows int, sessions int) {
	t.Helper()
	err := db.Pool.QueryRow(context.Background(),
		`SELECT count(*), count(DISTINCT session_date) FROM workout_sets`).Scan(&rows, &sessions)
	if err != nil {
		t.Fatalf("counting sets: %v", err)
	}
	return rows, sessions
}

// TestReimportIsIdempotent is the regression test for the duplicate import of
// 2026-08-10 (INCIDENTS.md): importing an unchanged export a second time must
// leave workout_sets exactly as it was. Before the fix this held only while
// the process timezone stayed the same between the two runs.
func TestReimportIsIdempotent(t *testing.T) {
	db := testDB(t)
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("loading Europe/Berlin: %v", err)
	}
	p := NewProvider(db, slog.New(slog.NewTextHandler(io.Discard, nil)), loc)
	ctx := context.Background()

	first, err := p.Ingest(ctx, strings.NewReader(importCSV), 1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.SetsInserted != 6 {
		t.Fatalf("first import inserted %d rows, want 6", first.SetsInserted)
	}
	rowsAfterFirst, sessionsAfterFirst := countSets(t, db)
	if sessionsAfterFirst != 2 {
		t.Fatalf("sessions after first import = %d, want 2", sessionsAfterFirst)
	}

	second, err := p.Ingest(ctx, strings.NewReader(importCSV), 1)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.SetsInserted != 0 {
		t.Errorf("second import inserted %d rows, want 0", second.SetsInserted)
	}
	rowsAfterSecond, sessionsAfterSecond := countSets(t, db)
	if rowsAfterSecond != rowsAfterFirst {
		t.Errorf("rows = %d after re-import, want %d", rowsAfterSecond, rowsAfterFirst)
	}
	if sessionsAfterSecond != sessionsAfterFirst {
		t.Errorf("sessions = %d after re-import, want %d", sessionsAfterSecond, sessionsAfterFirst)
	}
}

// TestReimportUnderADifferentProcessTimezoneIsIdempotent reproduces the exact
// mechanism of the 2026-08-10 duplication: the first import ran on a machine
// in Europe/Berlin, the second in the deployed container, which has no
// /etc/localtime and therefore runs in UTC. With the zone taken from
// configuration instead of time.Local, the process timezone no longer reaches
// session_date and the second import inserts nothing.
func TestReimportUnderADifferentProcessTimezoneIsIdempotent(t *testing.T) {
	db := testDB(t)
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("loading Europe/Berlin: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	importUnder := func(processZone *time.Location) int64 {
		saved := time.Local
		time.Local = processZone
		defer func() { time.Local = saved }()

		res, err := NewProvider(db, log, loc).Ingest(ctx, strings.NewReader(importCSV), 1)
		if err != nil {
			t.Fatalf("import under %s: %v", processZone, err)
		}
		return res.SetsInserted
	}

	if got := importUnder(loc); got != 6 {
		t.Fatalf("first import inserted %d rows, want 6", got)
	}
	if got := importUnder(time.UTC); got != 0 {
		t.Errorf("re-import in a UTC process inserted %d rows, want 0", got)
	}
	if got := importUnder(time.FixedZone("Asia/Kathmandu", 5*3600+45*60)); got != 0 {
		t.Errorf("re-import in a +05:45 process inserted %d rows, want 0", got)
	}

	rows, sessions := countSets(t, db)
	if rows != 6 || sessions != 2 {
		t.Errorf("after three imports: %d rows in %d sessions, want 6 rows in 2 sessions", rows, sessions)
	}
}

// generateCSV renders an export with sessions sessions of setsPerSession
// working sets each, in the shape Parse expects.
func generateCSV(sessions, setsPerSession int) string {
	var b strings.Builder
	for s := 0; s < sessions; s++ {
		// One session every other day, starting 2025-01-01, all at 07:15.
		day := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, s*2)
		fmt.Fprintf(&b, "\"Full Body · Session %d\";\"%s 7:15 h\";\"1:00 hr\"\n", s, day.Format("2006-01-02"))
		fmt.Fprintf(&b, "\"1. Hack Squats · Machine · 8 reps\"\n#;KG;REPS;RIR\n")
		for n := 1; n <= setsPerSession; n++ {
			fmt.Fprintf(&b, "%d;100;8;1\n", n)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// TestImportLargerThanOneStatement covers an export whose row count exceeds the
// 65535-parameter ceiling of a single INSERT. A full Alpha history is around
// 4000 rows against 26 columns, so before InsertWorkoutSets batched, importing
// one failed outright with "extended protocol limited to 65535 parameters" —
// and an import that cannot run at all cannot be idempotent either.
func TestImportLargerThanOneStatement(t *testing.T) {
	db := testDB(t)
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("loading Europe/Berlin: %v", err)
	}
	p := NewProvider(db, slog.New(slog.NewTextHandler(io.Discard, nil)), loc)
	ctx := context.Background()

	const sessions, setsPerSession = 150, 20 // 3000 rows, well past 65535/26
	csv := generateCSV(sessions, setsPerSession)

	first, err := p.Ingest(ctx, strings.NewReader(csv), 1)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if want := int64(sessions * setsPerSession); first.SetsInserted != want {
		t.Fatalf("first import inserted %d rows, want %d", first.SetsInserted, want)
	}

	second, err := p.Ingest(ctx, strings.NewReader(csv), 1)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.SetsInserted != 0 {
		t.Errorf("second import inserted %d rows, want 0", second.SetsInserted)
	}

	rows, storedSessions := countSets(t, db)
	if rows != sessions*setsPerSession || storedSessions != sessions {
		t.Errorf("after two imports: %d rows in %d sessions, want %d rows in %d sessions",
			rows, storedSessions, sessions*setsPerSession, sessions)
	}
}
