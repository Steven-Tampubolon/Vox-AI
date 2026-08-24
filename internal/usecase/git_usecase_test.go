package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
)

// TestGitChatStream_ReturnsCorrectCharacter adalah regression test untuk
// bug fix step 1: sebelumnya ChatStream salah mengembalikan
// domain.CharacterExplain, padahal ini usecase Git.
func TestGitChatStream_ReturnsCorrectCharacter(t *testing.T) {
	chatRepo := newMockChatRepo()
	aiRepo := &mockAIRepo{
		streamFn: func(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
			return onChunk("feat(auth): add JWT validation")
		},
	}
	uc := usecase.NewGitUseCase(aiRepo, chatRepo)

	res, err := uc.ChatStream(context.Background(), &domain.ChatRequest{Message: "buatkan commit message"}, func(string) error { return nil })
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Character != domain.CharacterGit {
		t.Errorf("expected character git, got %s", res.Character)
	}
}

// TestGitChatStream_StreamErrorIsWrapped adalah regression test untuk bug
// fix step 1: sebelumnya kode mem-wrap `err` (nil) alih-alih `streamErr`
// saat GenerateStream gagal, sehingga error asli hilang.
func TestGitChatStream_StreamErrorIsWrapped(t *testing.T) {
	wantErr := errors.New("gemini API error: status 500")
	chatRepo := newMockChatRepo()
	aiRepo := &mockAIRepo{
		streamFn: func(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
			_ = onChunk("sebagian jawaban") // reply parsial sebelum gagal
			return wantErr
		},
	}
	uc := usecase.NewGitUseCase(aiRepo, chatRepo)

	_, err := uc.ChatStream(context.Background(), &domain.ChatRequest{Message: "halo"}, func(string) error { return nil })
	if err == nil {
		t.Fatalf("expected error to propagate from failed GenerateStream")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error to be wantErr (via errors.Is), got: %v", err)
	}
}
