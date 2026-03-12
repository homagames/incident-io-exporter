package client

import (
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
	defaultBaseURL = "https://api.incident.io"
	userAgent      = "incident-io-exporter"
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
		},
	}
}

// SetBaseURL overrides the default base URL (useful for testing).
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// get performs a GET request and decodes the JSON response.
func (c *Client) get(path string, params url.Values, target interface{}) error {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		return fmt.Errorf("rate limited, retry after: %s", retryAfter)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// paginate fetches all pages for a list endpoint.
// fetchPage should perform the API call with the given cursor and return the next cursor (empty if done).
func paginate(fetchPage func(cursor string) (string, error)) error {
	cursor := ""
	for {
		nextCursor, err := fetchPage(cursor)
		if err != nil {
			return err
		}
		if nextCursor == "" {
			return nil
		}
		cursor = nextCursor
		slog.Debug("paginating", "next_cursor", cursor)
	}
}

// addPaginationParams adds page_size and after params to a url.Values.
func addPaginationParams(params url.Values, pageSize int, cursor string) {
	params.Set("page_size", strconv.Itoa(pageSize))
	if cursor != "" {
		params.Set("after", cursor)
	}
}
