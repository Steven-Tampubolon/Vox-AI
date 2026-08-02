package gemini_test

import (
	"context"
	"os"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/audio/gemini"
	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load("../../../.env")
}

// getAPIKey mengambil API key dari environment variable.
// Jika tidak diset, test akan di-SKIP agar tidak gagal karena dummy key.
func getAPIKey(t *testing.T) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("[SKIP] GEMINI_API_KEY tidak ditemukan di environment. Melewati integration test.")
	}
	return apiKey
}

func TestTranscriber(t *testing.T) {
	apiKey := getAPIKey(t)
	transcriber := gemini.NewTranscriber(apiKey)

	res, err := transcriber.Transcribe(context.Background(), domain.AudioRequest{
		Data:     []byte("test"),
		Filename: "test.wav",
		MimeType: "audio/wav",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Text == "" {
		t.Errorf("expected non-empty text response")
	}
}

func TestSynthesizer(t *testing.T) {
	apiKey := getAPIKey(t)
	synthesizer := gemini.NewSynthesizer(apiKey)

	res, err := synthesizer.Synthesize(context.Background(), domain.SynthesizeRequest{
		Text:    "Halo",
		VoiceID: "Pulcherrima",
		Speed:   1.0,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(res.AudioData) == 0 {
		t.Errorf("expected audio data, got empty byte slice")
	}
}
