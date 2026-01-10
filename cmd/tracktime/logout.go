package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/config"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of tracktime.sh",
	Long:  `Removes the stored API key from your local configuration.`,
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if !cfg.IsConfigured() {
		fmt.Println("You are not logged in.")
		return nil
	}

	cfg.ClearAuth()

	if err := config.Save(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Logged out successfully.")
	return nil
}
