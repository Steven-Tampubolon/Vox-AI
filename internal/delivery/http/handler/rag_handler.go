package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
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

	// Baca semua bytes SEKALI - dipakai untuk validasi dan ekstrak
	rawBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal membaca file"})
		return
	}

	if len(rawBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file kosong"})
		return
	}

	// Validasi ekstensi + MIME dari bytes (bukan dari file pointer)
	mimeType, valid := validateFile(rawBytes, header.Filename)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("file tidak valid, MIME terdeteksi: %s", mimeType),
		})
		return
	}

	// Ekstrak teks sesuai tipe file
	var textContent string
	if strings.HasPrefix(mimeType, "application/pdf") {
		textContent, err = extractPDFText(rawBytes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("gagal ekstrak teks dari PDF: %s", err.Error())})
			return
		}
	} else {
		// TXT - langsung konversi bytes ke string
		textContent = string(rawBytes)
	}

	// Validasi hasil ekstrak tidak kosong
	textContent = strings.TrimSpace(textContent)
	if textContent == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tidak ada teks yang bisa diekstrak. Pastikan PDF bukan hasil scan.",
		})
		return
	}

	// Index dokumen ke RAG
	doc, err := h.useCase.IndexDocument(
		c.Request.Context(),
		conversationID,
		header.Filename,
		textContent,
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

// --- Helper functions ---

// validateFile validasi ekstensi dan MIME dari bytes yang sudah dibaca
func validateFile(data []byte, filename string) (string, bool) {
	// Cek ekstensi file
	ext := strings.ToLower(filepath.Ext(filename))

	allowed := map[string]string{
		".txt": "text/plain",
		".pdf": "application/pdf",
	}

	expectedMIME, ok := allowed[ext]
	if !ok {
		return "ekstensi tidak didukung", false
	}

	// Deteksi MIME dari actual bytes (max 512 bytes pertama)
	limit := 512
	if len(data) < limit {
		limit = len(data)
	}
	detectedMIME := http.DetectContentType(data[:limit])

	// Cek apakah MIME sesuai dengan ekpektasi
	if !strings.HasPrefix(detectedMIME, expectedMIME) {
		return detectedMIME, false
	}

	return detectedMIME, true
}

// extractPDFText ekstrak teks dari PDF menggunakan ledongthuc/pdf
func extractPDFText(data []byte) (string, error) {
	reader := bytes.NewReader(data)

	r, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("gagal membuka PDF: %w", err)
	}

	var result strings.Builder
	numPages := r.NumPage()

	if numPages == 0 {
		return "", fmt.Errorf("PDF tidak memiliki halaman")
	}

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Skip halaman gagal, lanjut halama berikutnya
			continue
		}

		result.WriteString(text)
		result.WriteString("\n\n")
	}

	return result.String(), nil
}
