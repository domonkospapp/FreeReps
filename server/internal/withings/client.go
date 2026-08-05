package withings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://wbsapi.withings.net"

// maxPages bounds the getmeas pagination loop. The API signals the last page
// through `more`; this is the backstop for a response that keeps claiming more
// without advancing, which would otherwise spin forever inside one sync cycle.
const maxPages = 100

// Withings status codes FreeReps distinguishes. See specs/withings-api.md.
const (
	statusOK              = 0
	statusUnauthorized    = 401
	statusTooManyRequests = 601
)

// invalidTokenStatuses are the codes that mean the token is no longer usable and
// the user has to authorize again.
var invalidTokenStatuses = map[int]bool{283: true, 284: true, 286: true, 293: true, 294: true}

// APIError represents a failed Withings call. Status is the code from the
// response body, not the HTTP status: Withings answers with HTTP 200 and
// signals failure inside the payload.
type APIError struct {
	Action string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("withings %s returned status %d: %s", e.Action, e.Status, e.Body)
}

// IsUnauthorized reports whether the call failed because the token is missing,
// expired or revoked — the cases where retrying with the same token is pointless.
func (e *APIError) IsUnauthorized() bool {
	return e.Status == statusUnauthorized || invalidTokenStatuses[e.Status]
}

// IsRateLimited reports whether the call was throttled.
func (e *APIError) IsRateLimited() bool { return e.Status == statusTooManyRequests }

// Client wraps the Withings Public API.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a Withings API client with sensible defaults.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// newTestClient creates a client pointing at a test server.
func newTestClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// post performs an authenticated form POST and returns the decoded envelope body.
func (c *Client) post(ctx context.Context, path, action, token string, form url.Values) (json.RawMessage, error) {
	form.Set("action", action)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Action: action, Status: resp.StatusCode, Body: string(body)}
	}

	var env Response
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if env.Status != statusOK {
		detail := env.Error
		if detail == "" {
			detail = string(body)
		}
		return nil, &APIError{Action: action, Status: env.Status, Body: detail}
	}
	return env.Body, nil
}

// GetMeasures fetches measurement groups, resolving pagination internally.
//
// lastUpdate > 0 requests everything created or modified since that Unix
// timestamp; otherwise the window from startDate to now is fetched, which is the
// initial backfill. The returned timestamp is the server's `updatetime` and is
// what the next call passes as lastUpdate — deriving it from the local clock
// would make the delta window depend on clock skew.
func (c *Client) GetMeasures(ctx context.Context, token string, measTypes []int, lastUpdate, startDate int64) ([]MeasureGroup, int64, error) {
	types := make([]string, len(measTypes))
	for i, t := range measTypes {
		types[i] = strconv.Itoa(t)
	}

	var (
		groups     []MeasureGroup
		updateTime int64
		offset     int
	)

	for page := 0; page < maxPages; page++ {
		form := url.Values{
			"meastypes": {strings.Join(types, ",")},
			// category 1 is real measurements. Category 2 would return the
			// user's target weight in the same shape as a measurement.
			"category": {"1"},
		}
		if lastUpdate > 0 {
			form.Set("lastupdate", strconv.FormatInt(lastUpdate, 10))
		} else {
			form.Set("startdate", strconv.FormatInt(startDate, 10))
			form.Set("enddate", strconv.FormatInt(time.Now().Unix(), 10))
		}
		if offset > 0 {
			form.Set("offset", strconv.Itoa(offset))
		}

		raw, err := c.post(ctx, "/measure", "getmeas", token, form)
		if err != nil {
			return nil, 0, err
		}

		var mr measureResponse
		if err := json.Unmarshal(raw, &mr); err != nil {
			return nil, 0, fmt.Errorf("decoding getmeas body: %w", err)
		}

		groups = append(groups, mr.MeasureGrps...)
		if mr.UpdateTime > updateTime {
			updateTime = mr.UpdateTime
		}
		if !bool(mr.More) || mr.Offset == 0 {
			return groups, updateTime, nil
		}
		offset = mr.Offset
	}

	return groups, updateTime, fmt.Errorf("getmeas pagination exceeded %d pages", maxPages)
}
