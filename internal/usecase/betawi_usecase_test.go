package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
)

func TestBetawiChat_CreatesNewConversationWhenIDEmpty(t *testing.T) {
	chatRepo := newMockChatRepo()
	aiRepo := &mockAIRepo{
		generateFn: func(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
			return "Aye, gimana kabarnya bro?", nil
		},
	}
	uc := usecase.NewBetawiUseCase(aiRepo, chatRepo)

	res, err := uc.Chat(context.Background(), &domain.ChatRequest{Message: "Halo bang"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Character != domain.CharacterBetawi {
		t.Errorf("expected character betawi, got %s", res.Character)
	}
	if res.ConversationID == "" {
		t.Errorf("expected new conversation to be created")
	}
	if len(chatRepo.messages[res.ConversationID]) != 2 {
		t.Errorf("expected 2 messages saved (user+ai), got %d", len(chatRepo.messages[res.ConversationID]))
	}
}

func TestBetawiChat_ReusesExistingConversation_SameCharacter(t *testing.T) {
	chatRepo := newMockChatRepo()
	chatRepo.conversations["conv-1"] = &domain.Conversation{ID: "conv-1", Character: domain.CharacterBetawi}
	aiRepo := &mockAIRepo{
		generateFn: func(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
			return "balasan", nil
		},
	}
	uc := usecase.NewBetawiUseCase(aiRepo, chatRepo)

	res, err := uc.Chat(context.Background(), &domain.ChatRequest{ConversationID: "conv-1", Message: "lanjut"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ConversationID != "conv-1" {
		t.Errorf("expected existing conversation conv-1 reused, got %s", res.ConversationID)
	}
}

func TestBetawiChat_WrongCharacterConversation_ReturnsError(t *testing.T) {
	chatRepo := newMockChatRepo()
	// conversation ini sebenarnya milik karakter git
	chatRepo.conversations["conv-1"] = &domain.Conversation{ID: "conv-1", Character: domain.CharacterGit}
	uc := usecase.NewBetawiUseCase(&mockAIRepo{}, chatRepo)

	_, err := uc.Chat(context.Background(), &domain.ChatRequest{ConversationID: "conv-1", Message: "halo"})
	if err == nil {
		t.Fatalf("expected error karena conversation milik karakter lain, got nil")
	}
	if !strings.Contains(err.Error(), "bukan betawi") {
		t.Errorf("expected error message mention 'bukan betawi', got: %v", err)
	}
}

// TestBetawiChatStream_SaveErrorIsWrapped adalah regression test untuk bug
// fix step 2: sebelumnya SaveMessage gagal mem-wrap `err` (selalu nil),
// bukan `saveErr` yang berisi error asli.
func TestBetawiChatStream_SaveErrorIsWrapped(t *testing.T) {
	wantErr := errors.New("disk penuh")
	chatRepo := &failingSaveChatRepo{mockChatRepo: newMockChatRepo(), failOnNthSave: 2, failErr: wantErr}
	aiRepo := &mockAIRepo{
		streamFn: func(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
			return onChunk("halo")
		},
	}
	uc := usecase.NewBetawiUseCase(aiRepo, chatRepo)

	_, err := uc.ChatStream(context.Background(), &domain.ChatRequest{Message: "halo"}, func(string) error { return nil })
	if err == nil {
		t.Fatalf("expected error to propagate from failed SaveMessage")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error to be wantErr (via errors.Is), got: %v", err)
	}
}

// failingSaveChatRepo membuat SaveMessage gagal pada panggilan ke-N,
// untuk mensimulasikan gagal simpan balasan AI (bukan pesan user).
type failingSaveChatRepo struct {
	*mockChatRepo
	callCount     int
	failOnNthSave int
	failErr       error
}

func (f *failingSaveChatRepo) SaveMessage(ctx context.Context, msg *domain.Message) error {
	f.callCount++
	if f.callCount == f.failOnNthSave {
		return f.failErr
	}
	return f.mockChatRepo.SaveMessage(ctx, msg)
}
