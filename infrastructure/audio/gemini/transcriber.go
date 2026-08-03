package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

const (
	sttBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"
	sttModel   = "gemini-3.5-flash-lite" // multimodal dan bisa terima input audio
)

// transcribePrompt - menegembalikan teks transkripsi mentah tanpa basa-basi/terjemahan/markdown
const transcribePrompt = `Transkripsikan audio berikut ke teks secara verbatim (apa adanya), 

dalam bahasa aslinya. Jangan menerjemahkan. Jangan tambahkan komentar, 
penjelasan, tanda kutip, 
atau format markdown apa pun -- kembalikan HANYA teks transkripsinya. 
Jika audio tidak berisi ucapan yang bisa dipahami, kembalikan string kosong.`

type sttInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type sttPart struct {
	Text       string         `json:"text,omitempty"`
	InlineData *sttInlineData `json:"inlineData,omitempty"`
}

type sttContent struct {
	Role  string    `json:"role,omitempty"`
	Parts []sttPart `json:"parts"`
}

type sttRequest struct {
	Contents []sttContent `json:"contents"`
}

type sttResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type Transcriber struct {
	apiKey     string
	httpClient *http.Client
}

func NewTranscriber(apiKey string) repository.AudioTranscriber {
	return &Transcriber{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (t *Transcriber) Transcribe(ctx context.Context, req domain.AudioRequest) (*domain.TranscribeResult, error) {
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("file audio kosong")
	}

	payload := sttRequest{
		Contents: []sttContent{
			{
				Role: "user",
				Parts: []sttPart{
					{Text: transcribePrompt},
					{InlineData: &sttInlineData{
						MimeType: normalizeAudioMime(req.MimeType, req.Filename),
						Data:     base64.StdEncoding.EncodeToString(req.Data),
					}},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal stt request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent", sttBaseURL, sttModel)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create stt request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", t.apiKey)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send stt request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("gagal menutup response body stt: %v", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini stt error: status %d, body: %s", resp.StatusCode, string(errBody))
	}

	var result sttResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode stt response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini tidak mengembalikan transkripsi")
	}

	text := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)

	return &domain.TranscribeResult{
		Text:     text,
		Language: "id",
		Duration: 0, // field durasi dibiarkan 0, karena gemini tidak mengembalikan durasi audio
	}, nil
}

// normalizeAudioMime bersihin mime type dari parameter tambahan
func normalizeAudioMime(mimeType, filename string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = mimeType[:idx]
	}

	if mimeType != "" && mimeType != "application/octet-stream" {
		return mimeType
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		return "audio/wav"
	case ".mp3":
		return "audio/mp3"
	case ".ogg":
		return "audio/ogg"
	case ".flac":
		return "audio/flac"
	case ".acc":
		return "audio/acc"
	case ".webm":
		return "audio/webm"
	default:
		return "audio/wav" // fallback konservatif
	}
}
