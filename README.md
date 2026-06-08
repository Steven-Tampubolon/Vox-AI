# 🦊 Apa itu Vox-AI?

Vox-AI adalah backend chat AI multi-karakter. Setiap karakter punya **kepribadian, system prompt, dan kemampuan** yang berbeda:

| Karakter | Slug | Kemampuan |
|---|---|---|
| 🎭 **Abang Betawi** | `betawi` | Ngobrol santai logat Betawi, balas & buat pantun 4 baris |
| 📄 **Dokter Dokumen** | `rag` | Tanya-jawab dari dokumen yang di-upload (PDF/TXT) via RAG |
| 🌿 **Git Master** | `git` | Bantu generate commit message & jelaskan Git workflow |
| 🧑‍🏫 **Profesor Analogi** | `explain` | Jelaskan konsep rumit pakai analogi sederhana |

Semua percakapan **disimpan permanen** di SQLite, lengkap dengan history per-conversation untuk konteks multi-turn.

---

## 🏛️ Arsitektur

Mengikuti **Clean Architecture / Onion** — dependencies hanya mengarah ke dalam.

```
┌──────────────────────────────────────────────────────────────┐
│  cmd/main.go  →  bootstrap (wiring DI)                       │
└──────────────────────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────────┐
│  internal/delivery/http  (Gin handlers + middleware)         │
│  ─ handler/        : betawi, rag, git, explain, conversation │
│  ─ middleware/     : CORS, Logger, RateLimiter               │
└──────────────────────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────────┐
│  internal/usecase   (business logic murni)                   │
│  ─ betawi_usecase   ─ rag_usecase                            │
│  ─ git_usecase      ─ explain_usecase                        │
└──────────────────────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────────┐
│  internal/repository  (interface)                            │
│  ─ AIRepository  ─ ChatRepository  ─ DocumentRepository      │
└──────────────────────────────────────────────────────────────┘
            │
            ▼
┌──────────────────────────────────────────────────────────────┐
│  infrastructure/   (implementasi konkret)                    │
│  ─ gemini/   : HTTP client ke Google Generative Language API │
│  ─ sqlite/   : chat_store, document_store + migrasi          │
└──────────────────────────────────────────────────────────────┘
```

### Struktur Folder

```
.
├── cmd/main.go                    # Entry point
├── bootstrap/bootstrap.go         # Dependency wiring
├── config/config.go               # Loader .env
├── cli/banner.go                  # ASCII banner & info startup
├── internal/
│   ├── domain/                    # Entitas inti (Character, Message, Conversation, Document, Chunk)
│   ├── delivery/http/
│   │   ├── router.go              # Route definition
│   │   ├── handler/               # HTTP handlers
│   │   └── middleware/            # CORS, Logger, RateLimiter
│   ├── repository/                # Interface (port)
│   └── usecase/                   # Business logic per-karakter
├── infrastructure/
│   ├── gemini/client.go           # Gemini API client (Generate + Embed)
│   └── sqlite/                    # Persistence layer
├── go.mod / go.sum
└── .env.example
```

---

## 🚀 Quickstart

### Prasyarat

- **Go 1.25+**
- API key dari [Google AI Studio](https://aistudio.google.com/app/apikey)

### Instalasi

```bash
# 1. Clone repo
git clone https://github.com/Steven-Tampubolon/Vox-AI.git
cd Vox-AI

# 2. Salin .env.example → .env dan isi
cp .env.example .env
nano .env
```

`.env`:
```env
GEMINI_API_KEY=isi_dengan_api_key_anda
PORT=8080
DB_PATH=./voxai.db
ALLOW_ORIGIN=http://localhost:3000
```

```bash
# 3. Install dependency & jalankan
go mod tidy
go run cmd/main.go
```

Server jalan di `http://localhost:8080`. SQLite (`voxai.db`) dibuat otomatis pada startup pertama.

---

## 🌐 API Reference

Base URL: `http://localhost:8080`

### 🩺 Health

```http
GET /health
→ 200 OK
{ \"status\": \"ok\", \"service\": \"VoxAI\" }
```

### 💬 Chat dengan Karakter

Semua endpoint chat memakai **body & response yang konsisten**:

```http
POST /api/v1/chat/{betawi|rag|git|explain}
Content-Type: application/json

{
  \"conversation_id\": \"\",                // kosong = buat sesi baru
  \"message\": \"Buatkan pantun tentang ngoding\"
}
```

```json
// Response 200
{
  \"conversation_id\": \"0e3f8b6e-7f...\",
  \"character\": \"betawi\",
  \"reply\": \"Pagi-pagi minum kopi panas, …\"
}
```

> ℹ️ Field `character` di body **diabaikan** — sudah dipaksa oleh endpoint masing-masing.

### 📄 Upload Dokumen (khusus karakter RAG)

```http
POST /api/v1/document/upload
Content-Type: multipart/form-data

file: <PDF atau TXT, wajib>
conversation_id: <opsional, kosong = buat baru>
```

```json
// Response 200
{
  \"conversation_id\": \"a8c...\",
  \"document_id\":     \"1d2...\",
  \"filename\":        \"skripsi.pdf\",
  \"chunk_count\":     17,
  \"message\":         \"dokumen berhasil diindeks, silahkan mulai bertanya\"
}
```

Aturan:
- Hanya `.pdf` atau `.txt` (validasi ekstensi **dan** MIME type)
- PDF hasil scan (tanpa teks) ditolak
- Upload dokumen baru akan **menggantikan** dokumen lama di conversation yang sama

Setelah upload, lanjutkan ke `POST /api/v1/chat/rag` dengan `conversation_id` yang sama.

### 📚 History Percakapan

```http
GET /api/v1/conversations
→ { \"conversations\": [ { \"id\", \"character\", \"title\", \"created_at\", \"updated_at\" } ] }

GET /api/v1/conversations/:id/messages
→ { \"messages\": [ { \"id\", \"conversation_id\", \"role\", \"content\", \"created_at\" } ] }
```

Role di message: `user` | `assistant` | `system`.

---

## ⚙️ Konfigurasi

| Env | Default | Keterangan |
|---|---|---|
| `GEMINI_API_KEY` | — *(wajib)* | API key Google AI Studio |
| `PORT` | `8080` | Port HTTP server |
| `DB_PATH` | `./voxai.db` | Path file SQLite |
| `ALLOW_ORIGIN` | `http://localhost:3000` | CORS origin yang diizinkan |

### Middleware Aktif

- **Logger** — log request/response setiap hit
- **CORS** — origin tunggal dari `ALLOW_ORIGIN`
- **Rate Limiter** — `5 request / menit / IP` (in-memory, sliding window)
- **Recovery** — tangkap panic agar server tidak crash

---

## 🧪 Smoke Test (cURL)

```bash
# Health
curl http://localhost:8080/health

# Chat Betawi
curl -X POST http://localhost:8080/api/v1/chat/betawi \
  -H \"Content-Type: application/json\" \
  -d '{\"message\":\"Buatin pantun tentang kopi dong bang!\"}'

# Upload + RAG
curl -X POST http://localhost:8080/api/v1/document/upload \
  -F \"file=@skripsi.pdf\"
# → ambil conversation_id, lalu:
curl -X POST http://localhost:8080/api/v1/chat/rag \
  -H \"Content-Type: application/json\" \
  -d '{\"conversation_id\":\"<id>\",\"message\":\"Ringkas dokumen ini\"}'
```

---

## 🛠️ Tech Stack

- **Bahasa**: Go 1.25
- **Web Framework**: Gin v1.12
- **Database**: SQLite via `modernc.org/sqlite` (pure-Go, tanpa CGO)
- **LLM**: `gemini-2.5-flash-lite` (chat) + `gemini-embedding-001` (RAG)
- **PDF Parser**: `ledongthuc/pdf`
- **UUID**: `google/uuid`
- **Env Loader**: `joho/godotenv`

---

## 👨‍💻 Author

**Steven Tampubolon** — [@Steven-Tampubolon](https://github.com/Steven-Tampubolon)

---

## 📜 Lisensi

MIT — bebas dipakai, dimodifikasi, dan didistribusikan.
"
