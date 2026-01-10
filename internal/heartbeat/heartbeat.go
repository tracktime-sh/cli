package heartbeat

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrMissingTimestamp = errors.New("missing required field: timestamp")
	ErrMissingEditor    = errors.New("missing required field: editor")
	ErrInvalidTimestamp = errors.New("invalid timestamp format")
)

type Heartbeat struct {
	Timestamp time.Time `json:"timestamp"`
	Editor    string    `json:"editor"`
	Project   string    `json:"project,omitempty"`
	Language  string    `json:"language,omitempty"`
	FilePath  string    `json:"file_path,omitempty"`
	IsWrite   bool      `json:"is_write,omitempty"`
}

type rawHeartbeat struct {
	Timestamp string `json:"timestamp"`
	Editor    string `json:"editor"`
	Project   string `json:"project,omitempty"`
	Language  string `json:"language,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	IsWrite   bool   `json:"is_write,omitempty"`
}

func (h *Heartbeat) UnmarshalJSON(data []byte) error {
	var raw rawHeartbeat
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.Timestamp == "" {
		return ErrMissingTimestamp
	}
	if raw.Editor == "" {
		return ErrMissingEditor
	}

	ts, err := parseTimestamp(raw.Timestamp)
	if err != nil {
		return ErrInvalidTimestamp
	}

	h.Timestamp = ts
	h.Editor = raw.Editor
	h.Project = raw.Project
	h.Language = raw.Language
	h.FilePath = raw.FilePath
	h.IsWrite = raw.IsWrite

	return nil
}

func (h *Heartbeat) MarshalJSON() ([]byte, error) {
	return json.Marshal(&rawHeartbeat{
		Timestamp: h.Timestamp.Format(time.RFC3339Nano),
		Editor:    h.Editor,
		Project:   h.Project,
		Language:  h.Language,
		FilePath:  h.FilePath,
		IsWrite:   h.IsWrite,
	})
}

func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, ErrInvalidTimestamp
}

func Parse(data []byte) (*Heartbeat, error) {
	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return nil, err
	}
	return &hb, nil
}

func ParseMultiple(data []byte) ([]*Heartbeat, error) {
	// Try parsing as array first
	var heartbeats []*Heartbeat
	if err := json.Unmarshal(data, &heartbeats); err == nil {
		return heartbeats, nil
	}

	// Try parsing as single object
	var hb Heartbeat
	if err := json.Unmarshal(data, &hb); err != nil {
		return nil, err
	}

	return []*Heartbeat{&hb}, nil
}

func (h *Heartbeat) Validate() error {
	if h.Timestamp.IsZero() {
		return ErrMissingTimestamp
	}
	if h.Editor == "" {
		return ErrMissingEditor
	}
	return nil
}
