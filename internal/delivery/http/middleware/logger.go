package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		// Jalankan handler selanjutnya
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// Pewarnaan Status Code ANSI
		color := "\033[32m" // hijau (2xx/3xx)
		if status >= 400 && status < 500 {
			color = "\033[33m" // kuning (4xx)
		} else if status >= 500 {
			color = "\033[31m" // merah (5xx)
		}
		reset := "\033[0m"

		// Format tanggal yang benar di Go: "2006/01/02 15:04:05"
		timestamp := time.Now().Format("2006/01/02 15:04:05")

		// Cetak log utama
		fmt.Printf("[VoxAI] %s | %s%d%s | %10s | %15s | %-7s %s\n",
			timestamp,
			color, status, reset,
			duration,
			clientIP,
			method, path,
		)

		// Jika ada error di Gin Context (misal dipicu oleh c.Error(err)), cetak detailnya
		if len(c.Errors) > 0 {
			fmt.Printf("  └─ \033[31m[Errors]: %s\033[0m\n", c.Errors.String())
		}
	}
}
