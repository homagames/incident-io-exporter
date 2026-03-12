package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAlerts_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/alerts", r.URL.Path)
		assert.Equal(t, "50", r.URL.Query().Get("page_size"))
		json.NewEncoder(w).Encode(listAlertsResponse{
			Alerts: []Alert{
				{ID: "alert-1", Status: "firing", Title: "Test Alert"},
				{ID: "alert-2", Status: "resolved", Title: "Resolved Alert"},
			},
			PaginationMeta: PaginationMeta{PageSize: 50},
		})
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	alerts, err := c.ListAlerts()
	require.NoError(t, err)
	require.Len(t, alerts, 2)
	assert.Equal(t, "firing", alerts[0].Status)
	assert.Equal(t, "resolved", alerts[1].Status)
}

func TestListAlerts_MultiplePages(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := listAlertsResponse{
			Alerts: []Alert{{ID: "alert"}},
		}
		if callCount < 2 {
			resp.PaginationMeta.After = "next"
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	alerts, err := c.ListAlerts()
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
}

func TestListAlerts_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	c := New("key")
	c.SetBaseURL(srv.URL)

	_, err := c.ListAlerts()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
