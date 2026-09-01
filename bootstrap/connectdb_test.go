package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/sqlite"
)

// TestConnectDB_EnforcesForeignKeys adalah regression test untuk security
// review Temuan 4: sebelum fix ini, SQLite tidak menegakkan FOREIGN KEY
// sama sekali walau skema tabel mendeklarasikannya - insert message dengan
// conversation_id yang tidak eksis akan berhasil begitu saja (data rusak
// diam-diam, tidak ketahuan). Setelah fix, insert semacam ini HARUS gagal.
func TestConnectDB_EnforcesForeignKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := connectDB(dbPath)
	if err != nil {
		t.Fatalf("connectDB gagal: %v", err)
	}
	defer db.Close()

	store := sqlite.NewChatStore(db)
	if err := store.Migrate(); err != nil {
		t.Fatalf("migrate gagal: %v", err)
	}

	// Coba insert message langsung ke conversation_id yang TIDAK PERNAH dibuat
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, ?, ?)`,
		"conversation-yang-tidak-ada", "user", "halo",
	)
	if err == nil {
		t.Fatal("expected FOREIGN KEY constraint error, tapi insert berhasil - FK enforcement tidak aktif")
	}
}

// TestConnectDB_AllowsValidForeignKey memastikan fix di atas TIDAK menolak
// insert yang valid (conversation_id yang benar-benar ada) - supaya tidak
// overcorrect dan malah mematahkan alur normal aplikasi.
func TestConnectDB_AllowsValidForeignKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := connectDB(dbPath)
	if err != nil {
		t.Fatalf("connectDB gagal: %v", err)
	}
	defer db.Close()

	store := sqlite.NewChatStore(db)
	if err := store.Migrate(); err != nil {
		t.Fatalf("migrate gagal: %v", err)
	}

	ctx := context.Background()
	_, err = db.ExecContext(ctx,
		`INSERT INTO conversations (id, character, title) VALUES (?, ?, ?)`,
		"conv-valid", "betawi", "test",
	)
	if err != nil {
		t.Fatalf("insert conversation gagal: %v", err)
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES (?, ?, ?)`,
		"conv-valid", "user", "halo",
	)
	if err != nil {
		t.Fatalf("insert message dengan conversation_id valid harusnya berhasil, tapi gagal: %v", err)
	}
}

// TestConnectDB_ForeignKeyAppliesAcrossPooledConnections memastikan pragma
// diterapkan lewat DSN (bukan cuma sekali di koneksi pertama setelah
// sql.Open) - dites dengan memaksa connection pool boleh buka lebih dari 1
// koneksi fisik sekaligus, memverifikasi FK tetap ditegakkan di semuanya.
// Ini yang membedakan pendekatan DSN pragma vs db.Exec("PRAGMA...") sekali
// jalan (lihat penjelasan di connectDB).
func TestConnectDB_ForeignKeyAppliesAcrossPooledConnections(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := connectDB(dbPath)
	if err != nil {
		t.Fatalf("connectDB gagal: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5) // paksa pool boleh buka sampai 5 koneksi fisik

	store := sqlite.NewChatStore(db)
	if err := store.Migrate(); err != nil {
		t.Fatalf("migrate gagal: %v", err)
	}

	ctx := context.Background()
	failures := 0
	for i := 0; i < 5; i++ {
		_, err := db.ExecContext(ctx,
			`INSERT INTO messages (conversation_id, role, content) VALUES (?, ?, ?)`,
			"tidak-ada", "user", "halo",
		)
		if err != nil {
			failures++
		}
	}

	if failures != 5 {
		t.Errorf("expected semua 5 percobaan insert ditolak FK constraint, tapi cuma %d yang gagal - kemungkinan sebagian koneksi pool tidak punya FK enforcement aktif", failures)
	}
}
