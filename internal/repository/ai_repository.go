package repository

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/gemini"
)

// ChatMessage merepresentasikan satu pesan dalam history percakapan,
// independen dari detail implementasi provider AI (Gemini, Ollama, dst).
// Ini menggantikan penggunaan gemini.Content secara langsung di interface
// AIRepository, supaya layer internal tidak bergantung pada tipe
// infrastructure/gemini.
type ChatMessage struct {
	Role    string // "user" atau "model"
	Content string
}

// AIRepository adalah kontrak untuk komunikasi dengan AI model
// Saat ini implementasinya Gemini, bisa diganti Ollama
// tanpa ubah usecase sama sekali
type AIRepository interface {
	Generate(ctx context.Context, systemPrompt string, history []ChatMessage) (string, error)
	GenerateStream(ctx context.Context, systemPrompt string, history []ChatMessage, onChunk func(text string) error) error
	Embed(ctx context.Context, text string) ([]float64, error)
}

// GeminiAIRepository adalah implementasi AIRepository menggunakan Gemini
type GeminiAIRepository struct {
	client *gemini.Client
}

func NewGeminiAIRepository(client *gemini.Client) AIRepository {
	return &GeminiAIRepository{client: client}
}

func (r *GeminiAIRepository) Generate(ctx context.Context, systemPrompt string, history []ChatMessage) (string, error) {
	return r.client.Generate(ctx, systemPrompt, toGeminiContent(history))
}

// GenerateStream meneruskan panggilan streaming ke gemini.Client.
// onChunk dipanggil setiap ada potongan teks baru dari model.
func (r *GeminiAIRepository) GenerateStream(ctx context.Context, systemPrompt string, history []ChatMessage, onChunk func(text string) error) error {
	return r.client.GenerateStream(ctx, systemPrompt, toGeminiContent(history), onChunk)
}

func (r *GeminiAIRepository) Embed(ctx context.Context, text string) ([]float64, error) {
	return r.client.Embed(ctx, text)
}

// toGeminiContent menerjemahkan ChatMessage (tipe internal) ke gemini.Content
// (tipe infrastructure) - satu-satunya tempat konversi ini terjadi.
func toGeminiContent(history []ChatMessage) []gemini.Content {
	contents := make([]gemini.Content, len(history))
	for i, msg := range history {
		contents[i] = gemini.Content{
			Role:  msg.Role,
			Parts: []gemini.Part{{Text: msg.Content}},
		}
	}
	return contents
}
