package handler

import (
	"context"
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
)

type BetawiHandler struct {
	useCase *usecase.BetawiUseCase
}

func NewBetawiHandler(uc *usecase.BetawiUseCase) *BetawiHandler {
	return &BetawiHandler{useCase: uc}
}

func (h *BetawiHandler) Chat(c *gin.Context) {
	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format request tidak valid"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message tidak boleh kosong"})
		return
	}

	req.Character = domain.CharacterBetawi

	streamChat(c, func(ctx context.Context, onChunk func(text string) error) (*domain.ChatResponse, error) {
		return h.useCase.ChatStream(ctx, &req, onChunk)
	})
}
