package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a memory",
	Long: `Delete a memory and its relations.

Examples:
  cortex delete abc123
  cortex delete abc123 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

var deleteForce bool

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation")
}

func runDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	// Verify memory exists
	memory, err := engine.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get memory: %w", err)
	}
	if memory == nil {
		return fmt.Errorf("memory not found: %s", id)
	}

	// Confirm deletion
	if !deleteForce {
		fmt.Printf("Memory to delete:\n")
		fmt.Printf("  ID: %s\n", memory.ID)
		fmt.Printf("  Type: %s\n", memory.Type)
		fmt.Printf("  Content: %s\n", truncate(memory.Content, 100))
		fmt.Print("\nAre you sure? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Delete
	if err := engine.Delete(id); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Printf("✓ Deleted memory: %s\n", id)

	return nil
}
