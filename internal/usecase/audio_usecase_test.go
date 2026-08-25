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

type mockCharacterChat struct {
	gotReq *domain.ChatRequest
}

func (m *mockCharacterChat) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	m.gotReq = req
	return &domain.ChatResponse{
		ConversationID: "conv-1",
		Character:      req.Character,
		Reply:          "Iya nih bang, ada yang bisa dibantu?",
	}, nil
}

func TestTranscribeAudio(t *testing.T) {
	uc := usecase.NewAudioUseCase(&mockTranscriber{}, &mockSynthesizer{}, nil)
	res, err := uc.Transcribe(context.Background(), domain.AudioRequest{Data: []byte("test")})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Text != "Halo bang" {
		t.Errorf("expected text 'Halo bang', got '%s'", res.Text)
	}
}

func TestVoiceChat(t *testing.T) {
	betawi := &mockCharacterChat{}
	uc := usecase.NewAudioUseCase(&mockTranscriber{}, &mockSynthesizer{}, map[domain.Character]usecase.CharacterChatUseCase{
		domain.CharacterBetawi: betawi,
	})

	res, err := uc.VoiceChat(context.Background(), "betawi", "conv-1", domain.AudioRequest{Data: []byte("test")})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.UserText != "Halo bang" {
		t.Errorf("expected user_text 'Halo bang', got '%s'", res.UserText)
	}
	if res.AIText != "Iya nih bang, ada yang bisa dibantu?" {
		t.Errorf("expected ai_text dari Chat usecase, got '%s'", res.AIText)
	}
	if res.AudioBase64 == "" {
		t.Errorf("expected non-empty base64 audio")
	}
	if betawi.gotReq == nil || betawi.gotReq.Message != "Halo bang" {
		t.Errorf("expected Chat() dipanggil dengan hasil transkripsi STT sebagai message")
	}
}

func TestVoiceChat_UnknownCharacter(t *testing.T) {
	uc := usecase.NewAudioUseCase(&mockTranscriber{}, &mockSynthesizer{}, map[domain.Character]usecase.CharacterChatUseCase{})
	_, err := uc.VoiceChat(context.Background(), "ngasal", "", domain.AudioRequest{Data: []byte("test")})
	if err == nil {
		t.Fatalf("expected error untuk karakter yang tidak valid, nil")
	}
}
