package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIncidentStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/incident_statuses", r.URL.Path)
		json.NewEncoder(w).Encode(listIncidentStatusesResponse{
			IncidentStatuses: []IncidentStatus{
				{ID: "st-1", Name: "Triage", Category: "triage", Rank: 0},
				{ID: "st-2", Name: "Active", Category: "live", Rank: 1},
				{ID: "st-3", Name: "Closed", Category: "closed", Rank: 2},
			},
		})
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	statuses, err := c.ListIncidentStatuses()
	require.NoError(t, err)
	require.Len(t, statuses, 3)
	assert.Equal(t, "Triage", statuses[0].Name)
	assert.Equal(t, "closed", statuses[2].Category)
}

func TestListIncidentStatuses_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	_, err := c.ListIncidentStatuses()
	require.Error(t, err)
}
