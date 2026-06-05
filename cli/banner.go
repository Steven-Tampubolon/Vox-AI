package cli

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Steven-Tampubolon/Vox-AI/config"
	"github.com/fatih/color"
)

// ─── Banner ──────────────────────────────────────────────────────────────────

func PrintBanner() {
	cyan := color.New(color.FgCyan, color.Bold)
	magenta := color.New(color.FgMagenta, color.Bold)
	white := color.New(color.FgWhite)

	// ASCII art animals
	fox := `⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⡀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣾⠙⠻⢶⣄⡀⠀⠀⠀⢀⣤⠶⠛⠛⡇⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣇⠀⠀⣙⣿⣦⣤⣴⣿⣁⠀⠀⣸⠇⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣡⣾⣿⣿⣿⣿⣿⣿⣿⣷⣌⠋⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣴⣿⣷⣄⡈⢻⣿⡟⢁⣠⣾⣿⣦⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣿⣿⣿⣿⠘⣿⠃⣿⣿⣿⣿⡏⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⠀⠈⠛⣰⠿⣆⠛⠁⠀⡀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⣿⣦⠀⠘⠛⠋⠀⣴⣿⠁⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⣾⣿⣿⣿⣿⡇⠀⠀⠀⢸⣿⣏⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⣠⣶⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠀⠀⠀⠾⢿⣿⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⣠⣿⣿⣿⣿⣿⣿⡿⠟⠋⣁⣠⣤⣤⡶⠶⠶⣤⣄⠈⠀⠀⠀⠀⠀⠀
⠀⠀⠀⢰⣿⣿⣮⣉⣉⣉⣤⣴⣶⣿⣿⣋⡥⠄⠀⠀⠀⠀⠉⢻⣄⠀⠀⠀⠀⠀
⠀⠀⠀⠸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣟⣋⣁⣤⣀⣀⣤⣤⣤⣤⣄⣿⡄⠀⠀⠀⠀
⠀⠀⠀⠀⠙⠿⣿⣿⣿⣿⣿⣿⣿⡿⠿⠛⠋⠉⠁⠀⠀⠀⠀⠈⠛⠃⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠉⠉⠉⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀`

	// ASCII art figure
	vox := `░██    ░██                              ░███    ░██████
░██    ░██                             ░██░██     ░██  
░██    ░██  ░███████  ░██    ░██      ░██  ░██    ░██  
░██    ░██ ░██    ░██  ░██  ░██      ░█████████   ░██  
 ░██  ░██  ░██    ░██   ░█████       ░██    ░██   ░██  
  ░██░██   ░██    ░██  ░██  ░██      ░██    ░██   ░██  
   ░███     ░███████  ░██    ░██ ░██ ░██    ░██ ░██████
                                                       
                                                       
                                                       `

	fmt.Println()
	cyan.Println(fox)
	cyan.Println(vox)

	magenta.Println("  ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	white.Println("   AI Chat — 4 Karakter — Clean Architecture")
	cyan.Println("   Backend  	: Go + Gin")
	cyan.Println("   LLM Model 	: 2.5 Flash Lite")
	cyan.Println("   Database 	: SQLite")
	cyan.Println("   Author   	: Steven Tampubolon")
	magenta.Println("  ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	fmt.Println()
}

// ─── System Info ─────────────────────────────────────────────────────────────

func PrintSystemInfo(cfg *config.Config) {
	green := color.New(color.FgGreen, color.Bold)
	yellow := color.New(color.FgYellow)
	white := color.New(color.FgWhite)

	hostname, _ := os.Hostname()

	green.Println("  [ SYSTEM ]")
	printRow(white, yellow, "  Hostname    ", hostname)
	printRow(white, yellow, "  OS          ", runtime.GOOS+"/"+runtime.GOARCH)
	printRow(white, yellow, "  Go Version  ", runtime.Version())
	printRow(white, yellow, "  DB Path     ", cfg.DBPath)
	printRow(white, yellow, "  CORS Origin ", cfg.AllowOrigin)
	printRow(white, yellow, "  Started at  ", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

func PrintEndpoints(port string) {
	green := color.New(color.FgGreen, color.Bold)
	// cyan := color.New(color.FgCyan)
	// yellow := color.New(color.FgYellow, color.Bold)
	// white := color.New(color.FgWhite)
	dim := color.New(color.Faint)

	base := "http://localhost:" + port

	// green.Println("  [ ENDPOINTS ]")
	// fmt.Println()

	// // Health
	// dim.Println("  ── System ──────────────────────────────────────")
	// PrintEndpoint(white, cyan, yellow, "GET ", base+"/health", "Health check")
	// fmt.Println()

	// // Characters
	// dim.Println("  ── 4 Karakter AI ───────────────────────────────")
	// PrintEndpoint(white, cyan, yellow, "POST", base+"/api/v1/chat/betawi", "Abang Betawi  — pantun & logat Betawi")
	// PrintEndpoint(white, cyan, yellow, "POST", base+"/api/v1/chat/rag", "Dokter Dokumen — Q&A dari dokumen")
	// PrintEndpoint(white, cyan, yellow, "POST", base+"/api/v1/chat/git", "Git Master     — commit message")
	// PrintEndpoint(white, cyan, yellow, "POST", base+"/api/v1/chat/explain", "Profesor Analogi — jelaskan konsep")
	// fmt.Println()

	// // Document
	// dim.Println("  ── Dokumen RAG ─────────────────────────────────")
	// PrintEndpoint(white, cyan, yellow, "POST", base+"/api/v1/document/upload", "Upload dokumen (TXT, PDF)")
	// fmt.Println()

	// // Conversations
	// dim.Println("  ── History ─────────────────────────────────────")
	// PrintEndpoint(white, cyan, yellow, "GET ", base+"/api/v1/conversations", "List semua sesi chat")
	// PrintEndpoint(white, cyan, yellow, "GET ", base+"/api/v1/conversations/:id/messages", "Pesan dalam satu sesi")
	// fmt.Println()

	green.Printf("  Server berjalan di %s\n", base)
	dim.Println("  Tekan Ctrl+C untuk berhenti")
	fmt.Println()
	dim.Println("  ────────────────────────────────────────────────")
	fmt.Println()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func printRow(label, value *color.Color, key, val string) {
	label.Printf("%s: ", key)
	value.Println(val)
}

func PrintEndpoint(white, urlColor, methodColor *color.Color, method, url, desc string) {
	methodColor.Printf("  [%s]  ", method)
	urlColor.Printf("%-55s", url)
	white.Printf("  %s\n", desc)
}
