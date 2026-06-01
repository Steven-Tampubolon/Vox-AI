package handler

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RAGHandler struct {
	useCase *usecase.RAGUseCase
}

func NewRAGHandler(uc *usecase.RAGUseCase) *RAGHandler {
	return &RAGHandler{useCase: uc}
}

// Chat tanya jawab berdasarkan dokumen yang sudah diupload
func (h *RAGHandler) Chat(c *gin.Context) {
	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format request tidak valid"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message tidak boleh kosong"})
		return
	}

	if req.ConversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id wajib diisi, upload dokumen terlebih dahulu"})
		return
	}

	req.Character = domain.CharacterRAG

	resp, err := h.useCase.Chat(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UploadDocument terima file dari frontend, index untuk RAG
func (h *RAGHandler) UploadDocument(c *gin.Context) {
	// Ambil conversation_id dari form, buat baru jika kosong
	conversationID := c.PostForm("conversation_id")
	if conversationID == "" {
		conversationID = uuid.New().String()
	}

	// ambil file dari multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file tidak ditemukan dalam request"})
		return
	}
	defer file.Close()

	// Validasi ekstensi dan MIME type
	filename := header.Filename
	if !isAllowedFile(filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "hanya file .txt, .md dan .pdf yang didukung saat ini",
		})
		return
	}

	if !isAllowedMimeType(file) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tipe file tidak valid",
		})
	}

	// Baca isi file
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal membaca file"})
		return
	}

	if len(content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file kosong"})
		return
	}

	// Index dokumen ke RAG
	doc, err := h.useCase.IndexDocument(
		c.Request.Context(),
		conversationID,
		filename,
		string(content),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": conversationID,
		"document_id":     doc.ID,
		"filename":        doc.Filename,
		"chunk_count":     doc.ChunkCount,
		"message":         "dokumen berhasil diindeks, silahkan mulai bertanya",
	})
}

// isAllowedFile validasi ekstensi file yang diizinkan
func isAllowedFile(filename string) bool {
	allowed := map[string]bool{
		".txt": true,
		".md":  true,
		".pdf": true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	return allowed[ext]
}

// isAllowedMimeType validasi MIME type yang dizinkan
func isAllowedMimeType(file multipart.File) bool {
	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return false
	}

	mimeType := http.DetectContentType(buffer[:n])

	allowed := map[string]bool{
		"text/plain; charset=utf-8":    true,
		"application/pdf":              true,
		"text/markdown; charset=utf-8": true,
	}

	return allowed[mimeType]
}
