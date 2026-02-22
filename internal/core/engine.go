// Package core provides the main Cortex engine
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/constantino-dev/cortex/internal/db"
	"github.com/constantino-dev/cortex/internal/embeddings"
	"github.com/constantino-dev/cortex/pkg/types"
)

// Engine is the main Cortex engine that coordinates all services
type Engine struct {
	db       *db.DB
	embedder embeddings.Provider
	config   *types.Config
}

// New creates a new Cortex engine
func New(cfg *types.Config) (*Engine, error) {
	// Ensure data directory exists
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize database
	database, err := db.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize embedding provider
	var embedder embeddings.Provider
	switch cfg.EmbeddingProvider {
	case "openai", "":
		if cfg.OpenAIKey == "" {
			database.Close()
			return nil, fmt.Errorf("OpenAI API key required")
		}
		embedder = embeddings.NewOpenAI(cfg.OpenAIKey)
	case "ollama":
		// TODO: implement Ollama provider
		database.Close()
		return nil, fmt.Errorf("ollama provider not yet implemented")
	default:
		database.Close()
		return nil, fmt.Errorf("unknown embedding provider: %s", cfg.EmbeddingProvider)
	}

	return &Engine{
		db:       database,
		embedder: embedder,
		config:   cfg,
	}, nil
}

// Close shuts down the engine
func (e *Engine) Close() error {
	return e.db.Close()
}

// Store saves a new memory or updates an existing one (if TopicKey matches)
func (e *Engine) Store(ctx context.Context, content string, opts types.StoreOptions) (*types.Memory, error) {
	// Check if we should update existing memory by topic key
	var existing *types.Memory
	if opts.TopicKey != "" {
		existing, _ = e.db.GetMemoryByTopicKey(opts.TopicKey)
	}

	var memory *types.Memory
	if existing != nil {
		// Update existing memory (topic key evolution)
		memory = existing
		memory.Content = content
		memory.UpdatedAt = timeNow()
		if opts.Tags != nil {
			memory.Tags = opts.Tags
		}
		if opts.Type != "" {
			memory.Type = opts.Type
		}
		if opts.Trust != "" {
			memory.Trust = opts.Trust
		}
	} else {
		// Create new memory
		memory = &types.Memory{
			ID:        generateID(),
			Content:   content,
			Type:      opts.Type,
			TopicKey:  opts.TopicKey,
			Tags:      opts.Tags,
			Trust:     opts.Trust,
			CreatedAt: timeNow(),
			UpdatedAt: timeNow(),
			AccessCnt: 0,
			Metadata: types.Metadata{
				Source:  opts.Source,
				Project: opts.Project,
				ExtraData: opts.ExtraData,
			},
		}

		// Set defaults
		if memory.Type == "" {
			memory.Type = types.TypeGeneral
		}
		if memory.Trust == "" {
			memory.Trust = types.TrustProposed
		}
	}

	// Save to database
	if err := e.db.SaveMemory(memory); err != nil {
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}

	// Generate and save embedding
	embedding, err := e.embedder.Embed(ctx, content)
	if err != nil {
		// Log but don't fail - memory is still saved
		fmt.Fprintf(os.Stderr, "warning: failed to generate embedding: %v\n", err)
	} else {
		if err := e.db.SaveEmbedding(memory.ID, embedding, e.embedder.Model()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save embedding: %v\n", err)
		}
	}

	return memory, nil
}

// Recall searches for relevant memories
func (e *Engine) Recall(ctx context.Context, query string, opts types.RecallOptions) ([]types.SearchResult, error) {
	// Set defaults
	if opts.Limit == 0 {
		opts.Limit = 5
	}
	if opts.MinScore == 0 {
		opts.MinScore = 0.3
	}
	if len(opts.TrustLevels) == 0 {
		// By default, only return validated+ memories
		opts.TrustLevels = []types.TrustLevel{
			types.TrustValidated,
			types.TrustProven,
		}
	}

	// Generate query embedding
	queryEmb, err := e.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Perform vector search
	vecResults, err := e.db.VectorSearch(queryEmb, opts.Limit*2, opts.TrustLevels)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Perform FTS search for keyword matching
	ftsIDs, _ := e.db.FTSSearch(query, opts.Limit*2)
	ftsSet := make(map[string]bool)
	for _, id := range ftsIDs {
		ftsSet[id] = true
	}

	// Combine results with hybrid scoring
	var results []types.SearchResult
	seen := make(map[string]bool)

	for _, vr := range vecResults {
		if seen[vr.MemoryID] {
			continue
		}
		seen[vr.MemoryID] = true

		memory, err := e.db.GetMemory(vr.MemoryID)
		if err != nil || memory == nil {
			continue
		}

		// Calculate hybrid score
		// Convert L2 distance to similarity (0-1)
		semanticScore := 1.0 - (vr.Distance / 2.0)
		if semanticScore < 0 {
			semanticScore = 0
		}

		// Keyword boost
		keywordBoost := 0.0
		if ftsSet[vr.MemoryID] {
			keywordBoost = 0.15
		}

		// Final score: 70% semantic + 30% keyword potential + boost
		finalScore := semanticScore*0.7 + keywordBoost

		if finalScore < opts.MinScore {
			continue
		}

		// Increment access count
		e.db.IncrementAccessCount(memory.ID)

		results = append(results, types.SearchResult{
			Memory:    *memory,
			Score:     finalScore,
			MatchType: "hybrid",
		})
	}

	// Limit results
	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// Get retrieves a specific memory by ID
func (e *Engine) Get(id string) (*types.Memory, error) {
	return e.db.GetMemory(id)
}

// List returns memories matching filters
func (e *Engine) List(opts types.RecallOptions) ([]*types.Memory, error) {
	return e.db.ListMemories(opts)
}

// Delete removes a memory
func (e *Engine) Delete(id string) error {
	return e.db.DeleteMemory(id)
}

// Validate updates the trust level of a memory
func (e *Engine) Validate(id string, trust types.TrustLevel) error {
	return e.db.UpdateTrust(id, trust)
}

// Relate creates a relation between two memories
func (e *Engine) Relate(fromID, toID string, relType types.RelationType, note string) (*types.Relation, error) {
	// Verify both memories exist
	from, err := e.db.GetMemory(fromID)
	if err != nil || from == nil {
		return nil, fmt.Errorf("source memory not found: %s", fromID)
	}
	to, err := e.db.GetMemory(toID)
	if err != nil || to == nil {
		return nil, fmt.Errorf("target memory not found: %s", toID)
	}

	relation := &types.Relation{
		ID:        generateID(),
		FromID:    fromID,
		ToID:      toID,
		Type:      relType,
		Note:      note,
		CreatedAt: timeNow(),
	}

	if err := e.db.SaveRelation(relation); err != nil {
		return nil, fmt.Errorf("failed to save relation: %w", err)
	}

	return relation, nil
}

// GetRelations returns all relations for a memory
func (e *Engine) GetRelations(memoryID string) ([]*types.Relation, error) {
	from, err := e.db.GetRelationsFrom(memoryID)
	if err != nil {
		return nil, err
	}
	to, err := e.db.GetRelationsTo(memoryID)
	if err != nil {
		return nil, err
	}
	return append(from, to...), nil
}

// Stats returns engine statistics
func (e *Engine) Stats() (map[string]int, error) {
	return e.db.Stats()
}
