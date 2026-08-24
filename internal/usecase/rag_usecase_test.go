package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/domain"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
)

// mockDocRepo adalah in-memory fake untuk repository.DocumentRepository.
type mockDocRepo struct {
	mu        sync.Mutex
	documents map[string]*domain.Document
	chunks    []*domain.Chunk // urutan insert - dipakai untuk verifikasi urutan chunk
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{documents: map[string]*domain.Document{}}
}

func (m *mockDocRepo) SaveDocument(ctx context.Context, doc *domain.Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.documents[doc.ID] = doc
	return nil
}

func (m *mockDocRepo) SaveChunk(ctx context.Context, chunk *domain.Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunks = append(m.chunks, chunk)
	return nil
}

func (m *mockDocRepo) GetChunksByConversation(ctx context.Context, conversationID string) ([]*domain.Chunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.chunks, nil
}

func (m *mockDocRepo) DeleteByConversation(ctx context.Context, conversationID string) error {
	return nil
}

// TestIndexDocument_ChunksSavedInOriginalOrder adalah regression test untuk
// step 12 (concurrent embedding): meskipun Embed() dijalankan konkuren dan
// mock ini sengaja membuat chunk yang lebih belakang selesai lebih dulu
// (delay terbalik), hasil akhir yang disimpan ke DB harus tetap berurutan
// sesuai chunk aslinya, dan tiap chunk harus punya embedding yang benar
// (bukan tertukar dengan chunk lain).
func TestIndexDocument_ChunksSavedInOriginalOrder(t *testing.T) {
	chatRepo := newMockChatRepo()
	docRepo := newMockDocRepo()

	// bikin content panjang supaya splitIntoChunks hasilkan banyak chunk,
	// tiap chunk isinya unik (ada urutan angka) supaya gampang dicek.
	var sb strings.Builder
	for i := 0; i < 12; i++ {
		sb.WriteString(fmt.Sprintf("chunk-marker-%02d ", i))
		sb.WriteString(strings.Repeat("x", 45)) // ~500 char per chunk (chunkMaxChars)
		sb.WriteString(" ")
	}
	content := sb.String()

	aiRepo := &mockAIRepo{
		generateFn: nil,
		streamFn:   nil,
	}
	// embedFn kita override manual di bawah karena mockAIRepo.Embed selama
	// ini fixed - lihat catatan di bawah kalau perlu extend mockAIRepo.
	_ = aiRepo

	embedder := &orderedDelayAIRepo{}
	uc := usecase.NewRAGUseCase(embedder, chatRepo, docRepo)

	doc, err := uc.IndexDocument(context.Background(), "conv-1", "test.txt", content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if doc.ChunkCount != len(docRepo.chunks) {
		t.Fatalf("expected %d chunks saved, got %d", doc.ChunkCount, len(docRepo.chunks))
	}

	for _, chunk := range docRepo.chunks {
		wantEmbedding := embedder.expectedEmbedding(chunk.Content)
		if len(chunk.Embedding) != 1 || chunk.Embedding[0] != wantEmbedding {
			t.Errorf("chunk %q got embedding %v, want [%v] - kemungkinan urutan/index tertukar akibat konkurensi",
				chunk.Content[:20], chunk.Embedding, wantEmbedding)
		}
	}
}

// orderedDelayAIRepo adalah AIRepository fake yang deliberately membuat chunk
// dengan index lebih besar selesai LEBIH DULU (delay terbalik), untuk
// memaksa out-of-order completion di embedChunksConcurrently - supaya test
// benar-benar menguji bahwa hasil tetap dikembalikan sesuai urutan index,
// bukan urutan selesai.
type orderedDelayAIRepo struct {
	mu   sync.Mutex
	seen int
}

func (r *orderedDelayAIRepo) expectedEmbedding(text string) float64 {
	return float64(len(text))
}

func (r *orderedDelayAIRepo) Embed(ctx context.Context, text string) ([]float64, error) {
	r.mu.Lock()
	callIndex := r.seen
	r.seen++
	r.mu.Unlock()

	// panggilan belakangan (index besar) sengaja delay LEBIH PENDEK,
	// supaya selesai duluan dibanding panggilan awal.
	delay := time.Duration(20-callIndex%20) * time.Millisecond
	time.Sleep(delay)

	return []float64{r.expectedEmbedding(text)}, nil
}

func (r *orderedDelayAIRepo) Generate(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
	return "", nil
}

func (r *orderedDelayAIRepo) GenerateStream(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
	return nil
}

// TestIndexDocument_EmbedErrorPropagates memastikan kalau salah satu chunk
// gagal di-embed, IndexDocument tetap mengembalikan error (fail-fast),
// sama seperti perilaku versi sekuensial sebelumnya.
func TestIndexDocument_EmbedErrorPropagates(t *testing.T) {
	chatRepo := newMockChatRepo()
	docRepo := newMockDocRepo()
	wantErr := errors.New("gemini quota habis")

	failingRepo := &failingAIRepo{failErr: wantErr}
	uc := usecase.NewRAGUseCase(failingRepo, chatRepo, docRepo)

	content := strings.Repeat("lorem ipsum dolor sit amet ", 100) // multi-chunk

	_, err := uc.IndexDocument(context.Background(), "conv-1", "test.txt", content)
	if err == nil {
		t.Fatalf("expected error to propagate from failing Embed call")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error to be wantErr (via errors.Is), got: %v", err)
	}
}

// failingAIRepo membuat SETIAP panggilan Embed gagal.
type failingAIRepo struct {
	failErr error
}

func (r *failingAIRepo) Embed(ctx context.Context, text string) ([]float64, error) {
	return nil, r.failErr
}

func (r *failingAIRepo) Generate(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
	return "", nil
}

func (r *failingAIRepo) GenerateStream(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
	return nil
}
