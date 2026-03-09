// Package source defines the common interface and types for activity data sources.
package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"resty.dev/v3"
)

// DataSource represents a provider of activity data (e.g. Todoist, Slack).
type DataSource interface {
	Name() string
	Fetch(ctx context.Context, tr TimeRange) ([]DataItem, error)
}

// TimeRange represents a bounded window of time with a human-readable label.
type TimeRange struct {
	Start time.Time
	End   time.Time
	Label string // "day", "week", "month", "quarter"
}

// NewTimeRange creates a TimeRange ending now with the given label.
func NewTimeRange(label string) TimeRange {
	now := time.Now()
	var start time.Time

	switch label {
	case "day":
		start = now.AddDate(0, 0, -1)
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	case "quarter":
		start = now.AddDate(0, -3, 0)
	default:
		start = now.AddDate(0, 0, -1)
		label = "day"
	}

	return TimeRange{Start: start, End: now, Label: label}
}

// DataItem represents a single piece of activity data from any source.
type DataItem struct {
	ID       string
	Source   string
	Type     string // "task_completed", "task_created", "task_updated", "conversation", "journal"
	Title    string
	Content  string
	Metadata map[string]string
	// Timestamp is when the activity occurred.
	Timestamp time.Time
}

// RestyRequestLogger logs the request method and URL at the TRACE level for resty requests.
func RestyRequestLogger(log zerolog.Logger) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		log.Trace().
			Str("method", req.Method).
			Str("url", req.URL).
			Func(func(e *zerolog.Event) {
				if req.Body == nil {
					return
				}
				if b, err := json.Marshal(req.Body); err == nil {
					e.RawJSON("body", b)
				} else {
					e.Str("body", fmt.Sprintf("%v", req.Body))
				}
			}).
			Msg("request")
		return nil
	}
}

// RestyResponseLogger logs the response status and URL at the TRACE level for resty responses.
func RestyResponseLogger(log zerolog.Logger) resty.ResponseMiddleware {
	return func(_ *resty.Client, resp *resty.Response) error {
		log.Trace().
			Int("status", resp.StatusCode()).
			Str("url", resp.Request.URL).
			Str("duration", resp.Duration().String()).
			Func(func(e *zerolog.Event) {
				body := resp.Bytes()
				if len(body) == 0 {
					return
				}
				if json.Valid(body) {
					e.RawJSON("body", body)
				} else {
					e.Str("body", string(body))
				}
			}).
			Msg("response")
		return nil
	}
}

// NewLoggingClient creates an http.Client that logs requests and responses at the TRACE level.
func NewLoggingClient(log zerolog.Logger) http.Client {
	return http.Client{
		Transport: &LoggingRoundTripper{
			roundTripper: http.DefaultTransport,
			log:          log,
		},
	}
}

// LoggingRoundTripper wraps an http.RoundTripper to log requests and responses at the TRACE level.
type LoggingRoundTripper struct {
	roundTripper http.RoundTripper
	log          zerolog.Logger
}

// RoundTrip executes the HTTP request while logging the request and response at TRACE level.
func (l *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	l.log.Trace().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Func(func(e *zerolog.Event) {
			if req.Body == nil {
				return
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return
			}
			req.Body = io.NopCloser(bytes.NewReader(body))
			if json.Valid(body) {
				e.RawJSON("body", body)
			} else {
				e.Str("body", string(body))
			}
		}).
		Msg("request")

	startTime := time.Now()
	resp, err := l.roundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	l.log.Trace().
		Int("status", resp.StatusCode).
		Str("url", resp.Request.URL.String()).
		Str("duration", time.Since(startTime).String()).
		Func(func(e *zerolog.Event) {
			if resp.Body == nil {
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}
			resp.Body = io.NopCloser(bytes.NewReader(body))
			if json.Valid(body) {
				e.RawJSON("body", body)
			} else {
				e.Str("body", string(body))
			}
		}).
		Msg("response")
	return resp, nil
}
