package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Cortex memory store",
	Long: `Initialize a new Cortex memory store in the current directory.

This creates a .cortex directory with configuration and database files.`,
	RunE: runInit,
}

var (
	initOpenAIKey string
	initProvider  string
)

func init() {
	initCmd.Flags().StringVar(&initOpenAIKey, "openai-key", "", "OpenAI API key")
	initCmd.Flags().StringVar(&initProvider, "provider", "openai", "Embedding provider (openai, ollama)")
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Check if already initialized
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("cortex already initialized in this directory")
	}

	// Create config directory
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get OpenAI key
	apiKey := initOpenAIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if initProvider == "openai" && apiKey == "" {
		// Prompt for key
		fmt.Print("Enter OpenAI API key (or set OPENAI_API_KEY env var): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(input)

		if apiKey == "" {
			// Clean up
			os.RemoveAll(configPath)
			return fmt.Errorf("OpenAI API key is required")
		}
	}

	// Create config
	cfg := &types.Config{
		DBPath:            filepath.Join(configPath, dbFile),
		EmbeddingProvider: initProvider,
		OpenAIKey:         apiKey,
	}

	// Save config
	if err := saveConfig(cfg); err != nil {
		os.RemoveAll(configPath)
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Initialize database by opening engine
	engine, err := getEngine()
	if err != nil {
		os.RemoveAll(configPath)
		return fmt.Errorf("failed to initialize: %w", err)
	}
	engine.Close()

	fmt.Println("✓ Cortex initialized successfully")
	fmt.Printf("  Config: %s\n", filepath.Join(configPath, configFile))
	fmt.Printf("  Database: %s\n", cfg.DBPath)

	return nil
}
