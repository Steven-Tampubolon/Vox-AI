package http

import (
	"net/http"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/handler"
	"github.com/Steven-Tampubolon/Vox-AI/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	betawi *handler.BetawiHandler,
	rag *handler.RAGHandler,
	git *handler.GitHandler,
	explain *handler.ExplainHandler,
	conv *handler.ConversationHandler,
	allowOrigin string,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// ── Global middleware ──────────────────────────────────
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(allowOrigin))
	router.Use(middleware.RateLimiter(30, time.Minute))
	router.Use(gin.Recovery()) // tangkap panic agar server tidak crash

	// ── Health check ───────────────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "VoxAI"})
	})

	// ── API v1 ─────────────────────────────────────────────
	api := router.Group("/api/v1")
	{
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

		// Conversation history - untuk sidebar frontend
		api.GET("/conversations", conv.List)
		api.GET("/conversations/:id/messages", conv.GetMessages)
	}

	return router
}
