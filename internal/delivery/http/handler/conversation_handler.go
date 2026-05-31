package handler

import (
	"net/http"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	chatRepo repository.ChatRepository
}

func NewConversationHandler(chatRepo repository.ChatRepository) *ConversationHandler {
	return &ConversationHandler{chatRepo: chatRepo}
}

func (h *ConversationHandler) List(c *gin.Context) {
	convs, err := h.chatRepo.ListConversations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if convs == nil {
		convs = []*domain.Conversation{}
	}

	c.JSON(http.StatusOK, gin.H{"conversations": convs})
}

func (h *ConversationHandler) GetMessages(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id diperlukan"})
		return
	}

	msgs, err := h.chatRepo.GetMessages(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if msgs == nil {
		msgs = []*domain.Message{}
	}

	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}
