// Package hevy ingests strength training data from the Hevy REST API.
//
// The wire format is documented in server/specs/hevy-api.md and derived from the
// OpenAPI 3.0 specification served at https://api.hevyapp.com/docs/.
package hevy

// Set is one logged set within an exercise.
//
// Every value field is nullable: a plank has duration_seconds but no reps, a
// bodyweight set has reps but no weight_kg. RPE is null whenever the user did
// not rate the set.
type Set struct {
	Index           int      `json:"index"`
	Type            string   `json:"type"` // normal | warmup | failure | dropset
	WeightKg        *float64 `json:"weight_kg"`
	Reps            *int     `json:"reps"`
	DistanceMeters  *float64 `json:"distance_meters"`
	DurationSeconds *float64 `json:"duration_seconds"`
	RPE             *float64 `json:"rpe"`
	CustomMetric    *float64 `json:"custom_metric"`
}

// Exercise is one exercise within a workout, with its sets.
type Exercise struct {
	Index              int    `json:"index"`
	Title              string `json:"title"`
	Notes              string `json:"notes"`
	ExerciseTemplateID string `json:"exercise_template_id"`
	SupersetsID        *int   `json:"supersets_id"`
	Sets               []Set  `json:"sets"`
}

// Workout is one completed training session.
//
// RoutineID names the routine the session was started from, which is what makes
// a comparison of prescribed against performed possible.
type Workout struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	RoutineID   string     `json:"routine_id"`
	Description string     `json:"description"`
	StartTime   string     `json:"start_time"`
	EndTime     string     `json:"end_time"`
	UpdatedAt   string     `json:"updated_at"`
	CreatedAt   string     `json:"created_at"`
	Exercises   []Exercise `json:"exercises"`
}

// Event types returned by GET /v1/workouts/events.
const (
	EventUpdated = "updated"
	EventDeleted = "deleted"
)

// WorkoutEvent is one entry in the event feed. The OpenAPI schema models this as
// a oneOf over an updated and a deleted variant; both are folded into one struct
// here and distinguished by Type.
type WorkoutEvent struct {
	Type      string   `json:"type"`
	Workout   *Workout `json:"workout,omitempty"`
	ID        string   `json:"id,omitempty"`
	DeletedAt string   `json:"deleted_at,omitempty"`
}

// PaginatedWorkoutEvents is the response of GET /v1/workouts/events.
type PaginatedWorkoutEvents struct {
	Page      int            `json:"page"`
	PageCount int            `json:"page_count"`
	Events    []WorkoutEvent `json:"events"`
}

// PaginatedWorkouts is the response of GET /v1/workouts.
//
// This endpoint carries the initial backfill. The event feed does not: its own
// description states it exists so a client can keep an existing local cache up
// to date "without having to fetch the entire list of workouts", and querying it
// without a `since` bound returns nothing for a history that predates the key.
type PaginatedWorkouts struct {
	Page      int       `json:"page"`
	PageCount int       `json:"page_count"`
	Workouts  []Workout `json:"workouts"`
}

// UserInfo is the response of GET /v1/user/info, used to validate an API key
// before it is stored.
type UserInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}
