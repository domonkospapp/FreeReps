package hevy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/claude/freereps/internal/config"
	"github.com/claude/freereps/internal/storage"
)

// resyncOverlap is subtracted from the stored cursor on every cycle. A workout
// saved while a sync was in flight would otherwise fall between two windows and
// never arrive.
const resyncOverlap = time.Hour

// maxPages bounds one sync cycle. At ten events per page this covers 5000
// workouts, far above any real history, and stops a malformed page_count from
// producing an endless loop.
const maxPages = 500

// syncStats accumulates counts across one sync cycle for the import log.
type syncStats struct {
	workoutsReceived int
	workoutsInserted int
	workoutsDeleted  int
	workoutsSkipped  int
	setsReceived     int
	setsInserted     int64
	errors           []string
}

// Syncer polls the Hevy API and stores strength training sets in FreeReps.
type Syncer struct {
	client *Client
	db     *storage.DB
	cfg    config.HevyConfig
	log    *slog.Logger
}

// NewSyncer creates a new Hevy sync orchestrator.
func NewSyncer(client *Client, db *storage.DB, cfg config.HevyConfig, log *slog.Logger) *Syncer {
	return &Syncer{client: client, db: db, cfg: cfg, log: log}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (s *Syncer) Run(ctx context.Context) {
	if err := s.SyncOnce(ctx); err != nil {
		s.log.Error("initial hevy sync failed", "error", err)
	}

	ticker := time.NewTicker(s.cfg.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("hevy sync stopped")
			return
		case <-ticker.C:
			if err := s.SyncOnce(ctx); err != nil {
				s.log.Error("hevy sync cycle failed", "error", err)
			}
		}
	}
}

// SyncOnce performs one sync cycle for all users with stored credentials.
func (s *Syncer) SyncOnce(ctx context.Context) error {
	users, err := s.db.ListHevyCredentialUsers(ctx)
	if err != nil {
		return fmt.Errorf("listing hevy users: %w", err)
	}
	if len(users) == 0 {
		return nil
	}

	s.log.Info("hevy sync starting", "users", len(users))
	for _, uid := range users {
		s.SyncUser(ctx, uid)
	}
	s.log.Info("hevy sync complete")
	return nil
}

// TriggerSync runs a sync for one user on demand.
func (s *Syncer) TriggerSync(ctx context.Context, userID int) error {
	s.SyncUser(ctx, userID)
	return nil
}

// SyncUser pulls the event feed for one user and writes an import log.
func (s *Syncer) SyncUser(ctx context.Context, userID int) {
	start := time.Now()
	stats := &syncStats{}

	creds, err := s.db.GetHevyCredentials(ctx, userID)
	if err != nil {
		s.logImport(ctx, userID, start, stats, fmt.Errorf("getting credentials: %w", err))
		return
	}
	if creds == nil {
		return
	}

	// The cursor is taken before fetching. Using the completion time instead
	// would drop every event that arrives while the cycle is running.
	cursor := time.Now()

	since := ""
	state, err := s.db.GetHevySyncState(ctx, userID)
	if err != nil {
		s.logImport(ctx, userID, start, stats, fmt.Errorf("getting sync state: %w", err))
		return
	}
	if state != nil {
		since = state.LastEventAt.Add(-resyncOverlap).UTC().Format(time.RFC3339)
	}

	if err := s.syncEvents(ctx, userID, creds, since, stats); err != nil {
		s.logImport(ctx, userID, start, stats, err)
		return
	}

	if err := s.db.UpsertHevySyncState(ctx, userID, cursor); err != nil {
		s.logImport(ctx, userID, start, stats, fmt.Errorf("saving sync state: %w", err))
		return
	}

	s.logImport(ctx, userID, start, stats, nil)
}

// syncEvents walks every page of the event feed and applies each event.
func (s *Syncer) syncEvents(ctx context.Context, userID int, creds *storage.HevyCredentials, since string, stats *syncStats) error {
	for page := 1; page <= maxPages; page++ {
		resp, err := s.client.GetWorkoutEvents(ctx, creds.APIKey, since, page)
		if err != nil {
			return fmt.Errorf("fetching workout events page %d: %w", page, err)
		}

		for _, ev := range resp.Events {
			if err := s.applyEvent(ctx, userID, creds, ev, stats); err != nil {
				// One malformed workout must not abort the whole cycle;
				// record it and carry on so the rest still lands.
				stats.errors = append(stats.errors, err.Error())
				s.log.Warn("hevy event failed", "error", err)
			}
		}

		if resp.PageCount <= page {
			return nil
		}
	}
	return fmt.Errorf("event feed exceeded %d pages", maxPages)
}

// applyEvent writes a single update or deletion.
func (s *Syncer) applyEvent(ctx context.Context, userID int, creds *storage.HevyCredentials, ev WorkoutEvent, stats *syncStats) error {
	switch ev.Type {
	case EventDeleted:
		if ev.ID == "" {
			return fmt.Errorf("deleted event without id")
		}
		n, err := s.db.DeleteWorkoutSetsByExternalID(ctx, userID, SourceName, ev.ID)
		if err != nil {
			return err
		}
		if n > 0 {
			stats.workoutsDeleted++
		}
		return nil

	case EventUpdated:
		if ev.Workout == nil {
			return fmt.Errorf("updated event without workout")
		}
		w := *ev.Workout
		stats.workoutsReceived++

		startTime, err := WorkoutStart(w)
		if err != nil {
			return err
		}
		// Workouts before the cutoff are ignored. Without this an Alpha history
		// later imported into Hevy would flow back and count a second time
		// against the rows already in workout_sets.
		if startTime.Before(creds.SyncFrom) {
			stats.workoutsSkipped++
			return nil
		}

		rows, err := MapWorkout(w, userID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		// Delete before insert. ON CONFLICT DO NOTHING alone would discard a
		// correction made in the app, and a set removed there would survive.
		if _, err := s.db.DeleteWorkoutSetsByExternalID(ctx, userID, SourceName, w.ID); err != nil {
			return err
		}
		inserted, err := s.db.InsertWorkoutSets(ctx, rows)
		if err != nil {
			return err
		}
		stats.setsReceived += len(rows)
		stats.setsInserted += inserted
		stats.workoutsInserted++
		return nil

	default:
		return fmt.Errorf("unknown event type %q", ev.Type)
	}
}

// logImport writes an import log entry for one Hevy sync cycle.
func (s *Syncer) logImport(ctx context.Context, userID int, start time.Time, stats *syncStats, syncErr error) {
	durationMs := int(time.Since(start).Milliseconds())
	status := "success"
	var errMsg *string

	if syncErr != nil {
		status = "error"
		msg := syncErr.Error()
		errMsg = &msg
	}

	var metadata *json.RawMessage
	if len(stats.errors) > 0 || stats.workoutsDeleted > 0 || stats.workoutsSkipped > 0 {
		payload := map[string]any{}
		if len(stats.errors) > 0 {
			payload["event_errors"] = stats.errors
		}
		if stats.workoutsDeleted > 0 {
			payload["workouts_deleted"] = stats.workoutsDeleted
		}
		if stats.workoutsSkipped > 0 {
			payload["workouts_before_cutoff"] = stats.workoutsSkipped
		}
		raw, _ := json.Marshal(payload)
		rm := json.RawMessage(raw)
		metadata = &rm
	}

	if _, err := s.db.InsertImportLog(ctx, storage.ImportLog{
		UserID:           userID,
		Source:           "hevy_sync",
		Status:           status,
		WorkoutsReceived: stats.workoutsReceived,
		WorkoutsInserted: stats.workoutsInserted,
		SetsReceived:     stats.setsReceived,
		SetsInserted:     stats.setsInserted,
		DurationMs:       &durationMs,
		ErrorMessage:     errMsg,
		Metadata:         metadata,
	}); err != nil {
		s.log.Error("failed to log hevy import", "error", err)
	}
}
