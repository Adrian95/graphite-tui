package vercel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

	endpoint := fmt.Sprintf("%s/v13/deployments", apiBase)
	params := url.Values{}
	params.Set("projectId", c.config.ProjectID)
	params.Set("limit", strconv.Itoa(limit))
	if target != "" {
		params.Set("target", target)
	}
	if c.config.TeamID != "" {
		params.Set("teamId", c.config.TeamID)
	}

	endpoint = endpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", endpoint, nil)
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
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = "unknown error"
		}
		return nil, fmt.Errorf("vercel api returned %d: %s", resp.StatusCode, msg)
	}

	var data DeploymentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
