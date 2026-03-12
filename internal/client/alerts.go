package client

import (
	"net/url"
)

// AlertAttribute represents an attribute definition on an alert.
type AlertAttribute struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// AlertAttributeValue represents the value of an alert attribute.
type AlertAttributeValue struct {
	Literal string `json:"literal"`
	Label   string `json:"label"`
}

// AlertAttributeEntry represents an attribute entry on an alert.
type AlertAttributeEntry struct {
	Attribute AlertAttribute       `json:"attribute"`
	Value     *AlertAttributeValue `json:"value,omitempty"`
}

// Alert represents an alert from the incident.io API.
type Alert struct {
	ID         string                `json:"id"`
	Status     string                `json:"status"`
	Title      string                `json:"title"`
	Attributes []AlertAttributeEntry `json:"attributes"`
	CreatedAt  string                `json:"created_at"`
	UpdatedAt  string                `json:"updated_at"`
}

// Team returns the team name from the alert's attributes, or empty string if not set.
func (a *Alert) Team() string {
	for _, attr := range a.Attributes {
		if attr.Attribute.Name == "Team" && attr.Value != nil {
			return attr.Value.Label
		}
	}
	return ""
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
