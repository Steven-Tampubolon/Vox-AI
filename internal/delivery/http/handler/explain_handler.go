package handler

import (
	"context"
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
)

type ExplainHandler struct {
	useCase *usecase.ExplainUseCase
}

func NewExplainHandler(uc *usecase.ExplainUseCase) *ExplainHandler {
	return &ExplainHandler{useCase: uc}
}

func (h *ExplainHandler) Chat(c *gin.Context) {
	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format request tidak valid"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message tidak boleh kosong"})
		return
	}

	req.Character = domain.CharacterExplain

	streamChat(c, func(ctx context.Context, onChunk func(text string) error) (*domain.ChatResponse, error) {
		return h.useCase.ChatStream(ctx, &req, onChunk)
	})
}
