package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

var (
	APIURL  = "http://localhost:3000/api/v1"
	WebURL  = "http://localhost:3000"
	Version = "0.1.0-dev"
)

const (
	DefaultFlushIntervalSeconds = 120
)

type Config struct {
	APIKey               string    `json:"api_key"`
	MachineID            string    `json:"machine_id"`
	FlushIntervalSeconds int       `json:"flush_interval_seconds"`
	LastFlushAt          time.Time `json:"last_flush_at"`
}

func DefaultConfig() *Config {
	return &Config{
		MachineID:            uuid.New().String(),
		FlushIntervalSeconds: DefaultFlushIntervalSeconds,
	}
}

func Dir() (string, error) {
	var baseDir string

	if runtime.GOOS == "windows" {
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			baseDir = os.Getenv("USERPROFILE")
		}
	} else {
		baseDir = os.Getenv("HOME")
	}

	if baseDir == "" {
		return "", os.ErrNotExist
	}

	return filepath.Join(baseDir, ".tracktime"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.MachineID == "" {
		cfg.MachineID = uuid.New().String()
	}
	if cfg.FlushIntervalSeconds == 0 {
		cfg.FlushIntervalSeconds = DefaultFlushIntervalSeconds
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	path, err := Path()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func (c *Config) IsConfigured() bool {
	return c.APIKey != ""
}

func (c *Config) ShouldFlush() bool {
	if c.LastFlushAt.IsZero() {
		return true
	}
	return time.Since(c.LastFlushAt) > time.Duration(c.FlushIntervalSeconds)*time.Second
}

func (c *Config) UpdateLastFlush() {
	c.LastFlushAt = time.Now()
}

func (c *Config) ClearAuth() {
	c.APIKey = ""
}
