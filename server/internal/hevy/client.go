package hevy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.hevyapp.com"

// maxPageSize is the ceiling the API enforces on /v1/workouts and
// /v1/workouts/events. Requesting more is rejected, so paging is unavoidable
// even for a single sync cycle.
const maxPageSize = 10

// APIError is a non-200 response from the Hevy API.
type APIError struct {
	Path       string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hevy API %s returned %d: %s", e.Path, e.StatusCode, e.Body)
}

// IsNotFound reports whether the endpoint or resource does not exist.
func (e *APIError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

// IsUnauthorized reports whether the API key was rejected. Hevy requires an
// active Pro subscription for API access, so this also covers a lapsed plan.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// Client is a Hevy REST API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a client against the production API.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// newTestClient points the client at a test server.
func newTestClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// get performs an authenticated GET request. Unlike Oura, Hevy authenticates
// with a static account-wide key in the `api-key` header rather than a bearer
// token, so there is no refresh path.
func (c *Client) get(ctx context.Context, path, apiKey string, params url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Path: path, StatusCode: resp.StatusCode, Body: string(body)}
	}
	return body, nil
}

// GetWorkoutEvents fetches one page of the event feed. Events are ordered newest
// first and cover both updates and deletions since the given timestamp. Pass an
// empty `since` to fetch the full history.
func (c *Client) GetWorkoutEvents(ctx context.Context, apiKey, since string, page int) (*PaginatedWorkoutEvents, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("pageSize", strconv.Itoa(maxPageSize))
	if since != "" {
		params.Set("since", since)
	}

	body, err := c.get(ctx, "/v1/workouts/events", apiKey, params)
	if err != nil {
		return nil, err
	}

	var result PaginatedWorkoutEvents
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding workout events: %w", err)
	}
	return &result, nil
}

// GetWorkouts fetches one page of the full workout list, newest first. Used for
// the initial backfill, where the event feed returns nothing.
func (c *Client) GetWorkouts(ctx context.Context, apiKey string, page int) (*PaginatedWorkouts, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("pageSize", strconv.Itoa(maxPageSize))

	body, err := c.get(ctx, "/v1/workouts", apiKey, params)
	if err != nil {
		return nil, err
	}

	var result PaginatedWorkouts
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding workouts: %w", err)
	}
	return &result, nil
}

// GetUserInfo returns the authenticated user's profile. The sync does not need
// it; the credentials handler uses it to verify an API key before storing it.
func (c *Client) GetUserInfo(ctx context.Context, apiKey string) (*UserInfo, error) {
	body, err := c.get(ctx, "/v1/user/info", apiKey, nil)
	if err != nil {
		return nil, err
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding user info: %w", err)
	}
	return &info, nil
}
