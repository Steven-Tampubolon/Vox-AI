package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyHeader adalah nama header yang dipakai client untuk mengirim API key.
// Menggunakan header custom (X-API-Key) lebih aman dari query parameter karena
// header tidak tercatat di server access log, browser history, maupun referer.
const APIKeyHeader = "X-API-Key"

// APIKeyAuth middleware memvalidasi keberadaan dan kecocokan API key
// pada setiap request yang masuk ke route yang dilindungi.
//
// Finding 1 — Absence of Authentication:
// Semua endpoint /api/v1/* sebelumnya publik tanpa autentikasi.
// Middleware ini memastikan hanya client dengan API key yang valid
// (dikonfigurasi via APP_API_KEY di .env) yang bisa mengakses API,
// mencegah quota abuse Gemini dan akses tidak sah ke conversation data.
func APIKeyAuth(validKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Tolak konfigurasi yang tidak valid saat startup — mencegah
		// server berjalan dalam kondisi "auth disabled by accident"
		// akibat APP_API_KEY kosong di .env.
		if validKey == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "server tidak terkonfigurasi dengan benar, hubungi administrator",
			})
			return
		}

		key := c.GetHeader(APIKeyHeader)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API key diperlukan, sertakan header X-API-Key",
			})
			return
		}

		// Bandingkan key secara constant-time untuk mencegah timing attack.
		// Meskipun timing attack pada string comparison sangat sulit di
		// atas HTTP, ini adalah praktik terbaik untuk semua secret comparison.
		if !secureCompare(key, validKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API key tidak valid",
			})
			return
		}

		c.Next()
	}
}

// secureCompare membandingkan dua string dalam waktu konstan (constant-time)
// untuk mencegah timing attack — attacker tidak bisa mengukur seberapa
// "dekat" tebakan mereka dengan key yang benar.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
