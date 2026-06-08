package bootstrap

import (
	"database/sql"
	"log"

	"github.com/Steven-Tampubolon/Vox-AI/cli"
	"github.com/Steven-Tampubolon/Vox-AI/config"
	geminipkg "github.com/Steven-Tampubolon/Vox-AI/infrastructure/gemini"
	"github.com/Steven-Tampubolon/Vox-AI/infrastructure/sqlite"
	httpdelivery "github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http"
	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/handler"
	"github.com/Steven-Tampubolon/Vox-AI/internal/repository"
	"github.com/Steven-Tampubolon/Vox-AI/internal/usecase"
)

func AppInit() {
	cli.PrintBanner()

	// 1.Load config dari .env
	cfg := config.Load()
	if cfg.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY tidak ditemukan di .env")
	}

	// 2. Buka koneksi ke database
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatal("gagal  buka database:", err)
	}
	defer db.Close()

	// pastikan koneksi berhasil
	if err := db.Ping(); err != nil {
		log.Fatal("gagal ping database:", err)
	}

	// 3. Buat stores dan jalankan migrasi tabel
	chatStore := sqlite.NewChatStore(db)
	if err := chatStore.Migrate(); err != nil {
		log.Fatal("migrasi chat gagal:", err)
	}

	docStore := sqlite.NewDocumentStore(db)
	if err := docStore.Migrate(); err != nil {
		log.Fatal("migrasi document gagal:", err)
	}

	// 4. Buat repositories
	geminiClient := geminipkg.NewClient(cfg.GeminiAPIKey)
	aiRepo := repository.NewGeminiAIRepository(geminiClient)
	chatRepo := repository.NewSQLiteChatRepository(chatStore)
	docRepo := repository.NewSQLiteDocumentRepository(docStore)

	// 5. Buat usecases
	betawiUC := usecase.NewBetawiUseCase(aiRepo, chatRepo)
	ragUC := usecase.NewRAGUseCase(aiRepo, chatRepo, docRepo)
	gitUC := usecase.NewGitUseCase(aiRepo, chatRepo)
	explainUC := usecase.NewExplainUseCase(aiRepo, chatRepo)

	// 6. Buat handlers
	betawiH := handler.NewBetawiHandler(betawiUC)
	ragH := handler.NewRAGHandler(ragUC)
	gitH := handler.NewGitHandler(gitUC)
	explainH := handler.NewExplainHandler(explainUC)
	convH := handler.NewConversationHandler(chatRepo)
	characterH := handler.NewCharacterHandler()

	// 7. Setup router
	router := httpdelivery.NewRouter(
		betawiH, ragH, gitH, explainH, convH,
		characterH,
		cfg.AllowOrigin,
	)

	cli.PrintSystemInfo(cfg)
	cli.PrintEndpoints(cfg.Port)

	// 8. Jalankan server
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("server error:", err)
	}

}
