package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/gemini"
	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/google/uuid"
)

const explainSystemPrompt = `Kamu adalah Profesor Analogi - ahli menjelaskan konsep dengan cara yang mudah dipahami siapa saja.

Awali dengan perkenalan:
"Halo VOX! Perkenalkan saya Profesor Analogi,
saya akan jelaskan itu dengan sederhana"

Gayamu:
- Selalu jelaskan dengan analogi dari kehidupan sehari-hari
- Gunakan perumpamaan yang relatable untuk orang indonesia
- Bertahap: mulai dari yang paling sederhana, lalu perlahan  lebih dalam
- Gunakan contoh konkret, bukan teori abstrak

Struktur penjelasan:
1. ANALOGI - samakan konsep dengan sesuatu yang familiar
2. PENJELASAN SEDERHANA - jelaskan dengan bahasa sehari-hari
3. CONTOH NYATA - berikan contoh konkret
4. LEBIH DALAM (opsional) - jika user ingin tahu lebih lanjut

Contoh gaya penjelasan:
User: "Apa itu API?"
Kamu:
"Bayangkan kamu pergi ke restoran. Kamu tidak masuk dapur langsung
untuk masak sendiri - kamu cukup pesan ke pelayan. Pelayan itulah
yang jadi perantara antara kamu dan dapur.

API persis seperti pelayan itu. Aplikasi A tidak perlu tahu cara
kerja dalam aplikasi B - cukup kirim 'pesanan' lewat API, dan
hasilnya dikirim balik.

Contoh nyata: saat kamu login dengan Google di aplikasi lain,
aplikasi itu minta data ke Google sendiri."

Selalu akhiri dengan tawaran: "Mau Profesor jelaskan lebih dalam lagi VOX ?"`

type ExplainUseCase struct {
	aiRepo   repository.AIRepository
	chatRepo repository.ChatRepository
}

func NewExplainUseCase(
	aiRepo repository.AIRepository,
	chatRepo repository.ChatRepository,
) *ExplainUseCase {
	return &ExplainUseCase{
		aiRepo:   aiRepo,
		chatRepo: chatRepo,
	}

}

func (uc *ExplainUseCase) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	conv, err := uc.getOrCreateExplainConversation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get or create conversation: %w", err)
	}

	userMsg := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleUser,
		Content:        req.Message,
		CreatedAt:      time.Now(),
	}
	if err := uc.chatRepo.SaveMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	history, err := uc.buildExplainHistory(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("build history: %w", err)
	}

	reply, err := uc.aiRepo.Generate(ctx, explainSystemPrompt, history)
	if err != nil {
		return nil, fmt.Errorf("generate reply: %w", err)
	}

	aiMsg := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleAssistant,
		Content:        reply,
		CreatedAt:      time.Now(),
	}
	if err := uc.chatRepo.SaveMessage(ctx, aiMsg); err != nil {
		return nil, fmt.Errorf("save ai message: %w", err)
	}

	return &domain.ChatResponse{
		ConversationID: conv.ID,
		Character:      domain.CharacterExplain,
		Reply:          reply,
	}, nil
}

func (uc *ExplainUseCase) getOrCreateExplainConversation(ctx context.Context, req *domain.ChatRequest) (*domain.Conversation, error) {
	if req.ConversationID != "" {
		conv, err := uc.chatRepo.GetConversation(ctx, req.ConversationID)
		if err != nil {
			return nil, err
		}
		if conv != nil {
			// Validasi karakter - conversation harus milik karakter yang sama
			if conv.Character != domain.CharacterExplain {
				return nil, fmt.Errorf(
					"conversation ini milik karakter %s, bukan explain", conv.Character,
				)
			}
			return conv, nil
		}
	}

	now := time.Now()
	conv := &domain.Conversation{
		ID:        uuid.New().String(),
		Character: domain.CharacterExplain,
		Title:     truncate(req.Message, 40),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.chatRepo.SaveConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (uc *ExplainUseCase) buildExplainHistory(ctx context.Context, conversationID string) ([]gemini.Content, error) {
	messages, err := uc.chatRepo.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Batasi 20 pesan terakhir
	if len(messages) > 20 {
		messages = messages[len(messages)-20:]
	}

	var history []gemini.Content
	for _, msg := range messages {
		role := "user"
		if msg.Role == domain.RoleAssistant {
			role = "model"
		}
		history = append(history, gemini.Content{
			Role:  role,
			Parts: []gemini.Part{{Text: msg.Content}},
		})
	}
	return history, nil
}
