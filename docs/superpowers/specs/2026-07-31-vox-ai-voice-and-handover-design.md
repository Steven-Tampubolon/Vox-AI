# Design Spec: Vox-AI Voice / Audio API & Frontend Handover

- **Date**: 2026-07-31
- **Updated**: 2026-08-03
- **Status**: Implemented (v1.2.0)
- **Scope**: Backend Audio Upgrade (STT, TTS, Voice Chat) & OpenAPI/Frontend Handover Documentation

---

## 1. Overview
Vox-AI backend diperluas untuk mendukung kemampuan voice/audio di samping text chat
multi-karakter yang sudah ada. Fitur ini menghadirkan Speech-to-Text (STT),
Text-to-Speech (TTS), dan pipeline Voice Chat end-to-end, beserta OpenAPI 3.0 spec
dan dokumentasi integrasi Frontend untuk handover.

---

## 2. Architecture & Design Principles

Mengikuti Onion / Clean Architecture:
- **Domain Layer (`internal/domain`)**: Struktur data inti untuk audio request,
  transkripsi, sintesis, voice chat result, dan audio profile per karakter.
- **Repository Interfaces (`internal/repository`)**: Abstract port `AudioTranscriber`
  dan `AudioSynthesizer`.
- **Infrastructure Layer (`infrastructure/audio/gemini`)**: Adapter Gemini API konkret
  yang mengimplementasikan STT (generateContent multimodal) dan TTS
  (generateContent dengan speech_config, output PCM → WAV).
- **Business Logic Layer (`internal/usecase`)**: `AudioUsecase` mengkoordinasikan
  transkripsi audio, delegasi ke character usecase (yang handle history, system prompt,
  dan persistence), sintesis audio, serta `voice_profiles.go` sebagai sumber
  kebenaran konfigurasi suara per karakter.
- **Delivery Layer (`internal/delivery/http`)**: Gin HTTP handler dan route untuk
  STT, TTS, Voice Chat.

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
    Text         string  `json:"text"`
    VoiceID      string  `json:"voice_id"`
    Speed        float64 `json:"speed"`
    // AudioProfile membawa instruksi karakter (audio profile + director's note
    // + scene) dari usecase ke synthesizer. Tag json:"-" supaya tidak
    // terekspos ke FE via request/response.
    AudioProfile string  `json:"-"`
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

**`transcriber.go`** — STT via Gemini generateContent multimodal:
- Kirim audio sebagai base64 `inlineData` ke endpoint `generateContent`
  model `gemini-2.5-flash-lite` (model yang sama dengan chat teks, sudah multimodal)
- Prompt instruksi verbatim supaya output hanya teks transkripsi mentah
- `normalizeAudioMime` membersihkan parameter codec dari Content-Type browser
  (mis. `audio/webm;codecs=opus` → `audio/webm`) dan fallback berdasarkan ekstensi
- ⚠️ Format `audio/webm` (default `MediaRecorder` Chrome) tidak ada dalam daftar
  resmi format yang didukung Gemini STT. Format yang direkomendasikan:
  `audio/ogg;codecs=opus`, `audio/wav`, `audio/mp3`, `audio/flac`

**`synthesizer.go`** — TTS via Gemini generateContent dengan audio output:
- Endpoint: `generateContent` model `gemini-2.5-flash-preview-tts`
- Field `speech_config` menggunakan **snake_case** (`voice_config`,
  `prebuilt_voice_config`, `voice_name`) — camelCase diabaikan Gemini dan
  menyebabkan model fallback ke text generation (error 400)
- `responseModalities: ["audio"]` dan `temperature: 2` sesuai rekomendasi
  Google AI Studio untuk output TTS yang ekspresif
- Gemini mengembalikan raw PCM 16-bit mono (`audio/L16;codec=pcm;rate=24000`),
  bukan file WAV utuh — BE membungkus PCM dengan header WAV 44-byte via
  `pcmToWav()` supaya browser bisa langsung memutar via `new Audio(...)`

### 3.4 Usecase Layer

#### `audio_usecase.go`

**Desain kunci**: `AudioUsecase` tidak mengakses `AIRepository` atau `ChatRepository`
secara langsung. Sebaliknya, ia bergantung pada interface `CharacterChatUsecase` yang
sudah dipenuhi oleh `BetawiUseCase`, `RAGUseCase`, `GitUseCase`, dan `ExplainUseCase`.
Ini memastikan VoiceChat reuse system prompt, history building, dan persistence
yang sudah ada di masing-masing usecase karakter — tanpa duplikasi logic.

```go
type CharacterChatUsecase interface {
    Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error)
}

type AudioUsecase struct {
    transcriber repository.AudioTranscriber
    synthesizer repository.AudioSynthesizer
    characters  map[domain.Character]CharacterChatUsecase
}
```

Key Workflow untuk `VoiceChat`:
1. Validasi `character` slug — return error kalau tidak dikenal
2. Lookup `CharacterChatUsecase` dari map berdasarkan karakter
3. Lookup `VoiceProfile` (VoiceID + AudioProfile) dari `characterVoiceProfiles`
4. **STT** — `transcriber.Transcribe()` → `UserText`
5. **AI** — `chatUC.Chat()` dengan `UserText` sebagai message → `AIText`
   (history load, system prompt karakter, persistence ke SQLite semua
   dihandle oleh usecase karakter — bukan AudioUsecase)
6. **TTS** — `synthesizer.Synthesize()` dengan `AIText` + `VoiceProfile` → `AudioData`
7. Return `VoiceChatResult` dengan `AudioBase64` & metadata

#### `voice_profiles.go` ← file baru

Sumber kebenaran tunggal untuk konfigurasi suara per karakter. Berisi map
`characterVoiceProfiles` dengan tipe `map[domain.Character]VoiceProfile`.

```go
type VoiceProfile struct {
    VoiceID      string // nama suara Gemini TTS (mis. Puck, Charon, Fenrir)
    AudioProfile string // full audio profile + director's note + scene instruction
}
```

Mapping karakter → suara:
| Karakter | VoiceID | Persona Suara |
|---|---|---|
| `betawi` | Puck | Hangat, kasual, energik |
| `rag` | Charon | Profesional, informatif, presisi |
| `git` | Fenrir | Antusias, teknikal, encouraging |
| `explain` | Sadaltager | Bijaksana, sabar, edukatif |

Untuk mengganti suara atau instruksi karakter, cukup edit file ini —
tidak perlu menyentuh usecase atau handler manapun.

### 3.5 Delivery Layer (`internal/delivery/http/handler/audio_handler.go`)

Endpoints (tidak berubah dari spec awal):
- `POST /api/v1/audio/transcribe` — `multipart/form-data` field `file`
- `POST /api/v1/audio/synthesize` — JSON `{"text": "...", "voice_id": "..."}`
  → response binary WAV (bukan JSON)
- `POST /api/v1/voice/chat` — `multipart/form-data` field `file`, `character`,
  `conversation_id` (opsional)

---

## 4. Documentation & Handover Artifacts

1. **OpenAPI 3.0 Specification** (v1.2.0): `docs/api/openapi.yaml`
   - Definisi endpoint lengkap dengan schema request/response eksplisit
   - `components/schemas`: `ErrorResponse`, `ChatRequest`, `ChatResponse`,
     `TranscribeResponse`, `VoiceChatResponse`
   - Response type `/audio/synthesize` eksplisit sebagai `audio/wav` binary
   - Error response 400/500 di semua endpoint
   - Warning format `audio/webm` di endpoint STT dan voice chat

2. **Frontend Handover Integration Guide**: `docs/handover/frontend-voice-guide.md`
   - Setup `MediaRecorder` dengan format `audio/ogg;codecs=opus` (rekomendasi)
     beserta fallback detection
   - Warning: `audio_base64` bisa 5–10MB, jangan di-log ke console
   - Contoh lengkap dengan state management `conversation_id` antar giliran
   - Tabel error response yang mungkin muncul beserta penyebabnya
   - Catatan performa: latency 2–5 detik, tambahkan loading state
   - Endpoint `/audio/synthesize` return binary WAV — gunakan `response.blob()`

---

## 5. Verification & Testing Strategy

- **Unit tests** (`internal/usecase/audio_usecase_test.go`): Mock
  `AudioTranscriber`, `AudioSynthesizer`, dan `CharacterChatUsecase`.
  Test memverifikasi bahwa `VoiceChat` memanggil `Chat()` dengan hasil
  transkripsi STT sebagai message (bukan hardcoded string).
- **Manual curl test**: Verifikasi ketiga endpoint audio dengan file audio sample.
  Cek header WAV valid via `xxd output.wav | head -1` dan playback via `aplay`.
- **Integration test** (`internal/delivery/http/handler/audio_handler_test.go`):
  Belum diimplementasikan — kandidat untuk sprint berikutnya.