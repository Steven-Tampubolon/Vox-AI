package handler_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/handler"
	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
)

// stubTranscriber adalah fake AudioTranscriber untuk test handler.
type stubTranscriber struct {
	result *domain.TranscribeResult
	err    error
}

func (s *stubTranscriber) Transcribe(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error) {
	return s.result, s.err
}

// stubSynthesizer adalah fake AudioSynthesizer untuk test handler.
type stubSynthesizer struct {
	result *domain.SynthesizeResult
	err    error
}

func (s *stubSynthesizer) Synthesize(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error) {
	return s.result, s.err
}

// buildMultipartAudioRequest membuat *http.Request multipart/form-data berisi
// field "file" sebesar payloadSize byte, untuk simulasi upload audio.
func buildMultipartAudioRequest(t *testing.T, url string, payloadSize int, extraFields map[string]string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for k, v := range extraFields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("gagal tulis field %s: %v", k, err)
		}
	}

	fw, err := mw.CreateFormFile("file", "test.wav")
	if err != nil {
		t.Fatalf("gagal buat form file: %v", err)
	}
	if _, err := fw.Write(bytes.Repeat([]byte("A"), payloadSize)); err != nil {
		t.Fatalf("gagal tulis payload: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("gagal close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// TestTranscribe_RejectsPayloadOverLimit adalah regression test untuk
// Temuan 2 (security review): sebelum fix ini, endpoint /audio/transcribe
// tidak punya batas ukuran sama sekali dan langsung io.ReadAll payload
// berapa pun besarnya - berisiko OOM/DoS. Sekarang harus ditolak 413
// SEBELUM data besar itu selesai dibaca ke memori.
func TestTranscribe_RejectsPayloadOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioUC := usecase.NewAudioUseCase(&stubTranscriber{}, &stubSynthesizer{}, nil)
	h := handler.NewAudioHandler(audioUC)

	router := gin.New()
	router.POST("/audio/transcribe", h.Transcribe)

	// 11MB - di atas batas 10MB (maxUploadSize)
	req := buildMultipartAudioRequest(t, "/audio/transcribe", 11<<20, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestTranscribe_AcceptsPayloadUnderLimit memastikan fix di atas TIDAK
// menolak file audio berukuran wajar (di bawah 10MB) - regression check
// supaya batasnya tidak kelewat ketat.
func TestTranscribe_AcceptsPayloadUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioUC := usecase.NewAudioUseCase(&stubTranscriber{
		result: &domain.TranscribeResult{Text: "halo dunia"},
	}, &stubSynthesizer{}, nil)
	h := handler.NewAudioHandler(audioUC)

	router := gin.New()
	router.POST("/audio/transcribe", h.Transcribe)

	// 1MB - jelas di bawah batas
	req := buildMultipartAudioRequest(t, "/audio/transcribe", 1<<20, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestVoiceChat_RejectsPayloadOverLimit adalah regression test yang sama
// untuk endpoint /voice/chat (juga disebut eksplisit di Temuan 2).
func TestVoiceChat_RejectsPayloadOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioUC := usecase.NewAudioUseCase(&stubTranscriber{}, &stubSynthesizer{}, nil)
	h := handler.NewAudioHandler(audioUC)

	router := gin.New()
	router.POST("/voice/chat", h.VoiceChat)

	req := buildMultipartAudioRequest(t, "/voice/chat", 11<<20, map[string]string{
		"character": "betawi",
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d, body: %s", w.Code, w.Body.String())
	}
}
