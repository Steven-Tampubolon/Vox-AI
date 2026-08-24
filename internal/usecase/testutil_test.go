package usecase_test

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

// mockChatRepo adalah in-memory fake untuk repository.ChatRepository,
// dipakai bersama oleh test betawi/git/explain/rag usecase.
type mockChatRepo struct {
	conversations map[string]*domain.Conversation
	messages      map[string][]*domain.Message
}

func newMockChatRepo() *mockChatRepo {
	return &mockChatRepo{
		conversations: map[string]*domain.Conversation{},
		messages:      map[string][]*domain.Message{},
	}
}

func (m *mockChatRepo) SaveConversation(ctx context.Context, conv *domain.Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}

func (m *mockChatRepo) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	// meniru perilaku SQLiteChatRepository: id tidak ketemu -> (nil, nil), bukan error
	return m.conversations[id], nil
}

func (m *mockChatRepo) ListConversations(ctx context.Context) ([]*domain.Conversation, error) {
	return nil, nil
}

func (m *mockChatRepo) SaveMessage(ctx context.Context, msg *domain.Message) error {
	m.messages[msg.ConversationID] = append(m.messages[msg.ConversationID], msg)
	return nil
}

func (m *mockChatRepo) GetMessages(ctx context.Context, conversationID string) ([]*domain.Message, error) {
	return m.messages[conversationID], nil
}

func (m *mockChatRepo) DeleteConversation(ctx context.Context, id string) error { return nil }

func (m *mockChatRepo) UpdateConversationTitle(ctx context.Context, id, title string) error {
	return nil
}

func (m *mockChatRepo) ListConversationsByCharacter(ctx context.Context, character string) ([]*domain.Conversation, error) {
	return nil, nil
}

// mockAIRepo adalah fake untuk repository.AIRepository (tipe ChatMessage
// dari hasil refactor step 9 - lihat internal/repository/ai_repository.go).
type mockAIRepo struct {
	generateFn func(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error)
	streamFn   func(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error
}

func (m *mockAIRepo) Generate(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
	return m.generateFn(ctx, systemPrompt, history)
}

func (m *mockAIRepo) GenerateStream(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
	return m.streamFn(ctx, systemPrompt, history, onChunk)
}

func (m *mockAIRepo) Embed(ctx context.Context, text string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3}, nil
}
