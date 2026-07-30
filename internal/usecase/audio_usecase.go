package usecase

import (
	"context"
	"encoding/base64"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

type AudioUsecase struct {
	transcriber repository.AudioTranscriber
	synthesizer repository.AudioSynthesizer
	aiRepo      repository.AIRepository
	chatRepo    repository.ChatRepository
}

func NewAudioUsecase(
	transcriber repository.AudioTranscriber,
	synthesizer repository.AudioSynthesizer,
	aiRepo repository.AIRepository,
	chatRepo repository.ChatRepository,
) *AudioUsecase {
	return &AudioUsecase{
		transcriber: transcriber,
		synthesizer: synthesizer,
		aiRepo:      aiRepo,
		chatRepo:    chatRepo,
	}
}

func (u *AudioUsecase) TranscribeAudio(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error) {
	return u.transcriber.Transcribe(ctx, req)
}

func (u *AudioUsecase) SynthesizeText(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error) {
	return u.synthesizer.Synthesize(ctx, req)
}

func (u *AudioUsecase) VoiceChat(ctx context.Context, character string, convID string, req domain.AudioRequest) (*domain.VoiceChatResult, error) {
	stt, err := u.transcriber.Transcribe(ctx, req)
	if err != nil {
		return nil, err
	}

	aiText := "Iya nih bang, ada yang bisa dibantu?"
	tts, err := u.synthesizer.Synthesize(ctx, domain.SynthesizeRequest{Text: aiText})
	if err != nil {
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(tts.AudioData)
	return &domain.VoiceChatResult{
		UserText:       stt.Text,
		AIText:         aiText,
		AudioBase64:    b64,
		MimeType:       tts.MimeType,
		ConversationID: convID,
	}, nil
}
