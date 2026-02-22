package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store [content]",
	Short: "Store a new memory",
	Long: `Store a new memory in Cortex.

Content can be provided as an argument or piped from stdin.

Examples:
  cortex store "React hooks must be called at top level"
  cortex store -t pattern -k "react/hooks/rules" "Don't use hooks in loops"
  echo "Important fact" | cortex store
  cortex store --type error --tags "react,migration" "useState in loop causes issues"`,
	RunE: runStore,
}

var (
	storeType     string
	storeTopicKey string
	storeTags     string
	storeTrust    string
	storeSource   string
	storeProject  string
)

func init() {
	storeCmd.Flags().StringVarP(&storeType, "type", "t", "general", "Memory type (general, error, pattern, decision, context, procedure)")
	storeCmd.Flags().StringVarP(&storeTopicKey, "key", "k", "", "Topic key for memory evolution (e.g., react/hooks/rules)")
	storeCmd.Flags().StringVar(&storeTags, "tags", "", "Comma-separated tags")
	storeCmd.Flags().StringVar(&storeTrust, "trust", "proposed", "Trust level (proposed, validated, proven)")
	storeCmd.Flags().StringVar(&storeSource, "source", "cli", "Source of memory")
	storeCmd.Flags().StringVar(&storeProject, "project", "", "Project scope")
}

func runStore(cmd *cobra.Command, args []string) error {
	// Get content from args or stdin
	var content string
	if len(args) > 0 {
		content = strings.Join(args, " ")
	} else {
		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			content = strings.TrimSpace(string(data))
		}
	}

	if content == "" {
		return fmt.Errorf("no content provided. Usage: cortex store \"your memory content\"")
	}

	// Parse tags
	var tags []string
	if storeTags != "" {
		for _, tag := range strings.Split(storeTags, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	// Parse type
	memType := types.MemoryType(storeType)

	// Parse trust
	trust := types.TrustLevel(storeTrust)

	// Create engine
	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// Store memory
	memory, err := engine.Store(context.Background(), content, types.StoreOptions{
		Type:     memType,
		TopicKey: storeTopicKey,
		Tags:     tags,
		Trust:    trust,
		Source:   storeSource,
		Project:  storeProject,
	})
	if err != nil {
		return fmt.Errorf("failed to store: %w", err)
	}

	// Output
	if verbose {
		printJSON(memory)
	} else {
		fmt.Printf("✓ Stored memory: %s\n", memory.ID)
		if memory.TopicKey != "" {
			fmt.Printf("  Topic: %s\n", memory.TopicKey)
		}
	}

	return nil
}
