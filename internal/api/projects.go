package api

import (
	"encoding/json"
	"fmt"
)

type Project struct {
	ID                       int     `json:"id"`
	Name                     string  `json:"name"`
	Note                     *string `json:"note"`
	IsCompleted              bool    `json:"is_completed"`
	RunCount                 int     `json:"run_count"`
	AutomationRunCount       int     `json:"automation_run_count"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                *string `json:"updated_at"`
}

func (c *Client) ListProjects() ([]Project, error) {
	items, _, err := c.GetAllPages("/projects")
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, raw := range items {
		var p Project
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("unmarshal project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, nil
}
