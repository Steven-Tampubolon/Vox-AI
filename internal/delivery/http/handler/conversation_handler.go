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

// List semua sesi chat - support filter ?character=betawi
func (h *ConversationHandler) List(c *gin.Context) {
	character := c.Query("character")

	// Validasi karakter
	if character != "" {
		char := domain.Character(character)
		if !char.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "karakter tidak valid, pilih: betawi, rag, git, explain",
			})
			return
		}
	}

	convs, err := h.chatRepo.ListConversationsByCharacter(c.Request.Context(), character)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if convs == nil {
		convs = []*domain.Conversation{}
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": convs,
		"total":         len(convs),
	})
}

func (h *ConversationHandler) GetMessages(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id diperlukan"})
		return
	}

	// Pastikan conversation ada
	conv, err := h.chatRepo.GetConversation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation tidak ditemukan"})
		return
	}

	// Validasi karakter jika disertakan di query param
	character := c.Query("character")
	if character != "" {
		char := domain.Character(character)
		if !char.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Karakter tidak valid, pilih: betawi, rag, git, explain",
			})
			return
		}

		if conv.Character != char {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "conversation ini bukan milik karakter: " + character,
			})
			return
		}
	}

	msgs, err := h.chatRepo.GetMessages(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if msgs == nil {
		msgs = []*domain.Message{}
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation": conv,
		"messages":     msgs,
		"total":        len(msgs),
	})
}

// Delete hapus sesi chat beserta semua pesan nya
func (h *ConversationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id diperlukan"})
		return
	}

	// Pastikan conversation ada sebelum di hapus
	conv, err := h.chatRepo.GetConversation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation tidak ditemukan"})
		return
	}

	if err := h.chatRepo.DeleteConversation(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "conversation berhasil dihapus",
		"id":      id,
	})
}

// UpdateTitle rename judul sesi chat
func (h *ConversationHandler) UpdateTitle(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation id diperlukan"})
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format request tidak valid"})
		return
	}

	if body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title tidak boleh kosong"})
		return
	}

	// Pastikan conversation ada
	conv, err := h.chatRepo.GetConversation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation tidak ditemukan"})
		return
	}

	if err := h.chatRepo.UpdateConversationTitle(c.Request.Context(), id, body.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "judul berhasil diubah",
		"id":      id,
		"title":   body.Title,
	})
}
