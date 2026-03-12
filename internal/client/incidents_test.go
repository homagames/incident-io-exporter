package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIncidents_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/incidents", r.URL.Path)
		assert.Equal(t, "250", r.URL.Query().Get("page_size"))
		json.NewEncoder(w).Encode(listIncidentsResponse{
			Incidents: []Incident{
				{
					ID:   "inc-1",
					Name: "Test Incident",
					Mode: "standard",
					Status: IncidentStatus{
						ID:       "status-1",
						Name:     "Closed",
						Category: "closed",
					},
					Severity: &Severity{
						ID:   "sev-1",
						Name: "Critical",
						Rank: 1,
					},
				},
			},
			PaginationMeta: PaginationMeta{
				PageSize: 250,
				After:    "",
			},
		})
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	incidents, err := c.ListIncidents()
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	assert.Equal(t, "inc-1", incidents[0].ID)
	assert.Equal(t, "Closed", incidents[0].Status.Name)
	assert.Equal(t, "Critical", incidents[0].Severity.Name)
}

func TestListIncidents_MultiplePages(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := listIncidentsResponse{
			Incidents: []Incident{
				{ID: "inc-" + r.URL.Query().Get("after")},
			},
		}
		if callCount < 3 {
			resp.PaginationMeta.After = "cursor-" + string(rune('0'+callCount))
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	incidents, err := c.ListIncidents()
	require.NoError(t, err)
	assert.Len(t, incidents, 3)
	assert.Equal(t, 3, callCount)
}

func TestListIncidents_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	_, err := c.ListIncidents()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}
