// Package cli implements the Cortex command-line interface
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/constantino-dev/cortex/internal/core"
	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

const (
	configDir  = ".cortex"
	configFile = "config.json"
	dbFile     = "cortex.db"
)

var (
	// Global flags
	projectDir string
	verbose    bool

	// Root command
	rootCmd = &cobra.Command{
		Use:   "cortex",
		Short: "Cortex - External memory for AI agents",
		Long: `Cortex is a persistent memory system for AI agents and developers.

It allows storing, searching, and relating technical knowledge
that persists across sessions.

Use 'cortex init' to initialize a new memory store.`,
	}
)

// Execute runs the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectDir, "project", "p", "", "Project directory (default: current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(storeCmd)
	rootCmd.AddCommand(recallCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(relateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(statsCmd)
}

// getProjectDir returns the project directory
func getProjectDir() string {
	if projectDir != "" {
		return projectDir
	}
	cwd, _ := os.Getwd()
	return cwd
}

// getConfigPath returns the path to config directory
func getConfigPath() string {
	return filepath.Join(getProjectDir(), configDir)
}

// loadConfig loads the configuration
func loadConfig() (*types.Config, error) {
	configPath := filepath.Join(getConfigPath(), configFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config not found. Run 'cortex init' first")
	}

	var cfg types.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set DB path if not set
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(getConfigPath(), dbFile)
	}

	return &cfg, nil
}

// saveConfig saves the configuration
func saveConfig(cfg *types.Config) error {
	configPath := filepath.Join(getConfigPath(), configFile)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getEngine creates and returns a Cortex engine
func getEngine() (*core.Engine, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return core.New(cfg)
}

// printJSON prints a value as JSON
func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

// printError prints an error message
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
