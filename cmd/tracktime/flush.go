package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/api"
	"github.com/tracktime-sh/cli/internal/config"
	"github.com/tracktime-sh/cli/internal/queue"
)

const (
	FlushExitSuccess      = 0
	FlushExitPartial      = 1
	FlushExitNotConfig    = 2
	FlushExitNetworkError = 3
)

type FlushResult struct {
	Synced    int    `json:"synced"`
	Remaining int    `json:"remaining"`
	Errors    int    `json:"errors"`
	Error     string `json:"error,omitempty"`
}

var flushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Sync queued heartbeats to the server",
	Long:  `Forces a sync of all queued heartbeats to the backend server.`,
	RunE:  runFlush,
}

func runFlush(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		outputFlushResult(FlushResult{Error: "failed to load config"})
		os.Exit(FlushExitNotConfig)
	}

	if !cfg.IsConfigured() {
		outputFlushResult(FlushResult{Error: "not logged in: run 'tracktime login' first"})
		os.Exit(FlushExitNotConfig)
	}

	if !cfg.SyncEnabled() {
		outputFlushResult(FlushResult{Error: "offline mode enabled: run 'tracktime config set offline_mode false' to enable sync"})
		os.Exit(FlushExitNotConfig)
	}

	q, err := queue.Open()
	if err != nil {
		outputFlushResult(FlushResult{Error: "failed to open queue"})
		os.Exit(FlushExitNetworkError)
	}
	defer q.Close()

	client := api.NewClient(cfg.APIKey, cfg.MachineID)

	totalSynced := 0
	totalErrors := 0
	batchSize := 500

	for {
		heartbeats, ids, err := q.FetchUnsynced(batchSize)
		if err != nil {
			remaining, _ := q.Count()
			outputFlushResult(FlushResult{
				Synced:    totalSynced,
				Remaining: remaining,
				Errors:    totalErrors,
				Error:     fmt.Sprintf("failed to fetch heartbeats: %v", err),
			})
			os.Exit(FlushExitPartial)
		}

		if len(heartbeats) == 0 {
			break
		}

		resp, err := client.SendHeartbeats(heartbeats)
		if err != nil {
			totalErrors++

			if errors.Is(err, api.ErrUnauthorized) {
				remaining, _ := q.Count()
				outputFlushResult(FlushResult{
					Synced:    totalSynced,
					Remaining: remaining,
					Errors:    totalErrors,
					Error:     "unauthorized: try 'tracktime logout' then 'tracktime login' again",
				})
				os.Exit(FlushExitNotConfig)
			}

			if errors.Is(err, api.ErrBadRequest) {
				// Bad request means malformed data, mark as synced to avoid retrying
				q.MarkSynced(ids)
				continue
			}

			// Network or server error, stop and report
			remaining, _ := q.Count()
			outputFlushResult(FlushResult{
				Synced:    totalSynced,
				Remaining: remaining,
				Errors:    totalErrors,
				Error:     err.Error(),
			})
			os.Exit(FlushExitNetworkError)
		}

		if err := q.MarkSynced(ids); err != nil {
			remaining, _ := q.Count()
			outputFlushResult(FlushResult{
				Synced:    totalSynced,
				Remaining: remaining,
				Errors:    totalErrors,
				Error:     fmt.Sprintf("failed to mark as synced: %v", err),
			})
			os.Exit(FlushExitPartial)
		}

		totalSynced += resp.Accepted

		if len(heartbeats) < batchSize {
			break
		}
	}

	// Update last flush time
	cfg.UpdateLastFlush()
	config.Save(cfg)

	// Clean up synced heartbeats
	q.DeleteSynced()

	remaining, _ := q.Count()
	result := FlushResult{
		Synced:    totalSynced,
		Remaining: remaining,
		Errors:    totalErrors,
	}

	outputFlushResult(result)

	if totalErrors > 0 && remaining > 0 {
		os.Exit(FlushExitPartial)
	}

	return nil
}

func outputFlushResult(result FlushResult) {
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}
