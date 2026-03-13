package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultBaseURL     = "https://api.incident.io"
	userAgent          = "incident-io-exporter"
	maxRetries         = 3
	maxPaginationPages = 500
)

// Client is an HTTP client for the incident.io API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// PaginationMeta represents the pagination metadata returned by incident.io.
type PaginationMeta struct {
	PageSize         int    `json:"page_size"`
	After            string `json:"after"`
	TotalRecordCount int    `json:"total_record_count"`
}

// New creates a new incident.io API client.
func New(apiKey string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				MaxConnsPerHost:     10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// SetBaseURL overrides the default base URL (useful for testing).
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// get performs a GET request and decodes the JSON response.
// It retries on 429 (rate limit) responses with exponential backoff.
func (c *Client) get(path string, params url.Values, target interface{}) error {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			cancel()
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			cancel()
			return fmt.Errorf("executing request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			cancel()

			if attempt == maxRetries {
				return fmt.Errorf("rate limited after %d retries", maxRetries)
			}

			wait := parseRetryAfter(resp.Header.Get("Retry-After"), attempt)
			slog.Warn("rate limited, retrying", "attempt", attempt+1, "wait", wait)
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			cancel()
			return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}

		err = json.NewDecoder(resp.Body).Decode(target)
		resp.Body.Close()
		cancel()
		if err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts", maxRetries+1)
}

// parseRetryAfter parses the Retry-After header value as seconds.
// Falls back to exponential backoff if the header is missing or unparseable.
func parseRetryAfter(header string, attempt int) time.Duration {
	if header != "" {
		if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	// Exponential backoff: 1s, 2s, 4s
	return time.Duration(1<<uint(attempt)) * time.Second
}

// paginate fetches all pages for a list endpoint.
// fetchPage should perform the API call with the given cursor and return the next cursor (empty if done).
func paginate(fetchPage func(cursor string) (string, error)) error {
	cursor := ""
	for page := 0; page < maxPaginationPages; page++ {
		nextCursor, err := fetchPage(cursor)
		if err != nil {
			return err
		}
		if nextCursor == "" {
			return nil
		}
		if nextCursor == cursor {
			return fmt.Errorf("pagination stuck: cursor did not advance (cursor=%q)", cursor)
		}
		cursor = nextCursor
		slog.Debug("paginating", "page", page+1, "next_cursor", cursor)
	}
	return fmt.Errorf("pagination exceeded maximum of %d pages", maxPaginationPages)
}

// addPaginationParams adds page_size and after params to a url.Values.
func addPaginationParams(params url.Values, pageSize int, cursor string) {
	params.Set("page_size", strconv.Itoa(pageSize))
	if cursor != "" {
		params.Set("after", cursor)
	}
}
