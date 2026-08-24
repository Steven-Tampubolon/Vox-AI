package usecase

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
)

// concurrencyTrackingAIRepo melacak jumlah panggilan Embed yang sedang
// berjalan bersamaan (in-flight), untuk memverifikasi semaphore
// maxConcurrentEmbeds benar-benar membatasi konkurensi.
type concurrencyTrackingAIRepo struct {
	current   int32
	maxSeenMu sync.Mutex
	maxSeen   int32
}

func (r *concurrencyTrackingAIRepo) Embed(ctx context.Context, text string) ([]float64, error) {
	cur := atomic.AddInt32(&r.current, 1)
	defer atomic.AddInt32(&r.current, -1)

	r.maxSeenMu.Lock()
	if cur > r.maxSeen {
		r.maxSeen = cur
	}
	r.maxSeenMu.Unlock()

	time.Sleep(5 * time.Millisecond) // simulasi network round-trip
	return []float64{1}, nil
}

func (r *concurrencyTrackingAIRepo) Generate(ctx context.Context, systemPrompt string, history []repository.ChatMessage) (string, error) {
	return "", nil
}

func (r *concurrencyTrackingAIRepo) GenerateStream(ctx context.Context, systemPrompt string, history []repository.ChatMessage, onChunk func(string) error) error {
	return nil
}

// TestEmbedChunksConcurrently_RespectsConcurrencyLimit memverifikasi
// embedChunksConcurrently tidak pernah menjalankan lebih dari
// maxConcurrentEmbeds panggilan Embed secara bersamaan, walau jumlah chunk
// jauh lebih banyak dari batasnya.
func TestEmbedChunksConcurrently_RespectsConcurrencyLimit(t *testing.T) {
	tracker := &concurrencyTrackingAIRepo{}
	uc := &RAGUseCase{aiRepo: tracker}

	chunks := make([]string, maxConcurrentEmbeds*4)
	for i := range chunks {
		chunks[i] = fmt.Sprintf("chunk-%d", i)
	}

	embeddings, err := uc.embedChunksConcurrently(context.Background(), chunks)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(embeddings) != len(chunks) {
		t.Fatalf("expected %d embeddings, got %d", len(chunks), len(embeddings))
	}
	if tracker.maxSeen > int32(maxConcurrentEmbeds) {
		t.Errorf("concurrency limit violated: max %d in-flight, want <= %d", tracker.maxSeen, maxConcurrentEmbeds)
	}
	if tracker.maxSeen < 2 {
		t.Errorf("expected embeds to actually run concurrently (maxSeen=%d), test may not be exercising concurrency", tracker.maxSeen)
	}
}
