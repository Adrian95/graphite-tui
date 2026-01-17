package vercel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const apiBase = "https://api.vercel.com"

// Client handles Vercel API requests
// Uses REST API (no CLI dependency)
// One repo → one project
type Client struct {
	config Config
	client *http.Client
}

// NewClient returns a new Vercel API client
func NewClient(cfg Config) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

// GetDeployments returns latest deployments for the configured project
func (c *Client) GetDeployments(limit int, target string) (*DeploymentsResponse, error) {
	if !c.config.Enabled() {
		return &DeploymentsResponse{}, nil
	}

	url := fmt.Sprintf("%s/v13/deployments?projectId=%s&limit=%d", apiBase, c.config.ProjectID, limit)
	if target != "" {
		url += "&target=" + target
	}
	if c.config.TeamID != "" {
		url += "&teamId=" + c.config.TeamID
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("vercel api returned %d", resp.StatusCode)
	}

	var data DeploymentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
