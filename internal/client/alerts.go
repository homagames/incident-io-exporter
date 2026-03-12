package client

import (
	"net/url"
)

// Alert represents an alert from the incident.io API.
type Alert struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type listAlertsResponse struct {
	Alerts         []Alert        `json:"alerts"`
	PaginationMeta PaginationMeta `json:"pagination_meta"`
}

// ListAlerts fetches all alerts from the incident.io API.
func (c *Client) ListAlerts() ([]Alert, error) {
	var all []Alert

	err := paginate(func(cursor string) (string, error) {
		params := url.Values{}
		addPaginationParams(params, 50, cursor)

		var resp listAlertsResponse
		if err := c.get("/v2/alerts", params, &resp); err != nil {
			return "", err
		}

		all = append(all, resp.Alerts...)

		if resp.PaginationMeta.After == "" {
			return "", nil
		}
		return resp.PaginationMeta.After, nil
	})

	return all, err
}
