package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowOriginsRaw string) gin.HandlerFunc {
	// Memecah string "http://localhost:3000,http://localhost:5173" menjadi slice []string
	origins := strings.Split(allowOriginsRaw, ",")

	return func(c *gin.Context) {
		// Ambil origin dari request header frontend
		clientOrigin := c.GetHeader("Origin")

		// Cek apakah origin si client terdaftar di konfigurasi .env kita
		allowedOrigin := ""
		for _, o := range origins {
			if strings.TrimSpace(o) == clientOrigin {
				allowedOrigin = clientOrigin
				break
			}
		}

		// Jika terdaftar, pasang ke header (Gunakan nama tunggal: Access-Control-Allow-Origin)
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
		}

		// Tambahkan header pendukung lainnya
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS, PUT, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin, Accept")
		c.Header("Access-Control-Allow-Credentials", "true")

		// Handle Preflight Request (OPTIONS)
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent) // 204
			return
		}

		c.Next()
	}
}
