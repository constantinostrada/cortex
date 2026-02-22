package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show memory statistics",
	Long: `Show statistics about the Cortex memory store.

Examples:
  cortex stats`,
	RunE: runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	stats, err := engine.Stats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	if verbose {
		printJSON(stats)
	} else {
		fmt.Println("Cortex Statistics")
		fmt.Println("─────────────────")
		fmt.Printf("Memories:   %d\n", stats["memories"])
		fmt.Printf("Relations:  %d\n", stats["relations"])
		fmt.Printf("Embeddings: %d\n", stats["embeddings"])
	}

	return nil
}
