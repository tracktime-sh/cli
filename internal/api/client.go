package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tracktime-sh/cli/internal/config"
	"github.com/tracktime-sh/cli/internal/heartbeat"
)

const (
	DefaultTimeout = 10 * time.Second
	UserAgent      = "tracktime-cli/0.1.0"
)

var (
	ErrUnauthorized = errors.New("unauthorized: invalid API key")
	ErrBadRequest   = errors.New("bad request: malformed payload")
	ErrServerError  = errors.New("server error")
	ErrNetwork      = errors.New("network error")
)

type Client struct {
	apiKey     string
	machineID  string
	httpClient *http.Client
}

type SendHeartbeatsResponse struct {
	Accepted int `json:"accepted"`
}

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func NewClient(apiKey, machineID string) *Client {
	return &Client{
		apiKey:    apiKey,
		machineID: machineID,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

type apiHeartbeat struct {
	Timestamp string `json:"timestamp"`
	Editor    string `json:"editor"`
	Project   string `json:"project,omitempty"`
	Language  string `json:"language,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	IsWrite   bool   `json:"is_write,omitempty"`
	Machine   string `json:"machine,omitempty"`
}

func (c *Client) SendHeartbeats(heartbeats []*heartbeat.Heartbeat) (*SendHeartbeatsResponse, error) {
	if len(heartbeats) == 0 {
		return &SendHeartbeatsResponse{Accepted: 0}, nil
	}

	apiHeartbeats := make([]apiHeartbeat, len(heartbeats))
	for i, hb := range heartbeats {
		apiHeartbeats[i] = apiHeartbeat{
			Timestamp: hb.Timestamp.Format(time.RFC3339Nano),
			Editor:    hb.Editor,
			Project:   hb.Project,
			Language:  hb.Language,
			FilePath:  hb.FilePath,
			IsWrite:   hb.IsWrite,
			Machine:   c.machineID,
		}
	}

	payload, err := json.Marshal(apiHeartbeats)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal heartbeats: %w", err)
	}

	req, err := http.NewRequest("POST", config.APIURL+"/heartbeats", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if c.machineID != "" {
		req.Header.Set("X-Machine-ID", c.machineID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusCreated:
		var result SendHeartbeatsResponse
		if err := json.Unmarshal(body, &result); err != nil {
			result.Accepted = len(heartbeats)
		}
		return &result, nil

	case http.StatusUnauthorized:
		return nil, ErrUnauthorized

	case http.StatusBadRequest:
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, string(body))

	default:
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: status %d", ErrServerError, resp.StatusCode)
		}
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func (c *Client) GetUser() (*User, error) {
	req, err := http.NewRequest("GET", config.APIURL+"/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		var user User
		if err := json.Unmarshal(body, &user); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return &user, nil

	case http.StatusUnauthorized:
		return nil, ErrUnauthorized

	default:
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: status %d", ErrServerError, resp.StatusCode)
		}
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}
