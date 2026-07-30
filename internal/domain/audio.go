package domain

type AudioRequest struct {
	Data     []byte `json:"-"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
}

type TranscribeResult struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration_seconds"`
}

type SynthesizeRequest struct {
	Text    string  `json:"text"`
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
}

type SynthesizeResult struct {
	AudioData []byte `json:"-"`
	MimeType  string `json:"mime_type"`
}

type VoiceChatResult struct {
	UserText       string `json:"user_text"`
	AIText         string `json:"ai_text"`
	AudioBase64    string `json:"audio_base64"`
	MimeType       string `json:"mime_type"`
	ConversationID string `json:"conversation_id"`
}
