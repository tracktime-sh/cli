package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/api"
	"github.com/tracktime-sh/cli/internal/config"
)

var whoamiJSONOutput bool

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently logged in user",
	Long:  `Displays information about the currently authenticated user.`,
	RunE:  runWhoami,
}

func init() {
	whoamiCmd.Flags().BoolVar(&whoamiJSONOutput, "json", false, "Output in JSON format")
}

type WhoamiOutput struct {
	Email  string `json:"email"`
	UserID string `json:"user_id"`
}

func runWhoami(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if !cfg.IsConfigured() {
		if whoamiJSONOutput {
			fmt.Println("{}")
		} else {
			fmt.Println("Not logged in. Run 'tracktime login' to authenticate.")
		}
		os.Exit(1)
	}

	client := api.NewClient(cfg.APIKey, cfg.MachineID)

	user, err := client.GetUser()
	if err != nil {
		if errors.Is(err, api.ErrUnauthorized) {
			if whoamiJSONOutput {
				fmt.Println("{}")
			} else {
				fmt.Println("API key is invalid or expired.")
				fmt.Println("Run 'tracktime logout' then 'tracktime login' to re-authenticate.")
			}
			os.Exit(1)
		}
		fmt.Printf("Error fetching user info: %v\n", err)
		os.Exit(1)
	}

	if whoamiJSONOutput {
		output := WhoamiOutput{
			Email:  user.Email,
			UserID: user.ID,
		}
		data, _ := json.Marshal(output)
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Logged in as: %s\n", user.Email)
	fmt.Printf("User ID:      %s\n", user.ID)

	return nil
}
