package handler

import (
	"context"
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
)

type GitHandler struct {
	useCase *usecase.GitUseCase
}

func NewGitHandler(uc *usecase.GitUseCase) *GitHandler {
	return &GitHandler{useCase: uc}
}

func (h *GitHandler) Chat(c *gin.Context) {
	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format request tidak valid"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message tidak boleh kosong"})
		return
	}

	req.Character = domain.CharacterGit

	streamChat(c, func(ctx context.Context, onChunk func(text string) error) (*domain.ChatResponse, error) {
		return h.useCase.ChatStream(ctx, &req, onChunk)
	})
}
