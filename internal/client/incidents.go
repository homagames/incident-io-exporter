package client

import (
	"net/url"
)

// Incident represents an incident from the incident.io API.
type Incident struct {
	ID         string          `json:"id"`
	Reference  string          `json:"reference"`
	Name       string          `json:"name"`
	Mode       string          `json:"mode"`
	Visibility string          `json:"visibility"`
	Status     IncidentStatus  `json:"incident_status"`
	Severity   *Severity       `json:"severity"`
	Type       *IncidentType   `json:"incident_type"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
	Duration   []DurationValue `json:"duration_metrics"`
}

// IncidentStatus represents an incident status.
type IncidentStatus struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Rank     int64  `json:"rank"`
}

// IncidentType represents an incident type.
type IncidentType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DurationValue represents a duration metric attached to an incident.
type DurationValue struct {
	Metric       DurationMetric `json:"duration_metric"`
	ValueSeconds int64          `json:"value_seconds"`
}

// DurationMetric represents the definition of a duration metric.
type DurationMetric struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listIncidentsResponse struct {
	Incidents      []Incident     `json:"incidents"`
	PaginationMeta PaginationMeta `json:"pagination_meta"`
}

// ListIncidents fetches all incidents from the incident.io API.
func (c *Client) ListIncidents() ([]Incident, error) {
	var all []Incident

	err := paginate(func(cursor string) (string, error) {
		params := url.Values{}
		addPaginationParams(params, 250, cursor)

		var resp listIncidentsResponse
		if err := c.get("/v2/incidents", params, &resp); err != nil {
			return "", err
		}

		all = append(all, resp.Incidents...)

		if resp.PaginationMeta.After == "" {
			return "", nil
		}
		return resp.PaginationMeta.After, nil
	})

	return all, err
}
