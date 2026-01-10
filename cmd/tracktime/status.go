package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/config"
	"github.com/tracktime-sh/cli/internal/queue"
)

var statusJSONOutput bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current configuration and queue status",
	Long:  `Displays the current configuration state and queue status.`,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSONOutput, "json", false, "Output in JSON format")
}

type StatusOutput struct {
	Configured  bool    `json:"configured"`
	APIURL      string  `json:"api_url"`
	MachineID   *string `json:"machine_id,omitempty"`
	QueueSize   int     `json:"queue_size"`
	LastSync    *string `json:"last_sync,omitempty"`
	LastSyncRaw *string `json:"last_sync_raw,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		if statusJSONOutput {
			output := StatusOutput{
				Configured: false,
				APIURL:     config.APIURL,
				QueueSize:  0,
			}
			data, _ := json.Marshal(output)
			fmt.Println(string(data))
		} else {
			fmt.Printf("Configured: no (error loading config: %v)\n", err)
		}
		os.Exit(ExitNotConfigured)
	}

	configured := cfg.IsConfigured()

	var queueSize int
	q, err := queue.Open()
	if err == nil {
		defer q.Close()
		count, err := q.Count()
		if err == nil {
			queueSize = count
		}
	}

	if statusJSONOutput {
		output := StatusOutput{
			Configured: configured,
			APIURL:     config.APIURL,
			QueueSize:  queueSize,
		}

		if cfg.MachineID != "" {
			output.MachineID = &cfg.MachineID
		}

		if !cfg.LastFlushAt.IsZero() {
			lastSync := formatTimeAgo(cfg.LastFlushAt)
			lastSyncRaw := cfg.LastFlushAt.Format(time.RFC3339)
			output.LastSync = &lastSync
			output.LastSyncRaw = &lastSyncRaw
		}

		data, _ := json.Marshal(output)
		fmt.Println(string(data))

		if !configured {
			os.Exit(ExitNotConfigured)
		}
		return nil
	}

	configuredStr := "no"
	if configured {
		configuredStr = "yes"
	}

	fmt.Printf("Configured: %s\n", configuredStr)
	fmt.Printf("API URL:    %s\n", config.APIURL)

	if cfg.MachineID != "" {
		machineID := cfg.MachineID
		if len(machineID) > 8 {
			machineID = machineID[:8] + "..."
		}
		fmt.Printf("Machine ID: %s\n", machineID)
	}

	if q != nil {
		fmt.Printf("Queue size: %d heartbeats\n", queueSize)
	}

	if !cfg.LastFlushAt.IsZero() {
		fmt.Printf("Last sync:  %s\n", formatTimeAgo(cfg.LastFlushAt))
	} else {
		fmt.Printf("Last sync:  never\n")
	}

	if !configured {
		fmt.Println("\nTo configure, run: tracktime login")
		os.Exit(ExitNotConfigured)
	}

	return nil
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
