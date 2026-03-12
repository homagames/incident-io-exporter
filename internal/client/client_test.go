package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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

func TestGet_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	var result map[string]interface{}
	err := c.get("/v1/test", nil, &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
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
			return "next", nil
		}
		return "", nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
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
