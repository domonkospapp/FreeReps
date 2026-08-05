package withings

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetMeasuresRequestParameters verifies the request the client builds.
//
// It exists because of `category`: category 2 returns the user's target weight
// in the same shape as a measurement, so a request that omits the parameter
// mixes goals into the weight series as values nothing downstream can tell
// apart from a real measurement.
func TestGetMeasuresRequestParameters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
		}
		if r.URL.Path != "/measure" {
			t.Errorf("path = %q, want /measure", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth header = %q, want %q", got, "Bearer test-token")
		}
		if got := r.PostForm.Get("action"); got != "getmeas" {
			t.Errorf("action = %q, want getmeas", got)
		}
		if got := r.PostForm.Get("category"); got != "1" {
			t.Errorf("category = %q, want 1", got)
		}
		if got := r.PostForm.Get("meastypes"); got != "1,9,10" {
			t.Errorf("meastypes = %q, want %q", got, "1,9,10")
		}
		if got := r.PostForm.Get("lastupdate"); got != "1754300000" {
			t.Errorf("lastupdate = %q, want 1754300000", got)
		}
		if r.PostForm.Has("startdate") {
			t.Error("startdate sent alongside lastupdate")
		}
		fmt.Fprint(w, `{"status":0,"body":{"updatetime":1754380800,"more":false,"measuregrps":[]}}`)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, updateTime, err := client.GetMeasures(context.Background(), "test-token", []int{1, 9, 10}, 1754300000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateTime != 1754380800 {
		t.Errorf("updatetime = %d, want 1754380800", updateTime)
	}
}

// TestGetMeasuresBackfillWindow verifies that a zero lastUpdate produces a
// startdate/enddate window instead, which is what the first sync needs.
func TestGetMeasuresBackfillWindow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostForm.Has("lastupdate") {
			t.Error("lastupdate sent on a backfill request")
		}
		if got := r.PostForm.Get("startdate"); got != "1746000000" {
			t.Errorf("startdate = %q, want 1746000000", got)
		}
		if r.PostForm.Get("enddate") == "" {
			t.Error("enddate missing on a backfill request")
		}
		fmt.Fprint(w, `{"status":0,"body":{"updatetime":1,"more":false,"measuregrps":[]}}`)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	if _, _, err := client.GetMeasures(context.Background(), "t", []int{1}, 0, 1746000000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGetMeasuresErrorInBody verifies that a non-zero status is treated as an
// error. Withings answers HTTP 200 even when the call failed, so a client that
// only checks the HTTP status reads every failure as a successful empty result
// and the sync reports success while importing nothing.
func TestGetMeasuresErrorInBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":401,"body":null,"error":"Invalid params"}`)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, _, err := client.GetMeasures(context.Background(), "t", []int{1}, 0, 0)
	if err == nil {
		t.Fatal("expected an error for status 401, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if !apiErr.IsUnauthorized() {
		t.Errorf("IsUnauthorized() = false for status %d, want true", apiErr.Status)
	}
}

// TestGetMeasuresRevokedToken verifies that the Withings-specific token codes
// are recognized as unauthorized. They are not HTTP status codes and would
// otherwise be retried with the same dead token on every cycle.
func TestGetMeasuresRevokedToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":286,"error":"no user found"}`)
	}))
	defer srv.Close()

	_, _, err := newTestClient(srv.URL).GetMeasures(context.Background(), "t", []int{1}, 0, 0)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !apiErr.IsUnauthorized() {
		t.Fatalf("status 286 not reported as unauthorized: %v", err)
	}
}

// TestGetMeasuresPagination verifies that the client follows the more/offset
// cursor and concatenates the pages.
func TestGetMeasuresPagination(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		calls++
		switch calls {
		case 1:
			if r.PostForm.Has("offset") {
				t.Error("offset sent on the first page")
			}
			fmt.Fprint(w, `{"status":0,"body":{"updatetime":100,"more":true,"offset":2,
				"measuregrps":[{"grpid":1,"date":10,"category":1,"measures":[{"value":70000,"type":1,"unit":-3}]}]}}`)
		case 2:
			if got := r.PostForm.Get("offset"); got != "2" {
				t.Errorf("offset = %q, want 2", got)
			}
			fmt.Fprint(w, `{"status":0,"body":{"updatetime":200,"more":false,
				"measuregrps":[{"grpid":2,"date":20,"category":1,"measures":[{"value":71000,"type":1,"unit":-3}]}]}}`)
		default:
			t.Errorf("unexpected call %d", calls)
		}
	}))
	defer srv.Close()

	groups, updateTime, err := newTestClient(srv.URL).GetMeasures(context.Background(), "t", []int{1}, 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	if updateTime != 200 {
		t.Errorf("updatetime = %d, want 200 (the newest page)", updateTime)
	}
}

// TestGetMeasuresNumericMore verifies that `more` is accepted as a number.
// The documentation shows a boolean and the API has been observed sending 1/0;
// decoding straight into a bool would fail the whole response on the numeric
// form and lose the page that was already fetched.
func TestGetMeasuresNumericMore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":0,"body":{"updatetime":100,"more":0,"offset":0,"measuregrps":[]}}`)
	}))
	defer srv.Close()

	if _, _, err := newTestClient(srv.URL).GetMeasures(context.Background(), "t", []int{1}, 1, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
