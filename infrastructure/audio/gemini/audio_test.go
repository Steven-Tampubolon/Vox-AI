package gemini_test

import (
	"context"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/audio/gemini"
	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
)

func TestTranscriber(t *testing.T) {
	transcriber := gemini.NewTranscriber("dummy_api_key")
	res, err := transcriber.Transcribe(context.Background(), domain.AudioRequest{
		Data:     []byte("test"),
		Filename: "test.wav",
		MimeType: "audio/wav",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Text != "Suara sampel diterima" {
		t.Errorf("expected 'Suara sampel diterima', got '%s'", res.Text)
	}
	if res.Language != "id" {
		t.Errorf("expected 'id', got '%s'", res.Language)
	}
	if res.Duration != 1.5 {
		t.Errorf("expected 1.5, got %f", res.Duration)
	}
}

func TestSynthesizer(t *testing.T) {
	synthesizer := gemini.NewSynthesizer("dummy_api_key")
	res, err := synthesizer.Synthesize(context.Background(), domain.SynthesizeRequest{
		Text:    "Halo",
		VoiceID: "default",
		Speed:   1.0,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(res.AudioData) != "RIFF" {
		t.Errorf("expected audio data RIFF, got %v", res.AudioData)
	}
	if res.MimeType != "audio/wav" {
		t.Errorf("expected 'audio/wav', got '%s'", res.MimeType)
	}
}
