package handler

import (
	"io"
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AudioHandler struct {
	audioUC *usecase.AudioUsecase
}

func NewAudioHandler(audioUC *usecase.AudioUsecase) *AudioHandler {
	return &AudioHandler{audioUC: audioUC}
}

func (h *AudioHandler) Transcribe(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to read file"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to read file data"})
		return
	}

	res, err := h.audioUC.TranscribeAudio(c.Request.Context(), domain.AudioRequest{
		Data:     data,
		Filename: fileHeader.Filename,
		MimeType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, res.MimeType, res.AudioData)
}

func (h *AudioHandler) VoiceChat(c *gin.Context) {
	character := c.PostForm("character")
	convID := c.PostForm("conversation_id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to read file"})
		return
	}
	defer file.Close()

	data, _ := io.ReadAll(file)

	res, err := h.audioUC.VoiceChat(c.Request.Context(), character, convID, domain.AudioRequest{
		Data:     data,
		Filename: fileHeader.Filename,
		MimeType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
