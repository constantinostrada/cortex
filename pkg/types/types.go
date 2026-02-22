// Package types defines the core data structures for Cortex
package types

import (
	"time"
)

// TrustLevel represents the validation state of a memory
type TrustLevel string

const (
	TrustProposed  TrustLevel = "proposed"  // Agent suggested, not validated
	TrustValidated TrustLevel = "validated" // Human confirmed or used successfully
	TrustProven    TrustLevel = "proven"    // Multiple successful uses
	TrustDisputed  TrustLevel = "disputed"  // Someone questioned it
	TrustObsolete  TrustLevel = "obsolete"  // No longer applies
)

// MemoryType categorizes what kind of knowledge this is
type MemoryType string

const (
	TypeGeneral   MemoryType = "general"   // Default
	TypeError     MemoryType = "error"     // Something that failed
	TypePattern   MemoryType = "pattern"   // Reusable solution
	TypeDecision  MemoryType = "decision"  // Why something was chosen
	TypeContext   MemoryType = "context"   // Project state/info
	TypeProcedure MemoryType = "procedure" // How to do something
)

// Memory represents a single piece of stored knowledge
type Memory struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	Type      MemoryType `json:"type"`
	TopicKey  string     `json:"topic_key,omitempty"` // e.g., "react/hooks/rules"
	Tags      []string   `json:"tags,omitempty"`
	Trust     TrustLevel `json:"trust"`
	Metadata  Metadata   `json:"metadata,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	AccessCnt int        `json:"access_count"`
}

// Metadata holds optional extra information about a memory
type Metadata struct {
	Source    string            `json:"source,omitempty"`    // Where this came from
	Project   string            `json:"project,omitempty"`   // Which project it belongs to
	Author    string            `json:"author,omitempty"`    // Who created it (human/agent)
	ExtraData map[string]string `json:"extra,omitempty"`     // Arbitrary key-value pairs
}

// RelationType defines how two memories are connected
type RelationType string

const (
	RelCauses     RelationType = "causes"      // A causes B
	RelSolves     RelationType = "solves"      // A solves B
	RelReplaces   RelationType = "replaces"    // A replaces B
	RelRequires   RelationType = "requires"    // A requires B
	RelRelatedTo  RelationType = "related_to"  // A is related to B
	RelPartOf     RelationType = "part_of"     // A is part of B
	RelContradicts RelationType = "contradicts" // A contradicts B
)

// Relation represents a connection between two memories
type Relation struct {
	ID        string       `json:"id"`
	FromID    string       `json:"from_id"`
	ToID      string       `json:"to_id"`
	Type      RelationType `json:"type"`
	Note      string       `json:"note,omitempty"` // Optional explanation
	CreatedAt time.Time    `json:"created_at"`
}

// SearchResult wraps a memory with its relevance score
type SearchResult struct {
	Memory    Memory  `json:"memory"`
	Score     float64 `json:"score"`      // 0.0 - 1.0
	MatchType string  `json:"match_type"` // "semantic", "keyword", "hybrid"
}

// StoreOptions configures how a memory is stored
type StoreOptions struct {
	TopicKey   string            // If set, updates existing memory with same topic_key
	Tags       []string          // Tags for categorization
	Type       MemoryType        // Type of memory
	Trust      TrustLevel        // Initial trust level
	Project    string            // Project scope
	Source     string            // Origin (e.g., "cli", "agent:claude")
	ExtraData  map[string]string // Additional metadata
}

// RecallOptions configures how memories are searched
type RecallOptions struct {
	Limit      int        // Max results (default: 5)
	MinScore   float64    // Minimum relevance score (default: 0.3)
	Types      []MemoryType // Filter by type
	Tags       []string   // Filter by tags
	TrustLevels []TrustLevel // Filter by trust (default: validated+)
	Project    string     // Filter by project
	TopicKey   string     // Filter by topic key prefix
}

// Config holds Cortex configuration
type Config struct {
	DBPath           string `json:"db_path"`
	EmbeddingProvider string `json:"embedding_provider"` // "openai" or "ollama"
	OpenAIKey        string `json:"openai_key,omitempty"`
	OllamaURL        string `json:"ollama_url,omitempty"`
	OllamaModel      string `json:"ollama_model,omitempty"`
	DefaultProject   string `json:"default_project,omitempty"`
}
