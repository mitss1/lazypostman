package postman

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mitss1/lazypostman/internal/collection"
	"github.com/mitss1/lazypostman/internal/environment"
)

const baseURL = "https://api.getpostman.com"

// Client communicates with the Postman API
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a Postman API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Workspace represents a Postman workspace
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// CollectionInfo represents a collection listing entry
type CollectionInfo struct {
	ID   string `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
}

// EnvironmentInfo represents an environment listing entry
type EnvironmentInfo struct {
	ID   string `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
}

// UserInfo represents the current user
type UserInfo struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

func (c *Client) doGet(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("invalid API key (401 Unauthorized)")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetMe returns the current user info (useful to verify API key)
func (c *Client) GetMe() (*UserInfo, error) {
	data, err := c.doGet("/me")
	if err != nil {
		return nil, err
	}

	var resp struct {
		User UserInfo `json:"user"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing user: %w", err)
	}

	return &resp.User, nil
}

// ListWorkspaces returns all workspaces
func (c *Client) ListWorkspaces() ([]Workspace, error) {
	data, err := c.doGet("/workspaces")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Workspaces []Workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing workspaces: %w", err)
	}

	return resp.Workspaces, nil
}

// ListCollections returns all collections
func (c *Client) ListCollections() ([]CollectionInfo, error) {
	data, err := c.doGet("/collections")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Collections []struct {
			ID   string `json:"id"`
			UID  string `json:"uid"`
			Name string `json:"name"`
		} `json:"collections"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing collections: %w", err)
	}

	result := make([]CollectionInfo, len(resp.Collections))
	for i, c := range resp.Collections {
		result[i] = CollectionInfo{ID: c.ID, UID: c.UID, Name: c.Name}
	}

	return result, nil
}

// GetCollection fetches a full collection by UID
func (c *Client) GetCollection(uid string) (*collection.Collection, error) {
	data, err := c.doGet("/collections/" + uid)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Collection json.RawMessage `json:"collection"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing collection wrapper: %w", err)
	}

	var col collection.Collection
	if err := json.Unmarshal(resp.Collection, &col); err != nil {
		return nil, fmt.Errorf("parsing collection: %w", err)
	}

	return &col, nil
}

// ListEnvironments returns all environments.
// Falls back to scanning workspaces if the global endpoint returns empty.
func (c *Client) ListEnvironments() ([]EnvironmentInfo, error) {
	data, err := c.doGet("/environments")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Environments []struct {
			ID   string `json:"id"`
			UID  string `json:"uid"`
			Name string `json:"name"`
		} `json:"environments"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing environments: %w", err)
	}

	result := make([]EnvironmentInfo, len(resp.Environments))
	for i, e := range resp.Environments {
		result[i] = EnvironmentInfo{ID: e.ID, UID: e.UID, Name: e.Name}
	}

	// If no environments found, try workspace-scoped lookup
	if len(result) == 0 {
		workspaces, err := c.ListWorkspaces()
		if err != nil {
			return result, nil // return empty, don't fail
		}
		for _, ws := range workspaces {
			wsData, err := c.doGet("/environments?workspace=" + ws.ID)
			if err != nil {
				continue
			}
			var wsResp struct {
				Environments []struct {
					ID   string `json:"id"`
					UID  string `json:"uid"`
					Name string `json:"name"`
				} `json:"environments"`
			}
			if err := json.Unmarshal(wsData, &wsResp); err != nil {
				continue
			}
			for _, e := range wsResp.Environments {
				result = append(result, EnvironmentInfo{
					ID:   e.ID,
					UID:  e.UID,
					Name: fmt.Sprintf("%s (%s)", e.Name, ws.Name),
				})
			}
		}
	}

	return result, nil
}

// GetEnvironment fetches a full environment by UID
func (c *Client) GetEnvironment(uid string) (*environment.Environment, error) {
	data, err := c.doGet("/environments/" + uid)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Environment json.RawMessage `json:"environment"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing environment wrapper: %w", err)
	}

	var env environment.Environment
	if err := json.Unmarshal(resp.Environment, &env); err != nil {
		return nil, fmt.Errorf("parsing environment: %w", err)
	}

	return &env, nil
}
