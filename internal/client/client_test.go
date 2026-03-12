package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	c := New("test-key")
	assert.Equal(t, defaultBaseURL, c.baseURL)
	assert.Equal(t, "test-key", c.apiKey)
	assert.NotNil(t, c.httpClient)
}

func TestSetBaseURL(t *testing.T) {
	c := New("key")
	c.SetBaseURL("http://localhost")
	assert.Equal(t, "http://localhost", c.baseURL)
}

func TestGet_Success(t *testing.T) {
	expected := map[string]string{"hello": "world"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
		assert.Equal(t, "/v1/test", r.URL.Path)
		assert.Equal(t, "bar", r.URL.Query().Get("foo"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := New("test-key")
	c.SetBaseURL(srv.URL)

	var result map[string]string
	err := c.get("/v1/test", url.Values{"foo": {"bar"}}, &result)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGet_RateLimitRetry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// Succeed on third attempt
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	var result map[string]string
	err := c.get("/v1/test", nil, &result)
	require.NoError(t, err)
	assert.Equal(t, "true", result["ok"])
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestGet_RateLimitExhausted(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	var result map[string]interface{}
	err := c.get("/v1/test", nil, &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited after")
	assert.Equal(t, int32(maxRetries+1), atomic.LoadInt32(&attempts))
}

func TestGet_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	var result map[string]interface{}
	err := c.get("/v1/test", nil, &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 500")
}

func TestGet_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	var result map[string]interface{}
	err := c.get("/v1/test", nil, &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding response")
}

func TestPaginate(t *testing.T) {
	callCount := 0
	err := paginate(func(cursor string) (string, error) {
		callCount++
		if callCount < 3 {
			return "next-" + string(rune('0'+callCount)), nil
		}
		return "", nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestPaginate_StuckCursor(t *testing.T) {
	err := paginate(func(cursor string) (string, error) {
		return "same-cursor", nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination stuck")
}

func TestParseRetryAfter(t *testing.T) {
	// Valid header
	assert.Equal(t, 5*time.Second, parseRetryAfter("5", 0))
	// Empty header — exponential backoff
	assert.Equal(t, 1*time.Second, parseRetryAfter("", 0))
	assert.Equal(t, 2*time.Second, parseRetryAfter("", 1))
	assert.Equal(t, 4*time.Second, parseRetryAfter("", 2))
	// Invalid header
	assert.Equal(t, 1*time.Second, parseRetryAfter("abc", 0))
}

func TestAddPaginationParams(t *testing.T) {
	params := url.Values{}
	addPaginationParams(params, 100, "cursor123")
	assert.Equal(t, "100", params.Get("page_size"))
	assert.Equal(t, "cursor123", params.Get("after"))
}

func TestAddPaginationParams_NoCursor(t *testing.T) {
	params := url.Values{}
	addPaginationParams(params, 50, "")
	assert.Equal(t, "50", params.Get("page_size"))
	assert.Equal(t, "", params.Get("after"))
}
