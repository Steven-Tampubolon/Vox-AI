package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	baseURL = "https://generativelanguage.googleapis.com/v1beta/models"
	model   = "gemini-2.5-flash-lite"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// --- Request / Response structs ---

type Part struct {
	Text string `json:"text"`
}

type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

type GenerateRequest struct {
	SystemInstruction *Content  `json:"system_instruction,omitempty"`
	Contents          []Content `json:"contents"`
}

type Candidate struct {
	Content Content `json:"content"`
}

type GenerateResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// --- Public method ---

// Generate mengirim pesan ke Gemini dan mengembalikan teks jawaban
func (c *Client) Generate(ctx context.Context, systemPrompt string, history []Content) (string, error) {
	req := GenerateRequest{
		Contents: history,
	}

	if systemPrompt != "" {
		req.SystemInstruction = &Content{
			Parts: []Part{{Text: systemPrompt}},
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent", baseURL, model)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// HEADER
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error: status %d", resp.StatusCode)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	parts := result.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return "", fmt.Errorf("no parts in response")
	}

	return parts[0].Text, nil
}

// Embed mengubah teks menjadi vector embedding
func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	type EmbedRequest struct {
		Model   string  `json:"model"`
		Content Content `json:"content"`
	}

	type EmbedResponse struct {
		Embedding struct {
			Values []float64 `json:"values"`
		} `json:"embedding"`
	}

	payload := EmbedRequest{
		Model:   "models/gemini-embedding-001",
		Content: Content{Parts: []Part{{Text: text}}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}

	// HEADER
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send embed request: %w", err)
	}
	defer resp.Body.Close()

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	return result.Embedding.Values, nil
}
