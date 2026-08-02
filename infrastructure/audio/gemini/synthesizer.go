package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

const (
	ttsBaseURL   = "https://generativelanguage.googleapis.com/v1beta/models"
	ttsModel     = "gemini-2.5-flash-preview-tts" // model khusus TTS preview
	defaultVoice = "Pulcherrima"
)

type ttsPart struct {
	Text string `json:"text"`
}

type ttsContent struct {
	Parts []ttsPart `json:"parts"`
}

type ttsPrebuildVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type ttsVoiceConfig struct {
	PrebuiltVoiceConfig ttsPrebuildVoiceConfig `json:"prebuiltVoiceConfig"`
}

type ttsSpeechConfig struct {
	VoiceConfig ttsVoiceConfig `json:"voiceConfig"`
}

type ttsGenerateConfig struct {
	ResponseModalities []string        `json:"responseModalities"`
	SpeechConfig       ttsSpeechConfig `json:"speechConfig"`
}

type ttsRequest struct {
	Contents         []ttsContent      `json:"contents"`
	GenerationConfig ttsGenerateConfig `json:"generationConfig"`
}

type ttsResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				InlineData struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type Synthesizer struct {
	apiKey     string
	httpClient *http.Client
}

func NewSynthesizer(apiKey string) repository.AudioSynthesizer {
	return &Synthesizer{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *Synthesizer) Synthesize(ctx context.Context, req domain.SynthesizeRequest) (*domain.SynthesizeResult, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("text tidak boleh kosong")
	}

	voice := req.VoiceID
	if voice == "" || voice == "default" {
		voice = defaultVoice
	}

	payload := ttsRequest{
		Contents: []ttsContent{{Parts: []ttsPart{{Text: req.Text}}}},
		GenerationConfig: ttsGenerateConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: ttsSpeechConfig{
				VoiceConfig: ttsVoiceConfig{
					PrebuiltVoiceConfig: ttsPrebuildVoiceConfig{VoiceName: voice},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal tts request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent", ttsBaseURL, ttsModel)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create tts request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send tts request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("gagal menutup response body tts: %v", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini tts error: status %d, body: %s", resp.StatusCode, string(errBody))
	}

	var result ttsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode tts response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini tidak mengembalikan audio")
	}

	part := result.Candidates[0].Content.Parts[0].InlineData
	if part.Data == "" {
		return nil, fmt.Errorf("gemini tidak mengembalikan audio data")
	}

	pcm, err := base64.StdEncoding.DecodeString(part.Data)
	if err != nil {
		return nil, fmt.Errorf("decode base64 audio: %w", err)
	}

	// Gemini TTS mengembalikan RAW PCM 16-bit mono (bukan file WAV utuh)
	sampleRate := parsePCMRate(part.MimeType)
	wavData := pcmToWav(pcm, sampleRate, 1, 16)

	return &domain.SynthesizeResult{
		AudioData: wavData,
		MimeType:  "audio/wav",
	}, nil
}

// parsePCMRate ambil angka rate dari mimeType
func parsePCMRate(mimeType string) int {
	const key = "rate"
	idx := strings.Index(mimeType, key)
	if idx == -1 {
		return 24000
	}
	rateStr := mimeType[idx+len(key):]
	if semi := strings.Index(rateStr, ";"); semi != -1 {
		rateStr = rateStr[:semi]
	}
	rate, err := strconv.Atoi(strings.TrimSpace(rateStr))
	if err != nil || rate <= 0 {
		return 24000
	}
	return rate
}

// pcmToWav bungkus raw PCM 16-bit little-endian jadi file WAV yang valid
func pcmToWav(pcm []byte, sampleRate, numChannels, bitsPerSample int) []byte {
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := len(pcm)

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	writeUint32LE(buf, uint32(36+dataSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt")
	writeUint32LE(buf, 16) // ukuran sub-chunk fmt (PCM = 16)
	writeUint16LE(buf, 1)  // audio format 1 = PCM (uncompressed)
	writeUint16LE(buf, uint16(numChannels))
	writeUint32LE(buf, uint32(sampleRate))
	writeUint32LE(buf, uint32(byteRate))
	writeUint16LE(buf, uint16(blockAlign))
	writeUint16LE(buf, uint16(bitsPerSample))

	buf.WriteString("data")
	writeUint32LE(buf, uint32(dataSize))
	buf.Write(pcm)

	return buf.Bytes()
}

func writeUint32LE(buf *bytes.Buffer, v uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	buf.Write(b)
}

func writeUint16LE(buf *bytes.Buffer, v uint16) {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	buf.Write(b)
}
