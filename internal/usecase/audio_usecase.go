package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

// CharacterUseCase adalah kontrak minimal yang sudah dipenuhi usecase dari tiap karakter
type CharacterChatUsecase interface {
	Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error)
}

type AudioUsecase struct {
	transcriber repository.AudioTranscriber
	synthesizer repository.AudioSynthesizer
	character   map[domain.Character]CharacterChatUsecase
}

func NewAudioUsecase(
	transcriber repository.AudioTranscriber,
	synthesizer repository.AudioSynthesizer,
	character map[domain.Character]CharacterChatUsecase,
) *AudioUsecase {
	return &AudioUsecase{
		transcriber: transcriber,
		synthesizer: synthesizer,
		character:   character,
	}
}

// TranscribeAudio - dipakai handler POST /api/v1/audio/transcribe (STT murni)
func (u *AudioUsecase) Transcribe(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error) {
	return u.transcriber.Transcribe(ctx, req)
}

// SynthesizeText - dipakai handler POST /api/v1/audio/synthesize (TTS murni)
func (u *AudioUsecase) SynthesizeText(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error) {
	return u.synthesizer.Synthesize(ctx, req)
}

// VoiceChat - dipakai handler POST /api/v1/voice/chat.
// Pipeline: STT -> AI per-karakter (history + persistence) -> TTS
func (u *AudioUsecase) VoiceChat(ctx context.Context, character string, convID string, req domain.AudioRequest) (*domain.VoiceChatResult, error) {
	char := domain.Character(character)
	if !char.IsValid() {
		return nil, fmt.Errorf("karakter '%s' tidak dikenal, gunakan salah satu dari: betawi, rag, git, explain", character)
	}

	chatUc, ok := u.character[char]
	if !ok {
		return nil, fmt.Errorf("karakter '%s' belum didukung untuk voice chat", character)
	}

	// 1. STT - audio user jadi teks
	stt, err := u.transcriber.Transcribe(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("transcribe audio: '%w'", err)
	}
	if stt.Text == "" {
		return nil, fmt.Errorf("tidak ada ucapan yang terdekteksi pada rekaman, coba rekam ulang")
	}

	// 2. AI - reuse usecase karakter yang sudah ada: otomatis
	// getOrCreateConversation, simpan pesan user, build history, generate
	// balasan lewat Gemini, dan simpan balasan AI ke SQLite
	chatRes, err := chatUc.Chat(ctx, &domain.ChatRequest{
		ConversationID: convID,
		Character:      char,
		Message:        stt.Text,
	})
	if err != nil {
		return nil, fmt.Errorf("generate ai reply: %w", err)
	}

	// 3. TTS - balasan AI jadi audio
	tts, err := u.synthesizer.Synthesize(ctx, domain.SynthesizeRequest{Text: chatRes.Reply})
	if err != nil {
		return nil, fmt.Errorf("synthesize audio: %w", err)
	}

	return &domain.VoiceChatResult{
		UserText:       stt.Text,
		AIText:         chatRes.Reply,
		AudioBase64:    base64.RawStdEncoding.EncodeToString(tts.AudioData),
		MimeType:       tts.MimeType,
		ConversationID: chatRes.ConversationID,
	}, nil
}
