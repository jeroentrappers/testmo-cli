package api

import (
	"encoding/json"
	"fmt"
)

type Case struct {
	ID                  int     `json:"id"`
	Key                 int     `json:"key"`
	Name                string  `json:"name"`
	ProjectID           int     `json:"project_id"`
	RepoID              int     `json:"repo_id"`
	FolderID            int     `json:"folder_id"`
	TemplateID          int     `json:"template_id"`
	StateID             int     `json:"state_id"`
	StatusID            *int    `json:"status_id"`
	StatusAt            *string `json:"status_at"`
	Estimate            *int    `json:"estimate"`
	Forecast            *int    `json:"forecast"`
	HasAutomation       bool    `json:"has_automation"`
	HasAutomationStatus bool    `json:"has_automation_status"`
	CreatedAt           string  `json:"created_at"`
	CreatedBy           int     `json:"created_by"`
	UpdatedAt           *string `json:"updated_at"`
	UpdatedBy           *int    `json:"updated_by"`
	CustomPriority      *int    `json:"custom_priority,omitempty"`
	CustomDescription   *string `json:"custom_bdddescription,omitempty"`
}

type CreateCaseRequest struct {
	Cases []CreateCase `json:"cases"`
}

type CreateCase struct {
	Name            string   `json:"name"`
	FolderID        *int     `json:"folder_id,omitempty"`
	TemplateID      *int     `json:"template_id,omitempty"`
	StateID         *int     `json:"state_id,omitempty"`
	Estimate        *int     `json:"estimate,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	AutomationLinks []int    `json:"automation_links,omitempty"`
}

type UpdateCaseRequest struct {
	IDs             []int    `json:"ids"`
	Name            *string  `json:"name,omitempty"`
	FolderID        *int     `json:"folder_id,omitempty"`
	StateID         *int     `json:"state_id,omitempty"`
	StatusID        *int     `json:"status_id,omitempty"`
	Estimate        *int     `json:"estimate,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	AutomationLinks []int    `json:"automation_links,omitempty"`
}

type DeleteCaseRequest struct {
	IDs []int `json:"ids"`
}

type CreateCaseResponse struct {
	Result []Case `json:"result"`
}

func (c *Client) ListCases(projectID int, folderID *int) ([]Case, error) {
	path := fmt.Sprintf("/projects/%d/cases", projectID)
	if folderID != nil {
		path += fmt.Sprintf("?folder_id=%d", *folderID)
	}

	items, _, err := c.GetAllPages(path)
	if err != nil {
		return nil, err
	}

	var cases []Case
	for _, raw := range items {
		var tc Case
		if err := json.Unmarshal(raw, &tc); err != nil {
			return nil, fmt.Errorf("unmarshal case: %w", err)
		}
		cases = append(cases, tc)
	}
	return cases, nil
}

func (c *Client) CreateCases(projectID int, cases []CreateCase) ([]Case, error) {
	var allCreated []Case

	// Batch in chunks of 100
	for i := 0; i < len(cases); i += 100 {
		end := i + 100
		if end > len(cases) {
			end = len(cases)
		}
		batch := cases[i:end]

		req := CreateCaseRequest{Cases: batch}
		data, err := c.Post(fmt.Sprintf("/projects/%d/cases", projectID), req)
		if err != nil {
			return allCreated, fmt.Errorf("create cases batch %d-%d: %w", i, end, err)
		}

		var resp CreateCaseResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allCreated, fmt.Errorf("unmarshal create response: %w", err)
		}
		allCreated = append(allCreated, resp.Result...)
	}

	return allCreated, nil
}

func (c *Client) UpdateCases(projectID int, req UpdateCaseRequest) error {
	// Batch IDs in chunks of 100
	for i := 0; i < len(req.IDs); i += 100 {
		end := i + 100
		if end > len(req.IDs) {
			end = len(req.IDs)
		}
		batchReq := req
		batchReq.IDs = req.IDs[i:end]

		_, err := c.Patch(fmt.Sprintf("/projects/%d/cases", projectID), batchReq)
		if err != nil {
			return fmt.Errorf("update cases batch %d-%d: %w", i, end, err)
		}
	}
	return nil
}

func (c *Client) DeleteCases(projectID int, ids []int) error {
	for i := 0; i < len(ids); i += 100 {
		end := i + 100
		if end > len(ids) {
			end = len(ids)
		}
		req := DeleteCaseRequest{IDs: ids[i:end]}
		_, err := c.Delete(fmt.Sprintf("/projects/%d/cases", projectID), req)
		if err != nil {
			return fmt.Errorf("delete cases batch %d-%d: %w", i, end, err)
		}
	}
	return nil
}
