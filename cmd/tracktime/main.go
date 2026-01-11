package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "tracktime",
	Short: "Track your coding time",
	Long:  `tracktime is a CLI tool that tracks your coding time across editors.`,
}

func main() {
	rootCmd.Version = config.Version

	rootCmd.AddCommand(heartbeatCmd)
	rootCmd.AddCommand(flushCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(statsCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
