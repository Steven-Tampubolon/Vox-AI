# Design Spec: Vox-AI Voice / Audio API & Frontend Handover

- **Date**: 2026-07-31
- **Status**: Approved
- **Scope**: Backend Audio Upgrade (STT, TTS, Voice Chat) & OpenAPI/Frontend Handover Documentation

---

## 1. Overview
Vox-AI backend is being expanded to support voice/audio capabilities alongside existing multi-character text chats. 
This feature introduces Speech-to-Text (STT), Text-to-Speech (TTS), and an end-to-end Voice Chat pipeline, while providing complete OpenAPI 3.0 specs and Frontend integration documentation for handover.

---

## 2. Architecture & Design Principles

Following Onion / Clean Architecture:
- **Domain Layer (`internal/domain`)**: Core data structures for audio requests, transcriptions, synthesis, and voice chat results.
- **Repository Interfaces (`internal/repository`)**: Abstract `AudioTranscriber` and `AudioSynthesizer` ports.
- **Infrastructure Layer (`infrastructure/audio/gemini`)**: Concrete Gemini API adapter implementing STT and TTS interfaces.
- **Business Logic Layer (`internal/usecase`)**: `AudioUsecase` coordinating audio transcription, character prompt chat generation, audio synthesis, and conversation history persistence.
- **Delivery Layer (`internal/delivery/http`)**: Gin HTTP handlers and routes for STT, TTS, Voice Chat, and OpenAPI/Handover docs.

---

## 3. Detailed Component Specifications

### 3.1 Domain Layer (`internal/domain/audio.go`)
```go
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
```

### 3.2 Repository Interfaces (`internal/repository/audio_repository.go`)
```go
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
```

### 3.3 Infrastructure Layer (`infrastructure/audio/gemini`)
- **`transcriber.go`**: Encapsulates audio data handling and calls Gemini Multimodal API inline or file upload API for transcription.
- **`synthesizer.go`**: Calls Gemini audio output capability or fallback TTS service to output WAV/MP3 bytes.

### 3.4 Usecase Layer (`internal/usecase/audio_usecase.go`)
```go
type AudioUsecase struct {
    transcriber transcriber.AudioTranscriber
    synthesizer synthesizer.AudioSynthesizer
    aiRepo      repository.AIRepository
    chatRepo    repository.ChatRepository
}
```
Key Workflow for `VoiceChat`:
1. Receive audio input file (`domain.AudioRequest`).
2. `TranscribeAudio` -> get `UserText`.
3. Load existing history for `ConversationID` (or start new conversation).
4. Run AI Usecase prompt for specified character slug (`betawi`, `rag`, `git`, `explain`).
5. `SynthesizeText` for `AIText` -> get `AudioData`.
6. Save message pair to SQLite (`ChatRepository`).
7. Return `VoiceChatResult` with `AudioBase64` & metadata.

### 3.5 Delivery Layer (`internal/delivery/http/handler/audio_handler.go`)
Endpoints:
- `POST /api/v1/audio/transcribe`: Expects `file` in `multipart/form-data`.
- `POST /api/v1/audio/synthesize`: Expects JSON body `{"text": "...", "voice_id": "..."}`.
- `POST /api/v1/voice/chat`: Expects `multipart/form-data` with fields `file` (audio), `character` (string), `conversation_id` (optional string).

---

## 4. Documentation & Handover Artifacts

1. **OpenAPI 3.0 Specification**: `docs/api/openapi.yaml`
   - Complete endpoint definitions, schemas, request/response models for all Vox-AI endpoints (`/chat`, `/conversations`, `/audio`, `/voice`).
2. **Frontend Handover Integration Guide**: `docs/handover/frontend-voice-guide.md`
   - Detailed instructions for Frontend engineers:
     - MediaRecorder browser recording setup (WebM/WAV).
     - Payload schemas for voice chat and audio utilities.
     - Sample code snippets for playing audio responses using HTML5 Audio / Web Audio API.

---

## 5. Verification & Testing Strategy
- Unit tests for `internal/usecase/audio_usecase_test.go` using mocks for `AudioTranscriber` & `AudioSynthesizer`.
- Integration tests for `internal/delivery/http/handler/audio_handler_test.go`.
- Manual verification of audio endpoints using curl / sample audio file.
