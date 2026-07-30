package repository

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
)

type AudioTranscriber interface {
	Transcribe(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error)
}

type AudioSynthesizer interface {
	Synthesize(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error)
}
