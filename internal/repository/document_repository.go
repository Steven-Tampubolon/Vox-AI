package repository

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/sqlite"
	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
)

// DocumentRepository adlah kontrak untuk menyimpan dan mengambil dokumen dan embedding RAG
type DocumentRepository interface {
	SaveDocument(ctx context.Context, doc *domain.Document) error
	SaveChunk(ctx context.Context, chunk *domain.Chunk) error
	GetChunksByConversation(ctx context.Context, conversationID string) ([]*domain.Chunk, error)
	DeleteByConversation(ctx context.Context, coversationID string) error
}

// SQLiteDocumentRepository adalah implementasi DocumentRepository menggunakan SQLite
type SQLiteDocumentRepository struct {
	store *sqlite.DocumentStore
}

func NewSQLiteDocumentRepository(store *sqlite.DocumentStore) DocumentRepository {
	return &SQLiteDocumentRepository{store: store}
}

func (r *SQLiteDocumentRepository) SaveDocument(ctx context.Context, doc *domain.Document) error {
	return r.store.SaveDocument(ctx, doc)
}

func (r *SQLiteDocumentRepository) SaveChunk(ctx context.Context, chunk *domain.Chunk) error {
	return r.store.SaveChunk(ctx, chunk)
}

func (r *SQLiteDocumentRepository) GetChunksByConversation(ctx context.Context, conversationID string) ([]*domain.Chunk, error) {
	return r.store.GetChunksByConversation(ctx, conversationID)
}

func (r *SQLiteDocumentRepository) DeleteByConversation(ctx context.Context, conversationID string) error {
	return r.store.DeleteByConversation(ctx, conversationID)
}
