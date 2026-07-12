package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/gin-gonic/gin"
)

// chatStreanFunc - bentuk seragam dari semua usecase.ChatStream
type chatStreamFunc func(ctx context.Context, onChunk func(text string) error) (*domain.ChatResponse, error)

// writeSSE menulis satu event SSE (format: "data: <json>\n\n") dan langsung
// flush supaya browser menerimanya saat itu juga, tidak menunggu buffer penuh
func writeSSE(c *gin.Context, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// writeSSEDone - menulis penanda akhir stream literal "[DONE]"
// Frontend cukup dengarkan 1 sinyal ini untuk tahu kapan berhenti "listen"
func writeSSEDone(c *gin.Context) {
	if _, err := fmt.Fprint(c.Writer, "data: [DONE]\n\n"); err != nil {
		log.Printf("failed to write sse done: %v", err)
		return
	}
	c.Writer.Flush()
}

// streamChat menyiapkan Header SSE, menjalankan fn (pembungkus usecase.ChatStream)
// lalu menulis event yang sesuai berdasarkan hasilnya dipakai handler character
func streamChat(c *gin.Context, fn chatStreamFunc) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	// c.Request.Context() otomatis ke-cancel kalau koneksi client putus
	ctx := c.Request.Context()

	resp, err := fn(ctx, func(text string) error {
		return writeSSE(c, gin.H{"content": text})
	})

	switch {
	case err == nil:

		// sukses penuh - kirim conversation_id, penting untuk percakapan baru
		if resp != nil {
			_ = writeSSE(c, gin.H{"conversation_id": resp.ConversationID})
		}

	case errors.Is(err, context.Canceled):

		log.Printf("stream dihentikan oleh client: %v", err)

	default:

		// error asli: Gemini overload, DB gagal simpan, dsb
		log.Printf("stream error: %v", err)
		_ = writeSSE(c, gin.H{"error": err.Error()})
	}

	writeSSEDone(c)
}
