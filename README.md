# Cortex

**External memory for AI agents**

Cortex is a persistent memory system that allows AI agents and developers to store, search, and relate technical knowledge across sessions. It solves the problem of AI agents being stateless - repeating errors, losing architectural decisions, and generating inconsistent solutions.

## Features

- **Semantic Search**: Find relevant memories using natural language queries
- **Memory Types**: Categorize knowledge as errors, patterns, decisions, context, or procedures
- **Trust Levels**: Track validation state (proposed → validated → proven)
- **Relations**: Connect memories with causal relationships
- **Topic Keys**: Evolve memories over time without creating duplicates
- **MCP Integration**: Use as a tool with Claude, Cursor, and other MCP-compatible agents

---

## Installation

### Prerequisites

- Go 1.22+ (`brew install go`)
- OpenAI API key (for embeddings)

### From Source

```bash
# 1. Clone the repo
git clone https://github.com/constantinostrada/cortex.git
cd cortex

# 2. Build
make build

# 3. Install globally
make install

# 4. Add to PATH (if not already)
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 5. Verify installation
cortex --help
```

---

## Quick Start

```bash
# Initialize in your project (once per project)
cd my-project
cortex init
# It will ask for your OPENAI_API_KEY

# Store memories
cortex store "React hooks must be called at top level"
cortex store -t error "useState in loop causes infinite re-renders"
cortex store -t pattern -k "react/hooks/rules" "Never call hooks conditionally"

# Search memories
cortex recall "react hooks"

# List all memories
cortex list

# Show memory details
cortex show <memory-id>

# Validate a memory (confirm it's correct)
cortex validate <memory-id>

# Create relations
cortex relate <error-id> solves <pattern-id>
```

---

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

---

## Memory Types

| Type | When to use | Example |
|------|-------------|---------|
| `general` | General information | "Project uses TypeScript 5.0" |
| `error` | Something that failed | "useState in loop causes infinite re-renders" |
| `pattern` | Reusable solution | "Use useCallback for memoized callbacks" |
| `decision` | Why something was chosen | "Chose Zustand over Redux for simplicity" |
| `context` | Project state/info | "Migration is in phase 2" |
| `procedure` | How to do something | "To deploy: run npm build && npm deploy" |

### Usage

```bash
# Store with type
cortex store -t error "Memory leak when not cleaning up useEffect"
cortex store -t pattern "Always return cleanup function from useEffect"
cortex store -t decision "Using React Query for server state management"
```

---

## Trust Levels (Validation)

| Level | Icon | Description |
|-------|------|-------------|
| `proposed` | 🟡 | Agent suggested, not validated |
| `validated` | 🟢 | Human confirmed or used successfully |
| `proven` | ⭐ | Multiple successful uses |
| `disputed` | 🔴 | Someone questioned it |
| `obsolete` | ⚫ | No longer applies |

**Important**: By default, `cortex recall` only returns `validated` or `proven` memories.

To include proposed memories:
```bash
cortex recall "query" --include-proposed
```

### Validation Flow

```bash
# Agent stores a memory (starts as "proposed")
cortex store "Use memo for expensive calculations"

# Human validates it after confirming it works
cortex validate <id>              # Sets to "validated"
cortex validate <id> proven       # Sets to "proven"
cortex validate <id> obsolete     # Marks as outdated
```

---

## Relations

Connect memories to build a knowledge graph:

| Type | Description | Example |
|------|-------------|---------|
| `causes` | A causes B | Error causes crash |
| `solves` | A solves B | Pattern solves error |
| `replaces` | A replaces B | New approach replaces old |
| `requires` | A requires B | Solution requires dependency |
| `related_to` | A is related to B | General relation |
| `part_of` | A is part of B | Step part of procedure |
| `contradicts` | A contradicts B | Conflicting information |

### Usage

```bash
# Create a relation
cortex relate <error-id> solves <pattern-id>
cortex relate <pattern-id> requires <context-id>
cortex relate <old-pattern-id> replaces <new-pattern-id> --note "Deprecated in v2"
```

---

## MCP Integration (AI Agents)

Cortex can be used as a tool by AI agents via the Model Context Protocol (MCP).

### Setup for Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "cortex": {
      "command": "/path/to/go/bin/cortex",
      "args": ["mcp", "-p", "/path/to/your/project"]
    }
  }
}
```

Restart Claude Desktop. The agent now has access to:

| Tool | Description |
|------|-------------|
| `cortex_store` | Store a new memory |
| `cortex_recall` | Search for relevant memories |
| `cortex_relate` | Create a relation between memories |
| `cortex_validate` | Update trust level |
| `cortex_learn_error` | Store an error with cause and solution |

---

## Recommended Workflow

### For Developers

```
1. BEFORE starting a task:
   cortex recall "what you're about to do"
   → Check for known patterns/errors

2. DURING work:
   Found a new error    → cortex store -t error "..."
   Discovered a pattern → cortex store -t pattern "..."
   Made a decision      → cortex store -t decision "..."

3. AFTER solving something:
   cortex relate <error-id> solves <pattern-id>
   cortex validate <id>  # If confirmed it works
```

### For AI Agents

```
1. Before making changes:
   cortex_recall("what I'm about to modify")

2. If something fails:
   cortex_learn_error(error, cause, solution)

3. If solution works:
   cortex_validate(id, "validated")
```

---

## Project Structure

```
your-project/
└── .cortex/
    ├── config.json   # Configuration (API keys, settings)
    └── cortex.db     # SQLite database (memories, embeddings)
```

**Note**: Don't commit `.cortex/` - each developer has their own local memory.

Add to `.gitignore`:
```
.cortex/
```

---

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key for embeddings |

### Config File

`.cortex/config.json`:
```json
{
  "db_path": ".cortex/cortex.db",
  "embedding_provider": "openai",
  "openai_key": "sk-..."
}
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CORTEX                               │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐         ┌─────────────┐                   │
│  │    CLI      │         │  MCP Server │                   │
│  │  (humans)   │         │  (agents)   │                   │
│  └──────┬──────┘         └──────┬──────┘                   │
│         └───────────┬───────────┘                           │
│                     ▼                                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   CORE ENGINE                       │   │
│  │  • Store memories    • Semantic search              │   │
│  │  • Create relations  • Trust validation             │   │
│  └─────────────────────────────────────────────────────┘   │
│                     │                                       │
│  ┌──────────────────┴──────────────────┐                   │
│  │         STORAGE LAYER               │                   │
│  │  SQLite + sqlite-vec + FTS5         │                   │
│  │  • Memories & relations             │                   │
│  │  • Vector embeddings (1536D)        │                   │
│  │  • Full-text search index           │                   │
│  └─────────────────────────────────────┘                   │
│                     │                                       │
│  ┌──────────────────┴──────────────────┐                   │
│  │         EMBEDDINGS                  │                   │
│  │  OpenAI text-embedding-3-small      │                   │
│  └─────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Reference

```bash
# Initialize
cortex init

# Store
cortex store "content"
cortex store -t error "error message"
cortex store -t pattern -k "topic/key" "pattern description"

# Search
cortex recall "query"
cortex recall "query" --include-proposed
cortex recall "query" -t error --limit 10

# Manage
cortex list
cortex show <id>
cortex validate <id>
cortex delete <id>

# Relations
cortex relate <from> causes <to>
cortex relate <from> solves <to>

# MCP Server
cortex mcp -p /path/to/project
```

---

## License

MIT
