package cli

import (
	"fmt"

	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var relateCmd = &cobra.Command{
	Use:   "relate <from-id> <relation> <to-id>",
	Short: "Create a relation between memories",
	Long: `Create a relation between two memories.

Relation types:
  causes      - A causes B (e.g., error causes crash)
  solves      - A solves B (e.g., pattern solves error)
  replaces    - A replaces B (e.g., new approach replaces old)
  requires    - A requires B (e.g., solution requires dependency)
  related_to  - A is related to B (general relation)
  part_of     - A is part of B (e.g., step part of procedure)
  contradicts - A contradicts B (conflicting information)

Examples:
  cortex relate abc123 causes def456
  cortex relate abc123 solves def456 --note "Using error boundary"
  cortex relate pattern1 replaces pattern2`,
	Args: cobra.ExactArgs(3),
	RunE: runRelate,
}

var relateNote string

func init() {
	relateCmd.Flags().StringVar(&relateNote, "note", "", "Note explaining the relation")
}

func runRelate(cmd *cobra.Command, args []string) error {
	fromID := args[0]
	relType := types.RelationType(args[1])
	toID := args[2]

	// Validate relation type
	validTypes := map[types.RelationType]bool{
		types.RelCauses:      true,
		types.RelSolves:      true,
		types.RelReplaces:    true,
		types.RelRequires:    true,
		types.RelRelatedTo:   true,
		types.RelPartOf:      true,
		types.RelContradicts: true,
	}

	if !validTypes[relType] {
		return fmt.Errorf("invalid relation type: %s\nValid types: causes, solves, replaces, requires, related_to, part_of, contradicts", relType)
	}

	engine, err := getEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	relation, err := engine.Relate(fromID, toID, relType, relateNote)
	if err != nil {
		return fmt.Errorf("failed to create relation: %w", err)
	}

	if verbose {
		printJSON(relation)
	} else {
		fmt.Printf("✓ Created relation: %s -[%s]-> %s\n", fromID, relType, toID)
	}

	return nil
}
