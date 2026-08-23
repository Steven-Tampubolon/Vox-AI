package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
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

// Chat - versi non-stream
func (uc *BetawiUseCase) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	// 1. Buat atau ambil conversation
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterBetawi, uc.chatRepo)
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
	history, err := buildHistory(ctx, conv.ID, uc.chatRepo)
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

// ChatStream - versi streaming untuk SSE
// onChunk dipanggil setiap ada potongan teks baru dari Gemini, supaya handler
// bisa langsung menulis ke http.ResponseWriter tanpa menunggu jawaban selesai.
func (uc *BetawiUseCase) ChatStream(ctx context.Context, req *domain.ChatRequest, onChunk func(text string) error) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterBetawi, uc.chatRepo)
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

	history, err := buildHistory(ctx, conv.ID, uc.chatRepo)
	if err != nil {
		return nil, fmt.Errorf("build history: %w", err)
	}

	var fullReply strings.Builder
	streamErr := uc.aiRepo.GenerateStream(ctx, betawiSystemPrompt, history, func(chunk string) error {
		fullReply.WriteString(chunk)
		return onChunk(chunk)
	})

	reply := fullReply.String()

	// Simpan reply meskipun stream berhenti di tengah jalan (mis. user tekan "STOP" / koneksi terputus)
	// supaya history di DB tetap konsisten
	if reply != "" {
		aiMsg := &domain.Message{
			ConversationID: conv.ID,
			Role:           domain.RoleAssistant,
			Content:        reply,
			CreatedAt:      time.Now(),
		}
		if saveErr := uc.chatRepo.SaveMessage(ctx, aiMsg); saveErr != nil {
			return nil, fmt.Errorf("save ai message: %w", saveErr)
		}
	}

	if streamErr != nil {
		return nil, fmt.Errorf("generate stream reply: %w", streamErr)
	}

	if reply == "" {
		return nil, fmt.Errorf("gemini tidak mengembalikan jawaban")
	}

	return &domain.ChatResponse{
		ConversationID: conv.ID,
		Character:      domain.CharacterBetawi,
		Reply:          reply,
	}, nil
}

// truncate potong string agar tidak terlalu panjang jadi judul
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
