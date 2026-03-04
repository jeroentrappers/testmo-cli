package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/secutec/testmo-cli/internal/config"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// PaginatedResponse is the standard paginated envelope from the Testmo API.
type PaginatedResponse struct {
	Page     int             `json:"page"`
	PrevPage *int            `json:"prev_page"`
	NextPage *int            `json:"next_page"`
	LastPage int             `json:"last_page"`
	PerPage  int             `json:"per_page"`
	Total    int             `json:"total"`
	Result   json.RawMessage `json:"result"`
	Expands  json.RawMessage `json:"expands,omitempty"`
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.BaseURL(),
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode == 429 {
			wait := 60
			if v := resp.Header.Get("Retry-After"); v != "" {
				if w, err := strconv.Atoi(v); err == nil {
					wait = w
				}
			}
			fmt.Printf("Rate limited, waiting %ds...\n", wait)
			time.Sleep(time.Duration(wait) * time.Second)
			continue
		}

		if resp.StatusCode == 204 {
			return nil, nil
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func (c *Client) Get(path string) ([]byte, error) {
	return c.do("GET", path, nil)
}

func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	return c.do("POST", path, body)
}

func (c *Client) Patch(path string, body interface{}) ([]byte, error) {
	return c.do("PATCH", path, body)
}

func (c *Client) Delete(path string, body interface{}) ([]byte, error) {
	return c.do("DELETE", path, body)
}

// GetAllPages fetches all pages of a paginated endpoint and returns combined results.
func (c *Client) GetAllPages(path string) ([]json.RawMessage, *PaginatedResponse, error) {
	var allResults []json.RawMessage
	var lastResp *PaginatedResponse

	separator := "?"
	if contains(path, "?") {
		separator = "&"
	}

	page := 1
	for {
		url := fmt.Sprintf("%s%spage=%d&per_page=100", path, separator, page)
		data, err := c.Get(url)
		if err != nil {
			return nil, nil, err
		}

		var resp PaginatedResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, nil, fmt.Errorf("unmarshal paginated response: %w", err)
		}
		lastResp = &resp

		// Parse individual items from result array
		var items []json.RawMessage
		if err := json.Unmarshal(resp.Result, &items); err != nil {
			return nil, nil, fmt.Errorf("unmarshal result array: %w", err)
		}
		allResults = append(allResults, items...)

		if resp.NextPage == nil {
			break
		}
		page = *resp.NextPage
	}

	return allResults, lastResp, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
