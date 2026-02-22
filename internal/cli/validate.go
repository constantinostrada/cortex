package cli

import (
	"fmt"

	"github.com/constantino-dev/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <id> [trust-level]",
	Short: "Update trust level of a memory",
	Long: `Update the trust level of a memory.

Trust levels:
  proposed  - Agent suggested, not validated (default for new memories)
  validated - Human confirmed or used successfully
  proven    - Multiple successful uses
  disputed  - Someone questioned it
  obsolete  - No longer applies

Examples:
  cortex validate abc123              # Promotes to 'validated'
  cortex validate abc123 proven       # Sets to 'proven'
  cortex validate abc123 obsolete     # Marks as obsolete`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Determine new trust level
	var newTrust types.TrustLevel
	if len(args) > 1 {
		newTrust = types.TrustLevel(args[1])
	} else {
		// Default: promote to validated
		newTrust = types.TrustValidated
	}

	// Validate trust level
	validLevels := map[types.TrustLevel]bool{
		types.TrustProposed:  true,
		types.TrustValidated: true,
		types.TrustProven:    true,
		types.TrustDisputed:  true,
		types.TrustObsolete:  true,
	}

	if !validLevels[newTrust] {
		return fmt.Errorf("invalid trust level: %s\nValid levels: proposed, validated, proven, disputed, obsolete", newTrust)
	}

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

	oldTrust := memory.Trust

	// Update trust
	if err := engine.Validate(id, newTrust); err != nil {
		return fmt.Errorf("failed to update trust: %w", err)
	}

	fmt.Printf("✓ Updated trust: %s → %s\n", oldTrust, newTrust)

	return nil
}
