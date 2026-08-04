package hevy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetWorkoutEventsSendsAPIKeyAndParams(t *testing.T) {
	var gotKey, gotSince, gotPage, gotPageSize string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("api-key")
		gotSince = r.URL.Query().Get("since")
		gotPage = r.URL.Query().Get("page")
		gotPageSize = r.URL.Query().Get("pageSize")
		_ = json.NewEncoder(w).Encode(PaginatedWorkoutEvents{Page: 1, PageCount: 1})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.GetWorkoutEvents(context.Background(), "secret-key", "2026-08-01T00:00:00Z", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotKey != "secret-key" {
		t.Errorf("api-key header = %q, want %q", gotKey, "secret-key")
	}
	if gotSince != "2026-08-01T00:00:00Z" {
		t.Errorf("since = %q, want %q", gotSince, "2026-08-01T00:00:00Z")
	}
	if gotPage != "1" {
		t.Errorf("page = %q, want 1", gotPage)
	}
	if gotPageSize != "10" {
		t.Errorf("pageSize = %q, want 10 (the API ceiling)", gotPageSize)
	}
}

func TestGetWorkoutEventsOmitsSinceWhenEmpty(t *testing.T) {
	var hasSince bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasSince = r.URL.Query()["since"]
		_ = json.NewEncoder(w).Encode(PaginatedWorkoutEvents{Page: 1, PageCount: 1})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.GetWorkoutEvents(context.Background(), "k", "", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasSince {
		t.Error("since parameter was sent although no cursor exists; that would bound the first full fetch")
	}
}

func TestGetWorkoutEventsDecodesBothEventTypes(t *testing.T) {
	body := `{
	  "page": 1,
	  "page_count": 1,
	  "events": [
	    {"type":"updated","workout":{"id":"w1","title":"Push","start_time":"2026-08-04T09:00:00Z","end_time":"2026-08-04T10:15:00Z","exercises":[]}},
	    {"type":"deleted","id":"w2","deleted_at":"2026-08-04T11:00:00Z"}
	  ]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	resp, err := c.GetWorkoutEvents(context.Background(), "k", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("got %d events, want 2", len(resp.Events))
	}
	if resp.Events[0].Type != EventUpdated || resp.Events[0].Workout == nil {
		t.Errorf("first event should carry a workout, got %+v", resp.Events[0])
	}
	if resp.Events[0].Workout.ID != "w1" {
		t.Errorf("workout id = %q, want w1", resp.Events[0].Workout.ID)
	}
	if resp.Events[1].Type != EventDeleted || resp.Events[1].ID != "w2" {
		t.Errorf("second event should be a deletion of w2, got %+v", resp.Events[1])
	}
	if resp.Events[1].Workout != nil {
		t.Error("deleted event must not carry a workout")
	}
}

func TestGetWorkoutEventsUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetWorkoutEvents(context.Background(), "bad", "", 1)
	if err == nil {
		t.Fatal("expected an error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if !apiErr.IsUnauthorized() {
		t.Errorf("IsUnauthorized() = false for status %d", apiErr.StatusCode)
	}
}

func TestGetUserInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/user/info" {
			t.Errorf("path = %q, want /v1/user/info", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"u1","name":"Test","url":"https://hevy.com/user/test"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	info, err := c.GetUserInfo(context.Background(), "k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "Test" {
		t.Errorf("name = %q, want Test", info.Name)
	}
}
