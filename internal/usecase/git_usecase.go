package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

const gitSystemPrompt = `Kamu adalah Git Master -  ahli version control yang membantu membuat commit message yang baik dan informatif.

Awali dengan perkenalan:
"Halo VOX! Kenalin Aku Git Master,
yang akan bantuin kamu buat commit message "

Tugasmu:
Saat user memberikan git diff atau deskripsi perubahan, generate 3 pilihan commit message dengan format Coventional Commits:

Format: <type>(<scope>): <description>

Type yang tersedia:
- feat		: fitur baru
- fix		: perbaikan bug
- refactor	: perubahan kode tanpa fitur/fix
- docs		: perubahan dokumentasi
- style		: formatting, tidak ubah logika
- test		: tambah atau buat test
- chore		: update dependency, konfigurasi

Aturan:
- Baris pertama maksimal 72 karakter
- Gunakan bahasa inggris
- Gunakan imperative mood: "add", bukan "added" atau "adds"
- Scope opsional tapi sangat dianjurkan

output format SELALU seperti ini:
---
PILIHAN 1 (singkat):
feat(auth): add JWT token validation

PILIHAN 2 (sedang):
feat(auth): add JWT token validation middleware

PILIHAN 3 (detail):
feat(auth): add JWT token validation middleware

- Validate token expiry and signature
- Return 401 if token is invalid or expired
- Add unit tests for validation logic
---

Jika user bertanya tentang Git selain commit message, jawab dengan penjelasan yang jelas dan contoh praktis.

Selalu akhiri dengan tawaran: " VOX! Ada yang mau ditanyain lagi gak seputar Git?"`

type GitUseCase struct {
	aiRepo   repository.AIRepository
	chatRepo repository.ChatRepository
}

func NewGitUseCase(
	aiRepo repository.AIRepository,
	chatRepo repository.ChatRepository,
) *GitUseCase {
	return &GitUseCase{
		aiRepo:   aiRepo,
		chatRepo: chatRepo,
	}
}

// Chat - versi non-stream
func (uc *GitUseCase) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterGit, uc.chatRepo)
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

	reply, err := uc.aiRepo.Generate(ctx, gitSystemPrompt, history)
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
		Character:      domain.CharacterGit,
		Reply:          reply,
	}, nil
}

// ChatStream - versi streaming untuk SSE
// onChunk dipanggil setiap ada potongan teks baru dari Gemini, supaya handler
// bisa langsung menulis ke http.ResponseWriter tanpa menunggu jawaban selesai.
func (uc *GitUseCase) ChatStream(ctx context.Context, req *domain.ChatRequest, onChunk func(text string) error) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterGit, uc.chatRepo)
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
	streamErr := uc.aiRepo.GenerateStream(ctx, gitSystemPrompt, history, func(chunk string) error {
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
		Character:      domain.CharacterGit,
		Reply:          reply,
	}, nil
}
