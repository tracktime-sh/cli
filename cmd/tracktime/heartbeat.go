package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/api"
	"github.com/tracktime-sh/cli/internal/config"
	"github.com/tracktime-sh/cli/internal/heartbeat"
	"github.com/tracktime-sh/cli/internal/queue"
)

const (
	ExitSuccess       = 0
	ExitInvalidInput  = 1
	ExitNotConfigured = 2
	ExitQueueError    = 3
)

type HeartbeatResult struct {
	Status      string `json:"status"`
	QueueSize   int    `json:"queue_size"`
	Synced      bool   `json:"synced"`
	SyncedCount int    `json:"synced_count,omitempty"`
	Error       string `json:"error,omitempty"`
}

var (
	stdinFlag bool
	fileFlag  string
	jsonFlag  string
)

var heartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "Queue a heartbeat from an editor",
	Long:  `Receives heartbeat data from an editor extension and queues it locally.`,
	RunE:  runHeartbeat,
}

func init() {
	heartbeatCmd.Flags().BoolVar(&stdinFlag, "stdin", false, "Read heartbeat JSON from stdin")
	heartbeatCmd.Flags().StringVar(&fileFlag, "file", "", "Read heartbeat JSON from file")
	heartbeatCmd.Flags().StringVar(&jsonFlag, "json", "", "Heartbeat JSON string")
}

func runHeartbeat(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		outputResult(HeartbeatResult{Status: "error", Error: "failed to load config"})
		os.Exit(ExitQueueError)
	}

	var input []byte

	switch {
	case stdinFlag:
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			outputResult(HeartbeatResult{Status: "error", Error: "failed to read stdin"})
			os.Exit(ExitInvalidInput)
		}
	case fileFlag != "":
		input, err = os.ReadFile(fileFlag)
		if err != nil {
			outputResult(HeartbeatResult{Status: "error", Error: "failed to read file"})
			os.Exit(ExitInvalidInput)
		}
	case jsonFlag != "":
		input = []byte(jsonFlag)
	default:
		outputResult(HeartbeatResult{Status: "error", Error: "no input provided, use --stdin, --file, or --json"})
		os.Exit(ExitInvalidInput)
	}

	heartbeats, err := heartbeat.ParseMultiple(input)
	if err != nil {
		outputResult(HeartbeatResult{Status: "error", Error: fmt.Sprintf("invalid JSON: %v", err)})
		os.Exit(ExitInvalidInput)
	}

	if len(heartbeats) == 0 {
		outputResult(HeartbeatResult{Status: "error", Error: "no heartbeats in input"})
		os.Exit(ExitInvalidInput)
	}

	for _, hb := range heartbeats {
		if err := hb.Validate(); err != nil {
			outputResult(HeartbeatResult{Status: "error", Error: err.Error()})
			os.Exit(ExitInvalidInput)
		}
	}

	q, err := queue.Open()
	if err != nil {
		outputResult(HeartbeatResult{Status: "error", Error: "failed to open queue"})
		os.Exit(ExitQueueError)
	}
	defer q.Close()

	if err := q.InsertBatch(heartbeats); err != nil {
		outputResult(HeartbeatResult{Status: "error", Error: "failed to queue heartbeat"})
		os.Exit(ExitQueueError)
	}

	queueSize, _ := q.Count()

	result := HeartbeatResult{
		Status:    "queued",
		QueueSize: queueSize,
		Synced:    false,
	}

	// Attempt opportunistic flush if configured, sync enabled, and due
	if cfg.IsConfigured() && cfg.SyncEnabled() && cfg.ShouldFlush() {
		syncedCount, err := doFlush(cfg, q)
		if err == nil && syncedCount > 0 {
			result.Synced = true
			result.SyncedCount = syncedCount
			result.QueueSize, _ = q.Count()
		}
	}

	outputResult(result)
	return nil
}

func doFlush(cfg *config.Config, q *queue.Queue) (int, error) {
	client := api.NewClient(cfg.APIKey, cfg.MachineID)

	totalSynced := 0
	batchSize := 500

	for {
		heartbeats, ids, err := q.FetchUnsynced(batchSize)
		if err != nil {
			return totalSynced, err
		}

		if len(heartbeats) == 0 {
			break
		}

		resp, err := client.SendHeartbeats(heartbeats)
		if err != nil {
			return totalSynced, err
		}

		if err := q.MarkSynced(ids); err != nil {
			return totalSynced, err
		}

		totalSynced += resp.Accepted

		if len(heartbeats) < batchSize {
			break
		}
	}

	if totalSynced > 0 {
		cfg.UpdateLastFlush()
		config.Save(cfg)
		q.DeleteSynced()
	}

	return totalSynced, nil
}

func outputResult(result HeartbeatResult) {
	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}
