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

const betawiSystemPrompt = `Kamu adalah Abang Betawi - sosok yang humoris, hangat dan fasih berpantun.

Kepribadianmu:
- Bicara dengan logat betawi yang kental tapi tetap mudah dipahami
- Selalu semangat dan ceria, suka bercanda tapi tidak kasar
- Sapaan khas: "Halo Ncing!", "Aye", "Bro", "Ente", "Nyang", "Kagak"

Kemampuanmu:
1. BALAS PANTUN - jika user kirim pantun (2 atau 4 baris), wajib balas dengan pantun 4 baris yang indah dan relevan. Format: baris 1-2 sampiran, baris 3-4 isi.
2. BUAT PANTUN - jika user minta pantun tentang topik tertentu, buat pantun 4 baris yang sesuai.
3. NGOBROL BIASA - jika bukan pantun, jawab dengan hangat dan selipkan pantun pendek di akhir.

Aturan pantun:
- Rima akhir baris 1 & 3 harus sama bunyinya
- Rima akhir baris 2 & 4 harus sama bunyinya
- Baris 1-2 adalah sampiran (tidak berhubungan langsung dengan isi)
- Baris 3-4 adalah isi (pesan sebenarnya)

Contoh pantun yang baik:
Buah mangga buah rambutan,
Dibawa pulang dari pekan.
Hati senang bukan buatan,
Bertemu teman lama berdekatan.`

type BetawiUseCase struct {
	aiRepo   repository.AIRepository
	chatRepo repository.ChatRepository
}

func NewBetawiUseCase(
	aiRepo repository.AIRepository,
	chatRepo repository.ChatRepository,
) *BetawiUseCase {
	return &BetawiUseCase{
		aiRepo:   aiRepo,
		chatRepo: chatRepo,
	}
}

func (uc *BetawiUseCase) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	// 1. Buat atau ambil conversation
	conv, err := uc.getOrCreateConversation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get or create conversation: %w", err)
	}

	// 2. Simpan pesan user ke database
	userMsg := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleUser,
		Content:        req.Message,
		CreatedAt:      time.Now(),
	}
	if err := uc.chatRepo.SaveMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// 3. Bangun history percakapan untuk konteks Gemini
	history, err := uc.buildHistory(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("build history: %w", err)
	}

	// 4. Kirim ke Gemini
	reply, err := uc.aiRepo.Generate(ctx, betawiSystemPrompt, history)
	if err != nil {
		return nil, fmt.Errorf("generate reply: %w", err)
	}

	// 5. Simpan jawaban AI ke database
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
		Character:      domain.CharacterBetawi,
		Reply:          reply,
	}, nil
}

// getOrCreateConversation buat sesi baru jika ConversationID kosong
func (uc *BetawiUseCase) getOrCreateConversation(ctx context.Context, req *domain.ChatRequest) (*domain.Conversation, error) {
	if req.ConversationID != "" {
		conv, err := uc.chatRepo.GetConversation(ctx, req.ConversationID)
		if err != nil {
			return nil, err
		}
		if conv != nil {
			// Validasi karakter - conversation harus milik karakter yang sama
			if conv.Character != domain.CharacterBetawi {
				return nil, fmt.Errorf(
					"conversation ini milik karakter %s, bukan betawi", conv.Character,
				)
			}
			return conv, nil
		}
	}

	// Buat conversation baru
	now := time.Now()
	conv := &domain.Conversation{
		ID:        uuid.New().String(),
		Character: domain.CharacterBetawi,
		Title:     truncate(req.Message, 40),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.chatRepo.SaveConversation(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// buildHistory ubah messages dari DB menjadi format Gemini
func (uc *BetawiUseCase) buildHistory(ctx context.Context, conversationID string) ([]gemini.Content, error) {
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

// truncate potong string agar tidak terlalu panjang jadi judul
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
