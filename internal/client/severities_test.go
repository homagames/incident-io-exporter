package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSeverities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/severities", r.URL.Path)
		json.NewEncoder(w).Encode(listSeveritiesResponse{
			Severities: []Severity{
				{ID: "sev-1", Name: "Critical", Rank: 1},
				{ID: "sev-2", Name: "Major", Rank: 2},
				{ID: "sev-3", Name: "Minor", Rank: 3},
			},
		})
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	sevs, err := c.ListSeverities()
	require.NoError(t, err)
	require.Len(t, sevs, 3)
	assert.Equal(t, "Critical", sevs[0].Name)
	assert.Equal(t, int64(1), sevs[0].Rank)
}

func TestListSeverities_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	_, err := c.ListSeverities()
	require.Error(t, err)
}
