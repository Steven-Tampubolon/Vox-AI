package handler

import (
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/gin-gonic/gin"
)

type CharacterHandler struct {
}

func NewCharacterHandler() *CharacterHandler {
	return &CharacterHandler{}
}

// List kembalikan semua karakter beserta metadata-nya
func (h *CharacterHandler) List(c *gin.Context) {
	character := domain.GetAllCharacters()
	c.JSON(http.StatusOK, gin.H{
		"characters": character,
		"total":      len(character),
	})
}
