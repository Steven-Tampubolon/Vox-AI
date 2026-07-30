package gemini

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

type Transcriber struct {
	apiKey string
}

func NewTranscriber(apiKey string) repository.AudioTranscriber {
	return &Transcriber{apiKey: apiKey}
}

func (t *Transcriber) Transcribe(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error) {
	return &domain.TranscribeResult{
		Text:     "Suara sampel diterima",
		Language: "id",
		Duration: 1.5,
	}, nil
}
