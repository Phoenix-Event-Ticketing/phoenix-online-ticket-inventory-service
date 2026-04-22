package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EventServiceClient checks event existence through the event service API.
type EventServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewEventServiceClient creates an event service HTTP client.
func NewEventServiceClient(baseURL string, timeout time.Duration) (*EventServiceClient, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("event service base URL is required")
	}
	if _, err := url.ParseRequestURI(trimmed); err != nil {
		return nil, fmt.Errorf("invalid event service URL: %w", err)
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &EventServiceClient{
		baseURL:    trimmed,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// EventExists returns true when the event service confirms the event id.
func (c *EventServiceClient) EventExists(ctx context.Context, eventID string) (bool, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return false, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/events/"+url.PathEscape(eventID), nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return true, nil
	case resp.StatusCode == http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("event service returned status %d", resp.StatusCode)
	}
}
