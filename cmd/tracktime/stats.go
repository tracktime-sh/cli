package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/queue"
)

var (
	statsDays       int
	statsJSONOutput bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show local time tracking statistics",
	Long:  `Displays statistics from locally stored heartbeats.`,
	RunE:  runStats,
}

func init() {
	statsCmd.Flags().IntVarP(&statsDays, "days", "d", 7, "Number of days to include in stats")
	statsCmd.Flags().BoolVar(&statsJSONOutput, "json", false, "Output in JSON format")
}

func runStats(cmd *cobra.Command, args []string) error {
	q, err := queue.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to open queue: %v\n", err)
		os.Exit(1)
	}
	defer q.Close()

	since := time.Now().AddDate(0, 0, -statsDays)
	stats, err := q.GetStats(since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get stats: %v\n", err)
		os.Exit(1)
	}

	if statsJSONOutput {
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Last %d days:\n\n", statsDays)
	fmt.Printf("Total time:   %s\n", stats.TotalTime)
	fmt.Printf("Heartbeats:   %d\n", stats.Heartbeats)

	if len(stats.Days) > 0 {
		fmt.Printf("\nBy day:\n")
		for _, day := range stats.Days {
			fmt.Printf("  %s: %s\n", day.Date, day.Duration)
		}
	}

	if len(stats.ByProject) > 0 {
		fmt.Printf("\nBy project:\n")
		for project, minutes := range stats.ByProject {
			if project == "" {
				project = "(none)"
			}
			fmt.Printf("  %s: %s\n", project, formatDuration(minutes))
		}
	}

	if len(stats.ByLanguage) > 0 {
		fmt.Printf("\nBy language:\n")
		for lang, minutes := range stats.ByLanguage {
			if lang == "" {
				lang = "(none)"
			}
			fmt.Printf("  %s: %s\n", lang, formatDuration(minutes))
		}
	}

	if len(stats.ByEditor) > 0 {
		fmt.Printf("\nBy editor:\n")
		for editor, minutes := range stats.ByEditor {
			fmt.Printf("  %s: %s\n", editor, formatDuration(minutes))
		}
	}

	return nil
}

func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
