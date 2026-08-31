package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AudioHandler struct {
	audioUC *usecase.AudioUseCase
}

func NewAudioHandler(audioUC *usecase.AudioUseCase) *AudioHandler {
	return &AudioHandler{audioUC: audioUC}
}

func (h *AudioHandler) Transcribe(c *gin.Context) {
	// Batasi ukuran body request agar file audio besar tidak menyebabkan OOM
	// (konsisten dengan maxUploadSize di rag_handler.go)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("file audio terlalu besar, maksimal %d MB", maxUploadSize/(1<<20)),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to read file"})
		return
	}
	defer func() {
		_ = file.Close()

	}()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("file audio terlalu besar, maksimal %d MB", maxUploadSize/(1<<20)),
		})
		return
	}

	res, err := h.audioUC.Transcribe(c.Request.Context(), domain.AudioRequest{
		Data:     data,
		Filename: fileHeader.Filename,
		MimeType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		respondInternalError(c, "audio.Transcribe", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *AudioHandler) Synthesize(c *gin.Context) {
	var req domain.SynthesizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.audioUC.SynthesizeText(c.Request.Context(), req)
	if err != nil {
		// 🚨 LOG ERROR INI AKAN MENAMPILKAN PESAN DARI GEMINI DI TERMINAL
		fmt.Printf("\n[GEMINI ERROR DETECTED]: %v\n\n", err)

		// Simpan error ke Gin Context agar tercium oleh middleware logger jika diperlukan
		_ = c.Error(err)

		if strings.Contains(err.Error(), "status 429") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Quota Gemini habis / Rate limit exceeded"})
			return
		}

		respondInternalError(c, "audio.Synthesize", err)
		return
	}

	c.Data(http.StatusOK, res.MimeType, res.AudioData)
}

func (h *AudioHandler) VoiceChat(c *gin.Context) {
	// Batasi ukuran body request agar file audio besar tidak menyebabkan OOM
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	character := c.PostForm("character")
	convID := c.PostForm("conversation_id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("file audio terlalu besar, maksimal %d MB", maxUploadSize/(1<<20)),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to read file"})
		return
	}
	defer func() {
		_ = file.Close()

	}()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("file audio terlalu besar, maksimal %d MB", maxUploadSize/(1<<20)),
		})
		return
	}

	res, err := h.audioUC.VoiceChat(c.Request.Context(), character, convID, domain.AudioRequest{
		Data:     data,
		Filename: fileHeader.Filename,
		MimeType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		respondInternalError(c, "audio.VoiceChat", err)
		return
	}

	c.JSON(http.StatusOK, res)
}
