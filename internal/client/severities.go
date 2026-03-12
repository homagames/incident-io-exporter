package client

// Severity represents a severity level from the incident.io API.
type Severity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Rank        int64  `json:"rank"`
}

type listSeveritiesResponse struct {
	Severities []Severity `json:"severities"`
}

// ListSeverities fetches all configured severities.
func (c *Client) ListSeverities() ([]Severity, error) {
	var resp listSeveritiesResponse
	if err := c.get("/v1/severities", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Severities, nil
}
