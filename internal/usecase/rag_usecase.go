package usecase

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/google/uuid"
)

const (
	chunkMaxChars       = 500 // ukuran maksimal karakter per chunk dokumen
	topKChunks          = 3   // jumlah chunk paling relevan yang diambil per query
	maxConcurrentEmbeds = 5   // batas gorroutine embed yang jalan bersamaan ke Gemini
)

const ragSystemPrompt = `Kamu adalah Dokter Dokumen - asisten yang sangat teliti dalam membaca dan menganalisis dokumen.

Awali dengan perkenalan:
"Halo VOX! Perkenalkan saya Dokter Dokumen,
saya akan bantu jawab pertanyaan berdasarkan dokumen yang VOX berikan."

Tugasmu:
1. JAWAB PERTANYAAN - jawab hanya berdasarkan isi dokumen yang diberikan
2. RINGKASAN - buat ringkasan singkat dan padat jika diminta
3. KESIMPULAN -  tarik kesimpulan utama dari dokumen jika diminta

Aturan ketat:
- Jika informasi TIDAK ADA di dokumen, katakan dengan jujur: "Informasi ini tidak ada di di dokumen."
- Jangan mengarang atau menambah informasi dari luar dokumen
- Selalu sebutkan dari bagian mana informasi diambil jika memungkinkan
- Gunakan bahasa yang sama dengan dokumen (Indonesia atau Inggris)

Jika tidak ada dokumen yang diunggah, minta user untuk upload dokumen terlebih dahulu.

Selalu akhiri dengan tawaran: "Ada yang mau ditanyakan lagi mengenai dokumen ini VOX?"`

type RAGUseCase struct {
	aiRepo   repository.AIRepository
	chatRepo repository.ChatRepository
	docRepo  repository.DocumentRepository
}

func NewRAGUseCase(
	aiRepo repository.AIRepository,
	chatRepo repository.ChatRepository,
	docRepo repository.DocumentRepository,
) *RAGUseCase {
	return &RAGUseCase{
		aiRepo:   aiRepo,
		chatRepo: chatRepo,
		docRepo:  docRepo,
	}
}

// IndexDocument proses dokumen: chunk > embed > simpan
func (uc *RAGUseCase) IndexDocument(ctx context.Context, conversationID, filename, content string) (*domain.Document, error) {
	// 1. Buat atau pastikan conversation ada
	conv, err := uc.chatRepo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		now := time.Now()
		conv := &domain.Conversation{
			ID:        conversationID,
			Character: domain.CharacterRAG,
			Title:     "RAG - " + filename,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.chatRepo.SaveConversation(ctx, conv); err != nil {
			return nil, err
		}
	}

	// 2. Hapus dokumen lama di conversation ini
	if err := uc.docRepo.DeleteByConversation(ctx, conversationID); err != nil {
		return nil, fmt.Errorf("delete old document: %w", err)
	}

	// 3. Potong dokumen jadi chunk
	chunks := splitIntoChunks(content, chunkMaxChars)

	// 4. Buat metadata dokumen
	doc := &domain.Document{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Filename:       filename,
		ChunkCount:     len(chunks),
		CreatedAt:      time.Now(),
	}
	if err := uc.docRepo.SaveDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("save document: %w", err)
	}

	// 5. Embed semua chunk secara konkuren (dibatasi maxConcurrentEmbeds), lalu simpan berurutan sesuai urutan chunk asli.
	embeddings, err := uc.embedChunksConcurrently(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("embed chunk: %w", err)
	}
	for i, chunkText := range chunks {
		chunk := &domain.Chunk{
			DocumentID: doc.ID,
			Content:    chunkText,
			Embedding:  embeddings[i],
		}
		if err := uc.docRepo.SaveChunk(ctx, chunk); err != nil {
			return nil, fmt.Errorf("save chunk: %w", err)
		}
	}

	return doc, nil
}

// embedChunksConcurrently menjalankan uc.aiRepo.Embed untuk setiap chunk secara konkuren
func (uc *RAGUseCase) embedChunksConcurrently(ctx context.Context, chunks []string) ([][]float64, error) {
	embeddings := make([][]float64, len(chunks))

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, maxConcurrentEmbeds)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	for i, chunkText := range chunks {
		i, chunkText := i, chunkText // hindari capture loop variable
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-childCtx.Done():
				return
			}

			embedding, err := uc.aiRepo.Embed(childCtx, chunkText)
			if err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			embeddings[i] = embedding
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return embeddings, nil
}

// Chat tanya jawab berdasarkan dokumen - versi non-stream
func (uc *RAGUseCase) Chat(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterRAG, uc.chatRepo)
	if err != nil {
		return nil, fmt.Errorf("get or create conversation: %w", err)
	}

	// 1. Simpen pesan user
	userMsg := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleUser,
		Content:        req.Message,
		CreatedAt:      time.Now(),
	}
	if err := uc.chatRepo.SaveMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// 2. embed query user + cari chunk  relevan
	systemWithContext, err := uc.buildContextPrompt(ctx, conv.ID, req.Message)
	if err != nil {
		return nil, err
	}

	// 3. Ambil history + kirim ke gemini
	history, err := buildHistory(ctx, conv.ID, uc.chatRepo)
	if err != nil {
		return nil, fmt.Errorf("build history: %w", err)
	}

	reply, err := uc.aiRepo.Generate(ctx, systemWithContext, history)
	if err != nil {
		return nil, fmt.Errorf("generate reply: %w", err)
	}

	// 4. Simpan jadi jawaban AI
	aiMsg := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleAssistant,
		Content:        reply,
		CreatedAt:      time.Now(),
	}
	if err := uc.chatRepo.SaveMessage(ctx, aiMsg); err != nil {
		return nil, fmt.Errorf("save ai message: %w", err)
	}

	return &domain.ChatResponse{
		ConversationID: conv.ID,
		Character:      domain.CharacterRAG,
		Reply:          reply,
	}, nil
}

// ChatStream - versi streaming untuk SSE.
// Retrieval (embed query + cari chunk relevan) tetap dilakukan blocking di awal -
// baru setelah context dokumen siap, generation di-stream ke onChunk.
func (uc *RAGUseCase) ChatStream(ctx context.Context, req *domain.ChatRequest, onChunk func(text string) error) (*domain.ChatResponse, error) {
	conv, err := getOrCreateConversation(ctx, req, domain.CharacterRAG, uc.chatRepo)
	if err != nil {
		return nil, fmt.Errorf("get or create conversation: %w", err)
	}

	userMsg := &domain.Message{
		ConversationID: conv.ID,
		Role:           domain.RoleUser,
		Content:        req.Message,
		CreatedAt:      time.Now(),
	}
	if err := uc.chatRepo.SaveMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	// Retrieval TIDAK di-stream - user akan lihat "typing indicator" selama
	// bagian ini jalan, baru teks mulai muncul begitu Generate dimulai
	systemWithContext, err := uc.buildContextPrompt(ctx, conv.ID, req.Message)
	if err != nil {
		return nil, err
	}

	history, err := buildHistory(ctx, conv.ID, uc.chatRepo)
	if err != nil {
		return nil, fmt.Errorf("build history: %w", err)
	}

	var fullReply strings.Builder
	streamErr := uc.aiRepo.GenerateStream(ctx, systemWithContext, history, func(chunk string) error {
		fullReply.WriteString(chunk)
		return onChunk(chunk)
	})

	reply := fullReply.String()

	if reply != "" {
		aiMsg := &domain.Message{
			ConversationID: conv.ID,
			Role:           domain.RoleAssistant,
			Content:        reply,
			CreatedAt:      time.Now(),
		}
		if saveErr := uc.chatRepo.SaveMessage(ctx, aiMsg); saveErr != nil {
			return nil, fmt.Errorf("save ai message: %w", saveErr)
		}
	}

	if streamErr != nil {
		return nil, fmt.Errorf("generate stream reply: %w", streamErr)
	}

	if reply == "" {
		return nil, fmt.Errorf("gemini tidak mengembalikan jawaban")
	}

	return &domain.ChatResponse{
		ConversationID: conv.ID,
		Character:      domain.CharacterRAG,
		Reply:          reply,
	}, nil
}

// --- Helper functions ---

// buildContextPrompt: embed query + cari chunk relevan + gabung ke system prompt
func (uc *RAGUseCase) buildContextPrompt(ctx context.Context, conversationID, message string) (string, error) {
	queryEmbedding, err := uc.aiRepo.Embed(ctx, message)
	if err != nil {
		return "", fmt.Errorf("embed chunks: %w", err)
	}

	allChunks, err := uc.docRepo.GetChunksByConversation(ctx, conversationID)
	if err != nil {
		return "", fmt.Errorf("get chunks: %w", err)
	}

	relevanChunks := findTopK(allChunks, queryEmbedding, topKChunks)

	if len(relevanChunks) > 0 {
		docContext := strings.Join(relevanChunks, "\n\n---\n\n")
		return fmt.Sprintf("%s\n\nDOKUMEN RELEVAN:\n%s", ragSystemPrompt, docContext), nil
	}
	return ragSystemPrompt, nil
}

// SplitIntoChunks potong teks per N karakter, jaga batas kalimat
func splitIntoChunks(text string, maxChars int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if current.Len()+len(para) > maxChars && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

// findTopK cari K chunk mirip dengan query berdasarkan cosine similarity
func findTopK(chunks []*domain.Chunk, queryEmbedding []float64, k int) []string {
	type scored struct {
		text  string
		score float64
	}

	var results []scored
	for _, chunk := range chunks {
		score := cosineSimilarity(queryEmbedding, chunk.Embedding)
		results = append(results, scored{chunk.Content, score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var texts []string
	for i := 0; i < k && i < len(results); i++ {
		texts = append(texts, results[i].text)
	}
	return texts
}

// cosineSimilarity hitung kemiripan dua vector - pure Go tanpa library
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
