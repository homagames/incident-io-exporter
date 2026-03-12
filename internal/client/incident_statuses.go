package client

type listIncidentStatusesResponse struct {
	IncidentStatuses []IncidentStatus `json:"incident_statuses"`
}

// ListIncidentStatuses fetches all configured incident statuses.
func (c *Client) ListIncidentStatuses() ([]IncidentStatus, error) {
	var resp listIncidentStatusesResponse
	if err := c.get("/v1/incident_statuses", nil, &resp); err != nil {
		return nil, err
	}
	return resp.IncidentStatuses, nil
}
