package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored memories",
	Long: `List all stored memories with optional filters.

Examples:
  cortex list
  cortex list --type error
  cortex list --limit 20
  cortex list --project my-project`,
	RunE: runList,
}

var (
	listLimit   int
	listTypes   string
	listTrust   string
	listProject string
	listTopicKey string
)

func init() {
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 20, "Maximum results")
	listCmd.Flags().StringVarP(&listTypes, "type", "t", "", "Filter by type(s)")
	listCmd.Flags().StringVar(&listTrust, "trust", "", "Filter by trust level")
	listCmd.Flags().StringVar(&listProject, "project", "", "Filter by project")
	listCmd.Flags().StringVarP(&listTopicKey, "key", "k", "", "Filter by topic key prefix")
}

func runList(cmd *cobra.Command, args []string) error {
	// Build options
	opts := types.RecallOptions{
		Limit:    listLimit,
		Project:  listProject,
		TopicKey: listTopicKey,
	}

	// Parse types
	if listTypes != "" {
		for _, t := range strings.Split(listTypes, ",") {
			opts.Types = append(opts.Types, types.MemoryType(strings.TrimSpace(t)))
		}
	}

	// Parse trust
	if listTrust != "" {
		for _, t := range strings.Split(listTrust, ",") {
			opts.TrustLevels = append(opts.TrustLevels, types.TrustLevel(strings.TrimSpace(t)))
		}
	}

	// Create engine
	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// List memories
	memories, err := engine.List(opts)
	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	if len(memories) == 0 {
		fmt.Println("No memories found.")
		return nil
	}

	// Print results
	if verbose {
		printJSON(memories)
	} else {
		// Table header
		fmt.Printf("%-8s %-10s %-30s %-12s %s\n", "TYPE", "TRUST", "TOPIC", "UPDATED", "CONTENT")
		fmt.Println(strings.Repeat("-", 100))

		for _, m := range memories {
			typeStr := string(m.Type)
			if len(typeStr) > 8 {
				typeStr = typeStr[:8]
			}

			topicStr := m.TopicKey
			if topicStr == "" {
				topicStr = m.ID[:12]
			}
			if len(topicStr) > 30 {
				topicStr = topicStr[:27] + "..."
			}

			contentPreview := strings.ReplaceAll(m.Content, "\n", " ")
			if len(contentPreview) > 40 {
				contentPreview = contentPreview[:37] + "..."
			}

			updatedAgo := formatTimeAgo(m.UpdatedAt)

			fmt.Printf("%-8s %-10s %-30s %-12s %s\n",
				typeStr, m.Trust, topicStr, updatedAgo, contentPreview)
		}

		fmt.Printf("\nTotal: %d memories\n", len(memories))
	}

	return nil
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
