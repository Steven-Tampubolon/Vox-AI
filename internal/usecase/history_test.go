package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
)

// TestBuildHistory_TruncatesTo20Messages memverifikasi maxHistoryMessages
// (constant baru dari refactor step 6) tetap membatasi 20 pesan terakhir,
// perilaku yang sama seperti sebelum diekstrak ke chat_helpers.go.
func TestBuildHistory_TruncatesTo20Messages(t *testing.T) {
	chatRepo := newMockChatRepo()
	chatRepo.conversations["conv-1"] = &domain.Conversation{ID: "conv-1", Character: domain.CharacterExplain}

	// isi 25 pesan lama di conversation
	for i := 0; i < 25; i++ {
		role := domain.RoleUser
		if i%2 == 1 {
			role = domain.RoleAssistant
		}
		chatRepo.messages["conv-1"] = append(chatRepo.messages["conv-1"], &domain.Message{
			ConversationID: "conv-1",
			Role:           role,
			Content:        "pesan lama",
			CreatedAt:      time.Now(),
		})
	}

	var gotHistoryLen int
	aiRepo := &mockAIRepo{
		generateFn: func(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
			gotHistoryLen = len(history)
			return "jawaban", nil
		},
	}
	uc := usecase.NewExplainUseCase(aiRepo, chatRepo)

	_, err := uc.Chat(context.Background(), &domain.ChatRequest{ConversationID: "conv-1", Message: "pesan baru"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 25 pesan lama, dipotong ke 20 terakhir (pesan user baru belum masuk history
	// karena history dibangun sebelum dikirim ke Generate)
	if gotHistoryLen != 20 {
		t.Errorf("expected history truncated to 20 messages, got %d", gotHistoryLen)
	}
}
