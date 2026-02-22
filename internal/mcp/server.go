// Package mcp implements the Model Context Protocol server for Cortex
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/constantino-dev/cortex/internal/core"
	"github.com/constantino-dev/cortex/pkg/types"
)

// Server implements the MCP protocol over stdio
type Server struct {
	engine *core.Engine
	reader *bufio.Reader
	writer io.Writer
}

// NewServer creates a new MCP server
func NewServer(engine *core.Engine) *Server {
	return &Server{
		engine: engine,
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// JSON-RPC structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP-specific structures
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Tools map[string]interface{} `json:"tools,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Run starts the MCP server
func (s *Server) Run() error {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *Server) handleRequest(req *Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "notifications/initialized":
		// Client acknowledged initialization, no response needed
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *Request) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: map[string]interface{}{},
		},
		ServerInfo: ServerInfo{
			Name:    "cortex",
			Version: "0.1.0",
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req *Request) {
	tools := []Tool{
		{
			Name:        "cortex_store",
			Description: "Store a new memory in Cortex. Use this to save important information, patterns, errors, or decisions that should be remembered.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to store",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"general", "error", "pattern", "decision", "context", "procedure"},
						"description": "Type of memory",
						"default":     "general",
					},
					"topic_key": map[string]interface{}{
						"type":        "string",
						"description": "Topic key for memory evolution (e.g., 'react/hooks/rules'). If a memory with this key exists, it will be updated.",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Tags for categorization",
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "cortex_recall",
			Description: "Search for relevant memories. Use this to retrieve context before making decisions or to check if similar issues have been encountered before.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     5,
					},
					"type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"general", "error", "pattern", "decision", "context", "procedure"},
						"description": "Filter by memory type",
					},
					"include_proposed": map[string]interface{}{
						"type":        "boolean",
						"description": "Include unvalidated memories",
						"default":     false,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "cortex_relate",
			Description: "Create a relation between two memories. Use this to connect related knowledge (e.g., an error and its solution).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"from_id": map[string]interface{}{
						"type":        "string",
						"description": "Source memory ID",
					},
					"to_id": map[string]interface{}{
						"type":        "string",
						"description": "Target memory ID",
					},
					"relation": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"causes", "solves", "replaces", "requires", "related_to", "part_of", "contradicts"},
						"description": "Type of relation",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Optional note explaining the relation",
					},
				},
				"required": []string{"from_id", "to_id", "relation"},
			},
		},
		{
			Name:        "cortex_validate",
			Description: "Update the trust level of a memory. Use this to confirm a memory is correct or mark it as obsolete.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Memory ID to validate",
					},
					"trust": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"proposed", "validated", "proven", "disputed", "obsolete"},
						"description": "New trust level",
						"default":     "validated",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "cortex_learn_error",
			Description: "Store an error with its cause and solution. This is a specialized version of cortex_store for learning from mistakes.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"error": map[string]interface{}{
						"type":        "string",
						"description": "The error that occurred",
					},
					"cause": map[string]interface{}{
						"type":        "string",
						"description": "What caused the error",
					},
					"solution": map[string]interface{}{
						"type":        "string",
						"description": "How to fix or avoid this error",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Additional context (e.g., file, technology)",
					},
				},
				"required": []string{"error", "solution"},
			},
		},
	}

	s.sendResult(req.ID, ToolsListResult{Tools: tools})
}

func (s *Server) handleToolsCall(req *Request) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	ctx := context.Background()
	var result string
	var isError bool

	switch params.Name {
	case "cortex_store":
		result, isError = s.toolStore(ctx, params.Arguments)
	case "cortex_recall":
		result, isError = s.toolRecall(ctx, params.Arguments)
	case "cortex_relate":
		result, isError = s.toolRelate(ctx, params.Arguments)
	case "cortex_validate":
		result, isError = s.toolValidate(ctx, params.Arguments)
	case "cortex_learn_error":
		result, isError = s.toolLearnError(ctx, params.Arguments)
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}

	s.sendResult(req.ID, ToolResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
		IsError: isError,
	})
}

func (s *Server) toolStore(ctx context.Context, args map[string]interface{}) (string, bool) {
	content, _ := args["content"].(string)
	if content == "" {
		return "Error: content is required", true
	}

	opts := types.StoreOptions{
		Source: "agent:mcp",
		Trust:  types.TrustProposed,
	}

	if t, ok := args["type"].(string); ok {
		opts.Type = types.MemoryType(t)
	}
	if tk, ok := args["topic_key"].(string); ok {
		opts.TopicKey = tk
	}
	if tags, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if t, ok := tag.(string); ok {
				opts.Tags = append(opts.Tags, t)
			}
		}
	}

	memory, err := s.engine.Store(ctx, content, opts)
	if err != nil {
		return fmt.Sprintf("Error storing memory: %v", err), true
	}

	return fmt.Sprintf("Stored memory with ID: %s (topic: %s)", memory.ID, memory.TopicKey), false
}

func (s *Server) toolRecall(ctx context.Context, args map[string]interface{}) (string, bool) {
	query, _ := args["query"].(string)
	if query == "" {
		return "Error: query is required", true
	}

	opts := types.RecallOptions{
		Limit:       5,
		MinScore:    0.3,
		TrustLevels: []types.TrustLevel{types.TrustValidated, types.TrustProven},
	}

	if limit, ok := args["limit"].(float64); ok {
		opts.Limit = int(limit)
	}
	if t, ok := args["type"].(string); ok {
		opts.Types = []types.MemoryType{types.MemoryType(t)}
	}
	if includeProposed, ok := args["include_proposed"].(bool); ok && includeProposed {
		opts.TrustLevels = append(opts.TrustLevels, types.TrustProposed)
	}

	results, err := s.engine.Recall(ctx, query, opts)
	if err != nil {
		return fmt.Sprintf("Error searching: %v", err), true
	}

	if len(results) == 0 {
		return "No relevant memories found.", false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant memories:\n\n", len(results)))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] %s (%.0f%% relevant, trust: %s)\n",
			i+1, r.Memory.Type, r.Score*100, r.Memory.Trust))
		sb.WriteString(fmt.Sprintf("ID: %s\n", r.Memory.ID))
		if r.Memory.TopicKey != "" {
			sb.WriteString(fmt.Sprintf("Topic: %s\n", r.Memory.TopicKey))
		}
		sb.WriteString(fmt.Sprintf("Content: %s\n\n", r.Memory.Content))
	}

	return sb.String(), false
}

func (s *Server) toolRelate(ctx context.Context, args map[string]interface{}) (string, bool) {
	fromID, _ := args["from_id"].(string)
	toID, _ := args["to_id"].(string)
	relType, _ := args["relation"].(string)
	note, _ := args["note"].(string)

	if fromID == "" || toID == "" || relType == "" {
		return "Error: from_id, to_id, and relation are required", true
	}

	relation, err := s.engine.Relate(fromID, toID, types.RelationType(relType), note)
	if err != nil {
		return fmt.Sprintf("Error creating relation: %v", err), true
	}

	return fmt.Sprintf("Created relation: %s -[%s]-> %s", relation.FromID, relation.Type, relation.ToID), false
}

func (s *Server) toolValidate(ctx context.Context, args map[string]interface{}) (string, bool) {
	id, _ := args["id"].(string)
	if id == "" {
		return "Error: id is required", true
	}

	trust := types.TrustValidated
	if t, ok := args["trust"].(string); ok {
		trust = types.TrustLevel(t)
	}

	if err := s.engine.Validate(id, trust); err != nil {
		return fmt.Sprintf("Error updating trust: %v", err), true
	}

	return fmt.Sprintf("Updated memory %s trust to: %s", id, trust), false
}

func (s *Server) toolLearnError(ctx context.Context, args map[string]interface{}) (string, bool) {
	errorMsg, _ := args["error"].(string)
	cause, _ := args["cause"].(string)
	solution, _ := args["solution"].(string)
	context, _ := args["context"].(string)

	if errorMsg == "" || solution == "" {
		return "Error: error and solution are required", true
	}

	// Format the content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("ERROR: %s\n", errorMsg))
	if cause != "" {
		content.WriteString(fmt.Sprintf("CAUSE: %s\n", cause))
	}
	content.WriteString(fmt.Sprintf("SOLUTION: %s", solution))

	opts := types.StoreOptions{
		Type:   types.TypeError,
		Source: "agent:mcp:learn_error",
		Trust:  types.TrustProposed,
		Tags:   []string{"learned-error"},
	}

	if context != "" {
		opts.ExtraData = map[string]string{"context": context}
	}

	memory, err := s.engine.Store(ctx, content.String(), opts)
	if err != nil {
		return fmt.Sprintf("Error storing learned error: %v", err), true
	}

	return fmt.Sprintf("Learned error stored with ID: %s. Remember to validate it after confirming the solution works.", memory.ID), false
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
	s.send(resp)
}

func (s *Server) send(v interface{}) {
	data, _ := json.Marshal(v)
	fmt.Fprintln(s.writer, string(data))
}
