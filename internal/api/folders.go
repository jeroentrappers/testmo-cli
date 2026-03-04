package api

import (
	"encoding/json"
	"fmt"
)

type Folder struct {
	ID           int     `json:"id"`
	ProjectID    int     `json:"project_id"`
	RepoID       int     `json:"repo_id"`
	ParentID     *int    `json:"parent_id"`
	Depth        int     `json:"depth"`
	Name         string  `json:"name"`
	Docs         *string `json:"docs"`
	DisplayOrder int     `json:"display_order"`
}

type CreateFolderRequest struct {
	Folders []CreateFolder `json:"folders"`
}

type CreateFolder struct {
	Name         string  `json:"name"`
	ParentID     *int    `json:"parent_id,omitempty"`
	Docs         *string `json:"docs,omitempty"`
	DisplayOrder *int    `json:"display_order,omitempty"`
}

type UpdateFolderRequest struct {
	IDs      []int   `json:"ids"`
	Name     *string `json:"name,omitempty"`
	ParentID *int    `json:"parent_id,omitempty"`
	Docs     *string `json:"docs,omitempty"`
}

type DeleteFolderRequest struct {
	IDs []int `json:"ids"`
}

type CreateFolderResponse struct {
	Result []Folder `json:"result"`
}

func (c *Client) ListFolders(projectID int) ([]Folder, error) {
	items, _, err := c.GetAllPages(fmt.Sprintf("/projects/%d/folders", projectID))
	if err != nil {
		return nil, err
	}

	var folders []Folder
	for _, raw := range items {
		var f Folder
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("unmarshal folder: %w", err)
		}
		folders = append(folders, f)
	}
	return folders, nil
}

func (c *Client) CreateFolders(projectID int, folders []CreateFolder) ([]Folder, error) {
	req := CreateFolderRequest{Folders: folders}
	data, err := c.Post(fmt.Sprintf("/projects/%d/folders", projectID), req)
	if err != nil {
		return nil, err
	}
	var resp CreateFolderResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal create folder response: %w", err)
	}
	return resp.Result, nil
}

func (c *Client) UpdateFolders(projectID int, req UpdateFolderRequest) error {
	_, err := c.Patch(fmt.Sprintf("/projects/%d/folders", projectID), req)
	return err
}

func (c *Client) DeleteFolders(projectID int, ids []int) error {
	req := DeleteFolderRequest{IDs: ids}
	_, err := c.Delete(fmt.Sprintf("/projects/%d/folders", projectID), req)
	return err
}
