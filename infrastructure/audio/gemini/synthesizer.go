package gemini

import (
	"context"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

type Synthesizer struct {
	apiKey string
}

func NewSynthesizer(apiKey string) repository.AudioSynthesizer {
	return &Synthesizer{apiKey: apiKey}
}

func (s *Synthesizer) Synthesize(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error) {
	dummyAudio := []byte{0x52, 0x49, 0x46, 0x46}
	return &domain.SynthesizeResult{
		AudioData: dummyAudio,
		MimeType:  "audio/wav",
	}, nil
}
