package repository

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/gemini"
)

// AIRepository adalah kontrak untuk komunikasi dengan AI model
// Saat ini implementasinya Gemini, bisa diganti Ollama
// tanpa ubah usecase sama sekali
type AIRepository interface {
	Generate(ctx context.Context, systemPrompt string, history []gemini.Content) (string, error)
	GenerateStream(ctx context.Context, systemPrompt string, history []gemini.Content, onChunk func(text string) error) error
	Embed(ctx context.Context, text string) ([]float64, error)
}

// GeminiAIRepository adalah implementasi AIRepository menggunakan Gemini
type GeminiAIRepository struct {
	client *gemini.Client
}

func NewGeminiAIRepository(client *gemini.Client) AIRepository {
	return &GeminiAIRepository{client: client}
}

func (r *GeminiAIRepository) Generate(ctx context.Context, systemPrompt string, history []gemini.Content) (string, error) {
	return r.client.Generate(ctx, systemPrompt, history)
}

// GenerateStream meneruskan panggilan streaming ke gemini.Client.
// onChunk dipanggil setiap ada potongan teks baru dari model.
func (r *GeminiAIRepository) GenerateStream(ctx context.Context, systemPrompt string, history []gemini.Content, onChunk func(text string) error) error {
	return r.client.GenerateStream(ctx, systemPrompt, history, onChunk)
}

func (r *GeminiAIRepository) Embed(ctx context.Context, text string) ([]float64, error) {
	return r.client.Embed(ctx, text)
}
