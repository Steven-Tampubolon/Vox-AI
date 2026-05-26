package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"vox-ai/internal/domain"
)

type DocumentStore struct {
	db *sql.DB
}

func NewDocumentStore(db *sql.DB) *DocumentStore {
	return &DocumentStore{db: db}
}

func (s *DocumentStore) Migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS documents (
		id              TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		filename        TEXT NOT NULL,
		chunk_count     INTEGER DEFAULT 0,
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chunks (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		document_id TEXT NOT NULL,
		content     TEXT NOT NULL,
		embedding   TEXT NOT NULL,
		FOREIGN KEY (document_id) REFERENCES documents(id)
	);`

	_, err := s.db.Exec(query)
	return err
}

// SaveDocument menyimpan metadata dokumen
func (s *DocumentStore) SaveDocument(ctx context.Context, doc *domain.Document) error {
	query := `
	INSERT INTO documents (id, conversation_id, filename, chunk_count, created_at)
	VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		doc.ID, doc.ConversationID, doc.Filename, doc.ChunkCount, doc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save document: %w", err)
	}
	return nil
}

// SaveChunk menyimpan satu chunk beserta embedding-nya
func (s *DocumentStore) SaveChunk(ctx context.Context, chunk *domain.Chunk) error {
	// Embedding disimpan sebagai JSON array di SQLite
	embJSON, err := json.Marshal(chunk.Embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	query := `INSERT INTO chunks (document_id, content, embedding) VALUES (?, ?, ?)`
	res, err := s.db.ExecContext(ctx, query, chunk.DocumentID, chunk.Content, string(embJSON))
	if err != nil {
		return fmt.Errorf("save chunk: %w", err)
	}

	id, _ := res.LastInsertId()
	chunk.ID = id
	return nil
}

// GetChunksByConversation mengambil semua chunk milik satu conversation
func (s *DocumentStore) GetChunksByConversation(ctx context.Context, conversationID string) ([]*domain.Chunk, error) {
	query := `
	SELECT c.id, c.document_id, c.content, c.embedding
	FROM chunks c
	JOIN documents d ON c.document_id = d.id
	WHERE d.conversation_id = ?`

	rows, err := s.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get chunks: %w", err)
	}
	defer rows.Close()

	var result []*domain.Chunk
	for rows.Next() {
		var chunk domain.Chunk
		var embJSON string

		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &embJSON); err != nil {
			return nil, err
		}

		// Parse JSON embedding kembali ke []float64
		if err := json.Unmarshal([]byte(embJSON), &chunk.Embedding); err != nil {
			return nil, fmt.Errorf("unmarshal embedding: %w", err)
		}

		result = append(result, &chunk)
	}
	return result, nil
}

// DeleteByConversation hapus semua dokumen + chunk milik satu conversation
func (s *DocumentStore) DeleteByConversation(ctx context.Context, conversationID string) error {
	_, err := s.db.ExecContext(ctx, `
	DELETE FROM chunks WHERE document_id IN (
		SELECT id FROM documents WHERE conversation_id = ?
	)`, conversationID)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx,
		`DELETE FROM documents WHERE conversation_id = ?`, conversationID,
	)
	return err
}
