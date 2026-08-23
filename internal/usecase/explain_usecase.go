package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
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

// Chat - versi non-stream
func (uc *ExplainUseCase) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterExplain, uc.chatRepo)
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

// ChatStream - versi streaming untuk SSE
// onChunk dipanggil setiap ada potongan teks baru dari Gemini, supaya handler
// bisa langsung menulis ke http.ResponseWriter tanpa menunggu jawaban selesai.
func (uc *ExplainUseCase) ChatStream(ctx context.Context, req *domain.ChatRequest, onChunk func(text string) error) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterExplain, uc.chatRepo)
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
	streamErr := uc.aiRepo.GenerateStream(ctx, explainSystemPrompt, history, func(chunk string) error {
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
		Character:      domain.CharacterExplain,
		Reply:          reply,
	}, nil
}
