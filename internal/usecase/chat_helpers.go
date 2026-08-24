package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/google/uuid"
)

// maxHistoryMessages adalah jumlah maksimum pesan terakhir yang dikirim
// sebagai konteks percakapan ke Gemini.
const maxHistoryMessages = 20

// getOrCreateConversation mengambil conversation yang sudah ada (sekaligus
// memvalidasi kepemilikan karakternya), atau membuat conversation baru jika
// req.ConversationID kosong.
//
// Sebelumnya logic ini diduplikasi identik di betawi_usecase.go,
// git_usecase.go, explain_usecase.go, dan rag_usecase.go sebagai
// getOrCreate<X>Conversation. Sekarang jadi satu source of truth.
func getOrCreateConversation(
	ctx context.Context,
	req *domain.ChatRequest,
	char domain.Character,
	chatRepo repository.ChatRepository,
) (*domain.Conversation, error) {
	if req.ConversationID != "" {
		conv, err := chatRepo.GetConversation(ctx, req.ConversationID)
		if err != nil {
			return nil, err
		}
		if conv != nil {
			// Validasi karakter - conversation harus milik karakter yang sama
			if conv.Character != char {
				return nil, fmt.Errorf(
					"conversation ini milik karakter %s, bukan %s", conv.Character, char,
				)
			}
			return conv, nil
		}
	}

	now := time.Now()
	conv := &domain.Conversation{
		ID:        uuid.New().String(),
		Character: char,
		Title:     truncate(req.Message, 40),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := chatRepo.SaveConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// buildHistory mengubah messages dari DB menjadi format Gemini, dibatasi
// maxHistoryMessages pesan terakhir.
//
// Sebelumnya logic ini diduplikasi identik di betawi_usecase.go,
// git_usecase.go, explain_usecase.go (dan hampir identik di rag_usecase.go)
// sebagai build<X>History. Sekarang jadi satu source of truth.
func buildHistory(
	ctx context.Context,
	conversationID string,
	chatRepo repository.ChatRepository,
) ([]repository.ChatMessage, error) {
	messages, err := chatRepo.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Batasi maxHistoryMessages pesan terakhir
	if len(messages) > maxHistoryMessages {
		messages = messages[len(messages)-maxHistoryMessages:]
	}

	var history []repository.ChatMessage
	for _, msg := range messages {
		role := "user"
		if msg.Role == domain.RoleAssistant {
			role = "model"
		}
		history = append(history, repository.ChatMessage{
			Role:    role,
			Content: msg.Content,
		})
	}
	return history, nil
}
