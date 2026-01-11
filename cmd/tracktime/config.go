package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tracktime-sh/cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage tracktime configuration",
	Long:  `View and modify tracktime configuration settings.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE:  runConfigList,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	key := strings.ToLower(args[0])

	switch key {
	case "offline_mode":
		fmt.Println(cfg.OfflineMode)
	case "flush_interval":
		fmt.Println(cfg.FlushIntervalSeconds)
	case "machine_id":
		fmt.Println(cfg.MachineID)
	case "api_url":
		fmt.Println(config.APIURL)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown config key: %s\n", key)
		fmt.Fprintf(os.Stderr, "available keys: offline_mode, flush_interval, machine_id, api_url\n")
		os.Exit(1)
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	key := strings.ToLower(args[0])
	value := args[1]

	switch key {
	case "offline_mode":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid boolean value: %s (use true/false)\n", value)
			os.Exit(1)
		}
		cfg.OfflineMode = boolVal
		if boolVal {
			fmt.Println("Offline mode enabled. Your data will stay local and won't sync to the server.")
		} else {
			fmt.Println("Offline mode disabled. Data will sync to the server.")
		}
	case "flush_interval":
		intVal, err := strconv.Atoi(value)
		if err != nil || intVal < 10 {
			fmt.Fprintf(os.Stderr, "error: invalid interval: %s (must be >= 10 seconds)\n", value)
			os.Exit(1)
		}
		cfg.FlushIntervalSeconds = intVal
		fmt.Printf("Flush interval set to %d seconds.\n", intVal)
	default:
		fmt.Fprintf(os.Stderr, "error: cannot set config key: %s\n", key)
		fmt.Fprintf(os.Stderr, "settable keys: offline_mode, flush_interval\n")
		os.Exit(1)
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to save config: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	output := map[string]interface{}{
		"offline_mode":   cfg.OfflineMode,
		"flush_interval": cfg.FlushIntervalSeconds,
		"machine_id":     cfg.MachineID,
		"api_url":        config.APIURL,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))

	return nil
}
