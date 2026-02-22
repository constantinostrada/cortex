package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show details of a memory",
	Long: `Show full details of a specific memory including relations.

Examples:
  cortex show abc123def456
  cortex show abc123 --relations`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

var showRelations bool

func init() {
	showCmd.Flags().BoolVarP(&showRelations, "relations", "r", true, "Show relations")
}

func runShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// Get memory
	memory, err := engine.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}
	if memory == nil {
		return fmt.Errorf("memory not found: %s", id)
	}

	// Print details
	if verbose {
		printJSON(memory)
	} else {
		fmt.Printf("┌─────────────────────────────────────────────────────────────┐\n")
		fmt.Printf("│ %s\n", memory.ID)
		fmt.Printf("├─────────────────────────────────────────────────────────────┤\n")
		fmt.Printf("│ Type:     %s\n", formatType(memory.Type))
		fmt.Printf("│ Trust:    %s\n", memory.Trust)
		if memory.TopicKey != "" {
			fmt.Printf("│ Topic:    %s\n", memory.TopicKey)
		}
		if len(memory.Tags) > 0 {
			fmt.Printf("│ Tags:     %s\n", strings.Join(memory.Tags, ", "))
		}
		fmt.Printf("│ Created:  %s\n", memory.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("│ Updated:  %s\n", memory.UpdatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("│ Accessed: %d times\n", memory.AccessCnt)
		fmt.Printf("├─────────────────────────────────────────────────────────────┤\n")
		fmt.Printf("│ Content:\n")
		for _, line := range strings.Split(memory.Content, "\n") {
			fmt.Printf("│   %s\n", line)
		}
		fmt.Printf("└─────────────────────────────────────────────────────────────┘\n")
	}

	// Show relations
	if showRelations {
		relations, err := engine.GetRelations(id)
		if err != nil {
			printError("failed to get relations: %v", err)
		} else if len(relations) > 0 {
			fmt.Println("\nRelations:")
			for _, r := range relations {
				direction := "→"
				otherID := r.ToID
				if r.ToID == id {
					direction = "←"
					otherID = r.FromID
				}
				fmt.Printf("  %s [%s] %s", direction, r.Type, otherID)
				if r.Note != "" {
					fmt.Printf(" (%s)", r.Note)
				}
				fmt.Println()
			}
		}
	}

	return nil
}
