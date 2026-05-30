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

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// Warna status: hijau=2xx, kuning=4xx, merah=5xx
		color := "\033[32m" // hijau
		if status >= 400 && status < 500 {
			color = "\033[33m" // kuning
		} else if status >= 500 {
			color = "\033[31m" // merah
		}
		reset := "\033[0m"

		fmt.Printf("[VoxAI] %s | %s%d%s | %8s | %s %s\n",
			time.Now().Format("2006/06/01/02 15:04:05"),
			color, status, reset,
			duration,
			method, path,
		)
	}
}
