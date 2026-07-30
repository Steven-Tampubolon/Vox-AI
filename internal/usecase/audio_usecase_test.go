package usecase_test

import (
	"context"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
)

type mockTranscriber struct{}

func (m *mockTranscriber) Transcribe(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error) {
	return &domain.TranscribeResult{Text: "Halo bang"}, nil
}

type mockSynthesizer struct{}

func (m *mockSynthesizer) Synthesize(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error) {
	return &domain.SynthesizeResult{AudioData: []byte("audio"), MimeType: "audio/wav"}, nil
}

func TestTranscribeAudio(t *testing.T) {
	uc := usecase.NewAudioUsecase(&mockTranscriber{}, &mockSynthesizer{}, nil, nil)
	res, err := uc.TranscribeAudio(context.Background(), domain.AudioRequest{Data: []byte("test")})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Text != "Halo bang" {
		t.Errorf("expected text 'Halo bang', got '%s'", res.Text)
	}
}

func TestVoiceChat(t *testing.T) {
	uc := usecase.NewAudioUsecase(&mockTranscriber{}, &mockSynthesizer{}, nil, nil)
	res, err := uc.VoiceChat(context.Background(), "betawi", "conv-1", domain.AudioRequest{Data: []byte("test")})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.UserText != "Halo bang" {
		t.Errorf("expected user_text 'Halo bang', got '%s'", res.UserText)
	}
	if res.AudioBase64 == "" {
		t.Errorf("expected non-empty base64 audio")
	}
}
