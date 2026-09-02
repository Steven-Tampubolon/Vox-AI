package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/handler"
	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(
	betawi *handler.BetawiHandler,
	rag *handler.RAGHandler,
	git *handler.GitHandler,
	explain *handler.ExplainHandler,
	conv *handler.ConversationHandler,
	character *handler.CharacterHandler,
	audio *handler.AudioHandler,
	allowOrigin string,
	apiKey string,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if err := router.SetTrustedProxies(nil); err != nil {
		panic(fmt.Sprintf("gagal set trusted proxies: %v", err))
	}

	// ── Global middleware ──────────────────────────────────
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(allowOrigin))
	router.Use(middleware.RateLimiter(10, time.Minute))
	router.Use(gin.Recovery()) // tangkap panic agar server tidak crash

	// ── Health check ───────────────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "VoxAI"})
	})

	// ── Swagger UI Documentation ───────────────────────────
	// 1. Serve file openapi.yaml secara statis agar bisa dibaca Swagger UI
	router.StaticFile("/openapi.yaml", "./docs/api/openapi.yaml")

	// 2. Setup Swagger UI yang mengarah ke file openapi.yaml
	swaggerURL := ginSwagger.URL("/openapi.yaml")
	router.GET("/api-docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))

	// ── API v1 ─────────────────────────────────────────────
	api := router.Group("/api/v1")
	api.Use(middleware.APIKeyAuth(apiKey))
	{
		// Character - list 4 karakter + metadata
		api.GET("/characters", character.List)
		// Chat - 4 karakter
		chat := api.Group("/chat")
		{
			chat.POST("/betawi", betawi.Chat)
			chat.POST("/rag", rag.Chat)
			chat.POST("/git", git.Chat)
			chat.POST("/explain", explain.Chat)
		}

		// Dokumen - upload untuk karakter RAG
		api.POST("/document/upload", rag.UploadDocument)

		// Conversations - history + CRUD - untuk sidebar frontend
		convGroup := api.Group("/conversations")
		{
			convGroup.GET("", conv.List)                     // list conversation support filter ?character
			convGroup.DELETE("/:id", conv.Delete)            // hapus sesi
			convGroup.PATCH("/:id", conv.UpdateTitle)        // rename judul
			convGroup.GET("/:id/messages", conv.GetMessages) // pesan dalam sesi support filter ?character
		}

		// Audio & Voice
		audioGroup := api.Group("/audio")
		{
			audioGroup.POST("/transcribe", audio.Transcribe)
			audioGroup.POST("/synthesize", audio.Synthesize)
		}
		api.POST("/voice/chat", audio.VoiceChat)
	}

	return router
}
