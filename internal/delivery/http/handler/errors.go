package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// respondInternalError meng-log detail error asli di server (bisa berisi
// info sensitif: nama model Gemini, quota info, detail query DB, dsb) dan
// mengembalikan pesan generik ke client, supaya detail infrastruktur tidak
// bocor. context dipakai sebagai prefix log agar mudah di-grep.
func respondInternalError(c *gin.Context, context string, err error) {
	log.Printf("[%s] internal error: %v", context, err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "terjadi kesalahan pada server, coba lagi nanti",
	})
}
