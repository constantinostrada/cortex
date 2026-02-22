package cli

import (
	"fmt"

	"github.com/constantino-dev/cortex/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for agent integration",
	Long: `Start the Model Context Protocol (MCP) server.

This allows AI agents to use Cortex as an external memory system.

The server communicates over stdio using JSON-RPC.

Example usage with Claude Desktop:
  Add to claude_desktop_config.json:
  {
    "mcpServers": {
      "cortex": {
        "command": "cortex",
        "args": ["mcp", "-p", "/path/to/project"]
      }
    }
  }`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	engine, err := getEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}
	defer engine.Close()

	server := mcp.NewServer(engine)
	return server.Run()
}
