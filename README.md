# Cortex

**External memory for AI agents**

Cortex is a persistent memory system that allows AI agents and developers to store, search, and relate technical knowledge across sessions.

## Features

- **Semantic Search**: Find relevant memories using natural language queries
- **Memory Types**: Categorize knowledge as errors, patterns, decisions, context, or procedures
- **Trust Levels**: Track validation state (proposed → validated → proven)
- **Relations**: Connect memories with causal relationships
- **Topic Keys**: Evolve memories over time without creating duplicates
- **MCP Integration**: Use as a tool with Claude, Cursor, and other MCP-compatible agents

## Installation

```bash
# From source
go install github.com/constantino-dev/cortex/cmd/cortex@latest

# Or build locally
git clone https://github.com/constantino-dev/cortex.git
cd cortex
make install
```

## Quick Start

```bash
# Initialize in your project
cd my-project
cortex init

# Store a memory
cortex store "React hooks must be called at top level, not inside loops"

# Store with metadata
cortex store -t pattern -k "react/hooks/rules" "Never call hooks conditionally"

# Search memories
cortex recall "react hooks best practices"

# List all memories
cortex list

# Show memory details
cortex show <memory-id>

# Create relations
cortex relate <error-id> solves <pattern-id>

# Validate a memory
cortex validate <memory-id>
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `cortex init` | Initialize a new memory store |
| `cortex store <content>` | Store a new memory |
| `cortex recall <query>` | Search memories semantically |
| `cortex list` | List stored memories |
| `cortex show <id>` | Show memory details |
| `cortex relate <from> <rel> <to>` | Create a relation |
| `cortex validate <id> [level]` | Update trust level |
| `cortex delete <id>` | Delete a memory |
| `cortex stats` | Show statistics |
| `cortex mcp` | Start MCP server |

## Memory Types

| Type | Description | Example |
|------|-------------|---------|
| `general` | General information | "Project uses TypeScript 5.0" |
| `error` | Something that failed | "useState in loop causes infinite re-renders" |
| `pattern` | Reusable solution | "Use useCallback for memoized callbacks" |
| `decision` | Why something was chosen | "Chose Zustand over Redux for simplicity" |
| `context` | Project state/info | "Migration is in phase 2" |
| `procedure` | How to do something | "To deploy: run npm build && npm deploy" |

## Trust Levels

| Level | Description |
|-------|-------------|
| `proposed` | Agent suggested, not validated |
| `validated` | Human confirmed or used successfully |
| `proven` | Multiple successful uses |
| `disputed` | Someone questioned it |
| `obsolete` | No longer applies |

## Relation Types

| Type | Description |
|------|-------------|
| `causes` | A causes B |
| `solves` | A solves B |
| `replaces` | A replaces B |
| `requires` | A requires B |
| `related_to` | A is related to B |
| `part_of` | A is part of B |
| `contradicts` | A contradicts B |

## MCP Integration

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "cortex": {
      "command": "cortex",
      "args": ["mcp", "-p", "/path/to/your/project"]
    }
  }
}
```

### MCP Tools

| Tool | Description |
|------|-------------|
| `cortex_store` | Store a new memory |
| `cortex_recall` | Search for relevant memories |
| `cortex_relate` | Create a relation between memories |
| `cortex_validate` | Update trust level |
| `cortex_learn_error` | Store an error with cause and solution |

## Configuration

Cortex stores its data in `.cortex/` in your project directory:

```
.cortex/
├── config.json    # Configuration
└── cortex.db      # SQLite database
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key for embeddings |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CORTEX                               │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐         ┌─────────────┐                   │
│  │    CLI      │         │  MCP Server │                   │
│  └──────┬──────┘         └──────┬──────┘                   │
│         └───────────┬───────────┘                           │
│                     ▼                                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   CORE ENGINE                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                     │                                       │
│  ┌──────────────────┴──────────────────┐                   │
│  │    SQLite + sqlite-vec + FTS5       │                   │
│  └─────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

## License

MIT
