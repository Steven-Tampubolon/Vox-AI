package handler_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/sqlite"
	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/handler"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/gin-gonic/gin"

	_ "modernc.org/sqlite"
)

// openTestDB membuka SQLite temporer asli (bukan mock) dan menjalankan
// migrasi chat + document, untuk integration test yang benar-benar
// memverifikasi baris di database, bukan cuma status code HTTP.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("gagal buka db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := sqlite.NewChatStore(db).Migrate(); err != nil {
		t.Fatalf("migrasi chat gagal: %v", err)
	}
	if err := sqlite.NewDocumentStore(db).Migrate(); err != nil {
		t.Fatalf("migrasi document gagal: %v", err)
	}
	return db
}

// TestDeleteConversation_AlsoDeletesDocumentsAndChunks adalah regression
// test untuk security review Temuan 3: sebelum fix ini, DELETE
// /conversations/:id hanya menghapus baris conversations+messages,
// meninggalkan documents+chunks yang berasosiasi dengan conversation itu
// tersimpan permanen (data yatim / kebocoran privasi).
func TestDeleteConversation_AlsoDeletesDocumentsAndChunks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTestDB(t)
	ctx := context.Background()

	chatStore := sqlite.NewChatStore(db)
	docStore := sqlite.NewDocumentStore(db)
	chatRepo := repository.NewSQLiteChatRepository(chatStore)
	docRepo := repository.NewSQLiteDocumentRepository(docStore)

	// --- Setup: buat 1 conversation, 1 document, 1 chunk ---
	now := time.Now()
	_, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, character, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"conv-1", "rag", "test conversation", now, now,
	)
	if err != nil {
		t.Fatalf("setup: insert conversation gagal: %v", err)
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO documents (id, conversation_id, filename, chunk_count, created_at) VALUES (?, ?, ?, ?, ?)`,
		"doc-1", "conv-1", "rahasia.pdf", 1, now,
	)
	if err != nil {
		t.Fatalf("setup: insert document gagal: %v", err)
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO chunks (document_id, content, embedding) VALUES (?, ?, ?)`,
		"doc-1", "data sensitif milik user", "[0.1,0.2,0.3]",
	)
	if err != nil {
		t.Fatalf("setup: insert chunk gagal: %v", err)
	}

	// --- Panggil DELETE /conversations/conv-1 lewat handler asli ---
	h := handler.NewConversationHandler(chatRepo, docRepo)
	router := gin.New()
	router.DELETE("/conversations/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/conversations/conv-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// --- Verifikasi LANGSUNG ke database, bukan cuma percaya status code ---
	var docCount, chunkCount, convCount int

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents WHERE conversation_id = ?`, "conv-1").Scan(&docCount); err != nil {
		t.Fatalf("query documents gagal: %v", err)
	}
	if docCount != 0 {
		t.Errorf("expected 0 dokumen tersisa untuk conv-1, masih ada %d - data yatim (Temuan 3) belum tertangani", docCount)
	}

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks WHERE document_id = ?`, "doc-1").Scan(&chunkCount); err != nil {
		t.Fatalf("query chunks gagal: %v", err)
	}
	if chunkCount != 0 {
		t.Errorf("expected 0 chunk tersisa untuk doc-1, masih ada %d - data yatim (Temuan 3) belum tertangani", chunkCount)
	}

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversations WHERE id = ?`, "conv-1").Scan(&convCount); err != nil {
		t.Fatalf("query conversations gagal: %v", err)
	}
	if convCount != 0 {
		t.Errorf("expected conversation conv-1 juga terhapus, masih ada")
	}
}

// TestDeleteConversation_NotFound memastikan fix ini tidak mengubah
// perilaku 404 untuk conversation yang memang tidak ada.
func TestDeleteConversation_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTestDB(t)

	chatRepo := repository.NewSQLiteChatRepository(sqlite.NewChatStore(db))
	docRepo := repository.NewSQLiteDocumentRepository(sqlite.NewDocumentStore(db))

	h := handler.NewConversationHandler(chatRepo, docRepo)
	router := gin.New()
	router.DELETE("/conversations/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/conversations/tidak-ada", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestDeleteConversation_WithoutDocuments memastikan menghapus conversation
// yang TIDAK PERNAH punya dokumen (kasus paling umum: chat betawi/git/explain)
// tetap berhasil normal - docRepo.DeleteByConversation harus no-op yang aman,
// bukan error, ketika tidak ada dokumen untuk dihapus.
func TestDeleteConversation_WithoutDocuments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTestDB(t)
	ctx := context.Background()

	now := time.Now()
	_, err := db.ExecContext(ctx,
		`INSERT INTO conversations (id, character, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"conv-2", "betawi", "chat biasa tanpa dokumen", now, now,
	)
	if err != nil {
		t.Fatalf("setup: insert conversation gagal: %v", err)
	}

	chatRepo := repository.NewSQLiteChatRepository(sqlite.NewChatStore(db))
	docRepo := repository.NewSQLiteDocumentRepository(sqlite.NewDocumentStore(db))

	h := handler.NewConversationHandler(chatRepo, docRepo)
	router := gin.New()
	router.DELETE("/conversations/:id", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/conversations/conv-2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}
}
