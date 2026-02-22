// Package db provides SQLite storage for Cortex
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/constantino-dev/cortex/pkg/types"
	_ "github.com/mattn/go-sqlite3"
	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes schema
func New(path string) (*DB, error) {
	// Register sqlite-vec extension
	sqlite_vec.Auto()

	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates the database schema
func (db *DB) migrate() error {
	schema := `
	-- Memories table
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'general',
		topic_key TEXT,
		tags TEXT, -- JSON array
		trust TEXT NOT NULL DEFAULT 'proposed',
		metadata TEXT, -- JSON object
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		access_count INTEGER DEFAULT 0
	);

	-- Index for topic_key lookups and evolution
	CREATE INDEX IF NOT EXISTS idx_memories_topic_key ON memories(topic_key);
	CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
	CREATE INDEX IF NOT EXISTS idx_memories_trust ON memories(trust);
	CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at);

	-- Relations table
	CREATE TABLE IF NOT EXISTS relations (
		id TEXT PRIMARY KEY,
		from_id TEXT NOT NULL,
		to_id TEXT NOT NULL,
		type TEXT NOT NULL,
		note TEXT,
		created_at TEXT NOT NULL,
		FOREIGN KEY (from_id) REFERENCES memories(id) ON DELETE CASCADE,
		FOREIGN KEY (to_id) REFERENCES memories(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_relations_from ON relations(from_id);
	CREATE INDEX IF NOT EXISTS idx_relations_to ON relations(to_id);
	CREATE INDEX IF NOT EXISTS idx_relations_type ON relations(type);

	-- Embeddings cache
	CREATE TABLE IF NOT EXISTS embeddings (
		memory_id TEXT PRIMARY KEY,
		embedding BLOB NOT NULL, -- float32 array as blob
		model TEXT NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (memory_id) REFERENCES memories(id) ON DELETE CASCADE
	);

	-- Virtual table for vector search (sqlite-vec)
	CREATE VIRTUAL TABLE IF NOT EXISTS vec_memories USING vec0(
		memory_id TEXT PRIMARY KEY,
		embedding float[1536]
	);

	-- Full-text search for keyword matching
	CREATE VIRTUAL TABLE IF NOT EXISTS fts_memories USING fts5(
		content,
		topic_key,
		tags,
		content=memories,
		content_rowid=rowid
	);

	-- Triggers to keep FTS in sync
	CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO fts_memories(rowid, content, topic_key, tags)
		VALUES (new.rowid, new.content, new.topic_key, new.tags);
	END;

	CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO fts_memories(fts_memories, rowid, content, topic_key, tags)
		VALUES('delete', old.rowid, old.content, old.topic_key, old.tags);
	END;

	CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
		INSERT INTO fts_memories(fts_memories, rowid, content, topic_key, tags)
		VALUES('delete', old.rowid, old.content, old.topic_key, old.tags);
		INSERT INTO fts_memories(rowid, content, topic_key, tags)
		VALUES (new.rowid, new.content, new.topic_key, new.tags);
	END;
	`

	_, err := db.conn.Exec(schema)
	return err
}

// SaveMemory stores or updates a memory
func (db *DB) SaveMemory(m *types.Memory) error {
	tagsJSON, _ := json.Marshal(m.Tags)
	metaJSON, _ := json.Marshal(m.Metadata)

	query := `
		INSERT INTO memories (id, content, type, topic_key, tags, trust, metadata, created_at, updated_at, access_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			type = excluded.type,
			topic_key = excluded.topic_key,
			tags = excluded.tags,
			trust = excluded.trust,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at,
			access_count = excluded.access_count
	`

	_, err := db.conn.Exec(query,
		m.ID, m.Content, m.Type, m.TopicKey, string(tagsJSON),
		m.Trust, string(metaJSON), m.CreatedAt.Format(time.RFC3339),
		m.UpdatedAt.Format(time.RFC3339), m.AccessCnt,
	)
	return err
}

// GetMemory retrieves a memory by ID
func (db *DB) GetMemory(id string) (*types.Memory, error) {
	query := `SELECT id, content, type, topic_key, tags, trust, metadata, created_at, updated_at, access_count
			  FROM memories WHERE id = ?`

	row := db.conn.QueryRow(query, id)
	return db.scanMemory(row)
}

// GetMemoryByTopicKey retrieves a memory by topic key
func (db *DB) GetMemoryByTopicKey(topicKey string) (*types.Memory, error) {
	query := `SELECT id, content, type, topic_key, tags, trust, metadata, created_at, updated_at, access_count
			  FROM memories WHERE topic_key = ? ORDER BY updated_at DESC LIMIT 1`

	row := db.conn.QueryRow(query, topicKey)
	return db.scanMemory(row)
}

// scanMemory scans a row into a Memory struct
func (db *DB) scanMemory(row *sql.Row) (*types.Memory, error) {
	var m types.Memory
	var tagsJSON, metaJSON, createdStr, updatedStr string
	var topicKey sql.NullString

	err := row.Scan(&m.ID, &m.Content, &m.Type, &topicKey, &tagsJSON, &m.Trust, &metaJSON, &createdStr, &updatedStr, &m.AccessCnt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if topicKey.Valid {
		m.TopicKey = topicKey.String
	}
	json.Unmarshal([]byte(tagsJSON), &m.Tags)
	json.Unmarshal([]byte(metaJSON), &m.Metadata)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

	return &m, nil
}

// ListMemories returns memories matching the given filters
func (db *DB) ListMemories(opts types.RecallOptions) ([]*types.Memory, error) {
	var conditions []string
	var args []interface{}

	if len(opts.Types) > 0 {
		placeholders := make([]string, len(opts.Types))
		for i, t := range opts.Types {
			placeholders[i] = "?"
			args = append(args, t)
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(opts.TrustLevels) > 0 {
		placeholders := make([]string, len(opts.TrustLevels))
		for i, t := range opts.TrustLevels {
			placeholders[i] = "?"
			args = append(args, t)
		}
		conditions = append(conditions, fmt.Sprintf("trust IN (%s)", strings.Join(placeholders, ",")))
	}

	if opts.Project != "" {
		conditions = append(conditions, "json_extract(metadata, '$.project') = ?")
		args = append(args, opts.Project)
	}

	if opts.TopicKey != "" {
		conditions = append(conditions, "topic_key LIKE ?")
		args = append(args, opts.TopicKey+"%")
	}

	query := "SELECT id, content, type, topic_key, tags, trust, metadata, created_at, updated_at, access_count FROM memories"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY updated_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*types.Memory
	for rows.Next() {
		var m types.Memory
		var tagsJSON, metaJSON, createdStr, updatedStr string
		var topicKey sql.NullString

		err := rows.Scan(&m.ID, &m.Content, &m.Type, &topicKey, &tagsJSON, &m.Trust, &metaJSON, &createdStr, &updatedStr, &m.AccessCnt)
		if err != nil {
			return nil, err
		}

		if topicKey.Valid {
			m.TopicKey = topicKey.String
		}
		json.Unmarshal([]byte(tagsJSON), &m.Tags)
		json.Unmarshal([]byte(metaJSON), &m.Metadata)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

		memories = append(memories, &m)
	}

	return memories, nil
}

// DeleteMemory removes a memory by ID
func (db *DB) DeleteMemory(id string) error {
	_, err := db.conn.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

// IncrementAccessCount increments the access count for a memory
func (db *DB) IncrementAccessCount(id string) error {
	_, err := db.conn.Exec("UPDATE memories SET access_count = access_count + 1 WHERE id = ?", id)
	return err
}

// UpdateTrust updates the trust level of a memory
func (db *DB) UpdateTrust(id string, trust types.TrustLevel) error {
	_, err := db.conn.Exec("UPDATE memories SET trust = ?, updated_at = ? WHERE id = ?",
		trust, time.Now().Format(time.RFC3339), id)
	return err
}

// SaveRelation stores a relation between two memories
func (db *DB) SaveRelation(r *types.Relation) error {
	query := `INSERT INTO relations (id, from_id, to_id, type, note, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, r.ID, r.FromID, r.ToID, r.Type, r.Note, r.CreatedAt.Format(time.RFC3339))
	return err
}

// GetRelationsFrom returns all relations starting from a memory
func (db *DB) GetRelationsFrom(memoryID string) ([]*types.Relation, error) {
	return db.getRelations("from_id = ?", memoryID)
}

// GetRelationsTo returns all relations pointing to a memory
func (db *DB) GetRelationsTo(memoryID string) ([]*types.Relation, error) {
	return db.getRelations("to_id = ?", memoryID)
}

func (db *DB) getRelations(condition string, arg interface{}) ([]*types.Relation, error) {
	query := fmt.Sprintf("SELECT id, from_id, to_id, type, note, created_at FROM relations WHERE %s", condition)
	rows, err := db.conn.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []*types.Relation
	for rows.Next() {
		var r types.Relation
		var createdStr string
		var note sql.NullString

		err := rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &note, &createdStr)
		if err != nil {
			return nil, err
		}

		if note.Valid {
			r.Note = note.String
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		relations = append(relations, &r)
	}

	return relations, nil
}

// DeleteRelation removes a relation by ID
func (db *DB) DeleteRelation(id string) error {
	_, err := db.conn.Exec("DELETE FROM relations WHERE id = ?", id)
	return err
}

// SaveEmbedding stores an embedding for a memory
func (db *DB) SaveEmbedding(memoryID string, embedding []float32, model string) error {
	// Save to embeddings table
	embBytes := float32ToBytes(embedding)
	_, err := db.conn.Exec(`
		INSERT INTO embeddings (memory_id, embedding, model, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(memory_id) DO UPDATE SET
			embedding = excluded.embedding,
			model = excluded.model,
			created_at = excluded.created_at
	`, memoryID, embBytes, model, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Save to vec_memories for vector search
	// sqlite-vec virtual tables don't support ON CONFLICT, so delete first
	db.conn.Exec(`DELETE FROM vec_memories WHERE memory_id = ?`, memoryID)
	_, err = db.conn.Exec(`
		INSERT INTO vec_memories (memory_id, embedding)
		VALUES (?, ?)
	`, memoryID, serializeVector(embedding))

	return err
}

// GetEmbedding retrieves an embedding for a memory
func (db *DB) GetEmbedding(memoryID string) ([]float32, error) {
	var embBytes []byte
	err := db.conn.QueryRow("SELECT embedding FROM embeddings WHERE memory_id = ?", memoryID).Scan(&embBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return bytesToFloat32(embBytes), nil
}

// VectorSearch performs semantic search using sqlite-vec
func (db *DB) VectorSearch(queryEmb []float32, limit int) ([]struct {
	MemoryID string
	Distance float64
}, error) {
	// sqlite-vec requires k=? constraint for KNN queries
	query := `
		SELECT memory_id, distance
		FROM vec_memories
		WHERE embedding MATCH ? AND k = ?
	`

	rows, err := db.conn.Query(query, serializeVector(queryEmb), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		MemoryID string
		Distance float64
	}
	for rows.Next() {
		var r struct {
			MemoryID string
			Distance float64
		}
		if err := rows.Scan(&r.MemoryID, &r.Distance); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// FTSSearch performs full-text search
func (db *DB) FTSSearch(query string, limit int) ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT m.id
		FROM fts_memories f
		JOIN memories m ON f.rowid = m.rowid
		WHERE fts_memories MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// Stats returns database statistics
func (db *DB) Stats() (map[string]int, error) {
	stats := make(map[string]int)

	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM memories").Scan(&count)
	stats["memories"] = count

	db.conn.QueryRow("SELECT COUNT(*) FROM relations").Scan(&count)
	stats["relations"] = count

	db.conn.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	stats["embeddings"] = count

	return stats, nil
}

// Helper functions for embedding serialization
func float32ToBytes(floats []float32) []byte {
	bytes := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := *(*uint32)(unsafe.Pointer(&f))
		bytes[i*4] = byte(bits)
		bytes[i*4+1] = byte(bits >> 8)
		bytes[i*4+2] = byte(bits >> 16)
		bytes[i*4+3] = byte(bits >> 24)
	}
	return bytes
}

func bytesToFloat32(bytes []byte) []float32 {
	floats := make([]float32, len(bytes)/4)
	for i := range floats {
		bits := uint32(bytes[i*4]) | uint32(bytes[i*4+1])<<8 | uint32(bytes[i*4+2])<<16 | uint32(bytes[i*4+3])<<24
		floats[i] = *(*float32)(unsafe.Pointer(&bits))
	}
	return floats
}

func serializeVector(v []float32) string {
	// sqlite-vec expects JSON array format
	b, _ := json.Marshal(v)
	return string(b)
}
