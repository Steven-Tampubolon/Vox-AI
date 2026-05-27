package repository

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/sqlite"
	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
)

// ChatRepository adalah kontrak untuk menyimpan dan mengambil data chat
type ChatRepository interface {
	SaveConversation(ctx context.Context, conv *domain.Conversation) error
	GetConversation(ctx context.Context, id string) (*domain.Conversation, error)
	ListConversations(ctx context.Context) ([]*domain.Conversation, error)
	SaveMessage(ctx context.Context, msg *domain.Message) error
	GetMessages(ctx context.Context, conversationID string) ([]*domain.Message, error)
}

// SQLiteChatRepository adalah implementasi ChatRepository menggunakan SQLite
type SQLiteChatRepository struct {
	store *sqlite.ChatStore
}

func NewSQLiteChatRepository(store *sqlite.ChatStore) ChatRepository {
	return &SQLiteChatRepository{store: store}
}

func (r *SQLiteChatRepository) SaveConversation(ctx context.Context, conv *domain.Conversation) error {
	return r.store.SaveConversation(ctx, conv)

}

func (r *SQLiteChatRepository) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	return r.store.GetConversation(ctx, id)
}

func (r *SQLiteChatRepository) ListConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return r.store.ListConversations(ctx)
}

func (r *SQLiteChatRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	return r.store.SaveMessage(ctx, msg)
}

func (r *SQLiteChatRepository) GetMessages(ctx context.Context, conversationID string) ([]*domain.Message, error) {
	return r.store.GetMessages(ctx, conversationID)
}
