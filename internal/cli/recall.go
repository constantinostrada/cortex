package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var recallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: "Search memories semantically",
	Long: `Search for relevant memories using semantic search.

Examples:
  cortex recall "how to handle async errors"
  cortex recall "react hooks" --limit 10
  cortex recall "migration patterns" --type pattern
  cortex recall "database decisions" --include-proposed`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRecall,
}

var (
	recallLimit           int
	recallTypes           string
	recallTags            string
	recallProject         string
	recallIncludeProposed bool
	recallMinScore        float64
)

func init() {
	recallCmd.Flags().IntVarP(&recallLimit, "limit", "n", 5, "Maximum results to return")
	recallCmd.Flags().StringVarP(&recallTypes, "type", "t", "", "Filter by type(s), comma-separated")
	recallCmd.Flags().StringVar(&recallTags, "tags", "", "Filter by tags, comma-separated")
	recallCmd.Flags().StringVar(&recallProject, "project", "", "Filter by project")
	recallCmd.Flags().BoolVar(&recallIncludeProposed, "include-proposed", false, "Include proposed (unvalidated) memories")
	recallCmd.Flags().Float64Var(&recallMinScore, "min-score", 0.3, "Minimum relevance score (0-1)")
}

func runRecall(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Build options
	opts := types.RecallOptions{
		Limit:    recallLimit,
		MinScore: recallMinScore,
		Project:  recallProject,
	}

	// Parse types
	if recallTypes != "" {
		for _, t := range strings.Split(recallTypes, ",") {
			opts.Types = append(opts.Types, types.MemoryType(strings.TrimSpace(t)))
		}
	}

	// Parse tags
	if recallTags != "" {
		for _, tag := range strings.Split(recallTags, ",") {
			opts.Tags = append(opts.Tags, strings.TrimSpace(tag))
		}
	}

	// Trust levels
	if recallIncludeProposed {
		opts.TrustLevels = []types.TrustLevel{
			types.TrustProposed,
			types.TrustValidated,
			types.TrustProven,
		}
	} else {
		opts.TrustLevels = []types.TrustLevel{
			types.TrustValidated,
			types.TrustProven,
		}
	}

	// Create engine
	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// Search
	results, err := engine.Recall(context.Background(), query, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No memories found matching your query.")
		if !recallIncludeProposed {
			fmt.Println("Tip: Try --include-proposed to search unvalidated memories.")
		}
		return nil
	}

	// Print results
	if verbose {
		printJSON(results)
	} else {
		for i, r := range results {
			fmt.Printf("\n[%d] %s (%.0f%% relevant)\n", i+1, formatType(r.Memory.Type), r.Score*100)
			fmt.Printf("    ID: %s\n", r.Memory.ID)
			if r.Memory.TopicKey != "" {
				fmt.Printf("    Topic: %s\n", r.Memory.TopicKey)
			}
			fmt.Printf("    Trust: %s\n", r.Memory.Trust)
			fmt.Printf("    Content: %s\n", truncate(r.Memory.Content, 200))
			if len(r.Memory.Tags) > 0 {
				fmt.Printf("    Tags: %s\n", strings.Join(r.Memory.Tags, ", "))
			}
		}
	}

	return nil
}

func formatType(t types.MemoryType) string {
	switch t {
	case types.TypeError:
		return "🔴 ERROR"
	case types.TypePattern:
		return "🟢 PATTERN"
	case types.TypeDecision:
		return "🔵 DECISION"
	case types.TypeContext:
		return "🟡 CONTEXT"
	case types.TypeProcedure:
		return "🟣 PROCEDURE"
	default:
		return "⚪ GENERAL"
	}
}

func truncate(s string, maxLen int) string {
	// Replace newlines with spaces for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
