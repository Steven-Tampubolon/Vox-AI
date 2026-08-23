# Code Quality Review — Vox-AI

> **Review Date:** 23 August 2026
> **Reviewer:** Antigravity (AI Code Review)
> **Scope:** Entire project folder `/vox-ai`
> **Project Language:** Go
> **Method:** Five-axis review — Correctness, Readability, Architecture, Keamanan, Performance

---

## Executive Summary

Proyek Vox-AI adalah Go-based AI assistant backend with four characters (Betawi, RAG, Git, Explain), using Clean Architecture (domain → repository → usecase → delivery). Overall, the architectural foundation is **very good** and the folder structure is clean. However, there are several **real bugs** and **massive code duplication** that need to be resolved immediately before this project can be considered production-ready.

**Verdict: Request Changes** — There are critical bugs and structural duplication that must be fixed.

---

## Reviewed File Map

| File | Size (lines) | Status |
|---|---|---|
| `internal/domain/*.go` | ~170 lines total | ✅ Good |
| `internal/repository/*.go` | ~160 lines total | ✅ Good |
| `internal/usecase/betawi_usecase.go` | 230 lines | ⚠️ Duplication |
| `internal/usecase/git_usecase.go` | 236 lines | 🚨 Critical bug |
| `internal/usecase/explain_usecase.go` | 225 lines | ⚠️ Duplication |
| `internal/usecase/rag_usecase.go` | 383 lines | ⚠️ Attention |
| `internal/usecase/audio_usecase.go` | 105 lines | ✅ Good |
| `internal/usecase/voice_profiles.go` | 87 lines | ⚠️ Typo |
| `internal/delivery/http/handler/*.go` | ~750 lines total | ⚠️ Attention |
| `internal/delivery/http/middleware/*.go` | ~180 lines total | ✅ Good |
| `infrastructure/gemini/client.go` | 271 lines | ⚠️ Attention |
| `infrastructure/audio/gemini/synthesizer.go` | 235 lines | ✅ Good |
| `infrastructure/audio/gemini/transcriber.go` | 167 lines | ⚠️ Minor |
| `infrastructure/sqlite/*.go` | ~390 lines total | ⚠️ Attention |
| `bootstrap/bootstrap.go` | 117 lines | ✅ Good |
| `config/config.go` | 36 lines | ✅ Good |

---

## 1. Correctness

### 🚨 Critical: Incorrect Error Wrapping Bug in `git_usecase.go`

**File:** `internal/usecase/git_usecase.go` lines 166-168

```go
// PROBLEMATIC CODE
if streamErr != nil {
    return nil, fmt.Errorf("generate stream eply: %w", err) // err always nil di sini!
}
```

**Problem:** The variable being wrapped is `err` (always `nil` because it is already out of scope), not `streamErr`. As a result, when `GenerateStream` fails, the returned error is `generate stream eply: <nil>`, which is uninformative — this bug hides the actual error from Gemini or the network.

There is also a typo: `"generate stream eply"` should be `"generate stream reply"`.

**Fix:** Ganti `err` with `streamErr`.

---

### 🚨 Critical: Incorrect Error Wrapping Bug in `betawi_usecase.go`

**File:** `internal/usecase/betawi_usecase.go` lines 143-146

```go
// PROBLEMATIC CODE
if saveErr := uc.chatRepo.SaveMessage(ctx, aiMsg); saveErr != nil {
    return nil, fmt.Errorf("save ai message: %w", err) // err nil, not saveErr!
}
```

**Problem:** Same as above — `err` here is not `saveErr`. The error from `SaveMessage` will not be propagated correctly. Conversely, in `explain_usecase.go:151`, the same code correctly uses `saveErr`. This is a dangerous inconsistency.

**Fix:** Ganti `err` with `saveErr`.

---

### 🚨 Critical: Incorrect Return Value di `git_usecase.go`

**File:** `internal/usecase/git_usecase.go` lines 174-178

```go
// PROBLEMATIC CODE
return &domain.ChatResponse{
    ConversationID: conv.ID,
    Character:      domain.CharacterExplain, // Should be CharacterGit!
    Reply:          reply,
}, nil
```

**Problem:** `GitUseCase.ChatStream()` returns `Character: domain.CharacterExplain` instead of `domain.CharacterGit`. The frontend relying on this field for routing/display will receive incorrect data.

**Fix:** Ganti `domain.CharacterExplain` with `domain.CharacterGit`.

---

### Bug: Variable Name Typo in `voice_profiles.go`

**File:** `internal/usecase/voice_profiles.go` lines 10

```go
var chatracterVoiceProfiles = ...
//  ^^^^^^^^^^^ "chatracter" — typo
```

**Problem:** Nama variabel typo (`chatracterVoiceProfiles`). It does not crash because it is only used di `audio_usecase.go`, but it makes code search difficult (`grep "characterVoiceProfiles"` will not find anything).

---

### Bug: STT Model Name Typo

**File:** `infrastructure/audio/gemini/transcriber.go` lines 22

```go
sttModel = "gemini-3.5-flash-lite" // this model does not exist/is not valid
```

**Problem:** Model `gemini-3.5-flash-lite` is most likely invalid (there is no Gemini 3.5 flash lite version). Kemungkinan should be `gemini-2.5-flash-lite` (sama seperti model di `infrastructure/gemini/client.go`). If the model is invalid, all transcription requests will fail with an HTTP error.

---

### Minor Bug: Error from Update `updated_at` Ignored

**File:** `infrastructure/sqlite/chat_store.go` lines 134-137

```go
// UPDATE error is ignored with _
_, _ = s.db.ExecContext(ctx,
    `UPDATE conversations SET updated_at = ? WHERE id = ?`,
    time.Now(), msg.ConversationID,
)
```

**Problem:** The error from the `UPDATE` operation is completely ignored. Although not fatal, this can cause `updated_at` to remain unupdated without any log at all.

---

### Bug Minor: Error `io.ReadAll` Ignored di `audio_handler.go`

**File:** `internal/delivery/http/handler/audio_handler.go` lines 105

```go
data, _ := io.ReadAll(file) // error diabaikan
```

**Problem:** If reading the file fails midway, `data` may be partial/empty and the error is not handled. The handler in `Transcribe()` (lines 39-43) already handles this error correctly — `VoiceChat` should be consistent.

---

### Minor Bug: Validation `character` Missing di `VoiceChat` Handler

**File:** `internal/delivery/http/handler/audio_handler.go` lines 85-118

**Problem:** `character` is taken from `PostForm` but is not validated at the handler level. Validation is performed in the usecase, tapi error-nya returned as HTTP 500 even though it should be HTTP 400 (client error).

---

### Minor Bug: Comment Typo in `gemini/client.go`

**File:** `infrastructure/gemini/client.go` lines 122

```go
// GenerateSteram blocking sampai:  <- "Steram" should be "Stream"
```

---

## 2. Readability & Simplicity

### Massive Duplication — Pattern `buildHistory` + `getOrCreateConversation`

**This is the biggest finding in this review.**

Four use cases (`betawi`, `git`, `explain`, `rag`) each duplicate identical logic:

1. `getOrCreate[X]Conversation` — identical logic, differing only in the character constant
2. `build[X]History` — 100% identical di betawi, git, explain; almost identical di rag
3. `Chat()` — identical structure: ambil conv → simpan user msg → build history → generate → simpan AI msg → return
4. `ChatStream()` — identical structure: ambil conv → simpan user msg → build history → stream → simpan reply → return

Estimate: **~400 duplicate lines** out of a total of ~900 lines usecase.

**Example of duplication `buildHistory` that is 100% identical:**

```go
// betawi_usecase.go:198-221 == git_usecase.go:212-234 == explain_usecase.go:201-224
func (uc *BetawiUseCase) buildHistory(ctx context.Context, conversationID string) ([]gemini.Content, error) {
    // IDENTIK in all three files
}
```

**Recommended remedy:** Extract into a single `BaseChatUseCase` or shared helpers `buildHistory(ctx, conversationID, chatRepo)` and `getOrCreateConversation(ctx, req, character, chatRepo)`.

---

### Scattered Magic Numbers (Magic Numbers)

```go
// rag_usecase.go:83
chunks := splitIntoChunks(content, 500)

// rag_usecase.go:250
relevanChunks := findTopK(allChunks, queryEmbedding, 3)

// betawi_usecase.go, git_usecase.go, dll.
if len(messages) > 20 {
```

**Problem:** Angka `500`, `3`, `20` are scattered without named constants. It is difficult to change a single value when it is scattered across multiple places.

**Remedy:** Extract into named constants:

```go
const (
    maxHistoryMessages = 20
    chunkMaxChars      = 500
    topKChunks         = 3
)
```

---

### Inconsistent Usecase Names

**Problem:** Penamaan struct usecase tidak konsisten:

- `BetawiUseCase` (capitalization "Case")
- `RAGUseCase` (all uppercase)
- `GitUseCase`
- `ExplainUseCase`
- `AudioUsecase` (lowercase "c") ← different from the others

**Remedy:** Standardize on one convention, for example, use `UseCase`.

---

### Step Number Comments Are Out of Sync in `rag_usecase.go`

**File:** `internal/usecase/rag_usecase.go` lines 135 & 141

```go
// 2. embed query user + cari chunk relevan
systemWithContext, err := uc.buildContextPrompt(ctx, conv.ID, req.Message)

// 5. Ambil history + kirim ke gemini  <- jumps from 2 to 5? What about steps 3 and 4?
history, err := uc.buildRAGHistory(ctx, conv.ID)
```

The step comments jump from `// 2.` to `// 5.` — steps 3 and 4 are missing. This misleads the reader.

---

## 3. Architecture

### ✅ Good Things

- **Clean Architecture correctly applied:** `domain → repository → usecase → delivery` with clean dependency injection.
- **Interface repository limits coupling:** Usecase does not know implementation details Gemini/SQLite.
- **SSE diabstrak ke `streamChat` helper:** A single control point for all streaming.
- **`CharacterChatUsecase` interface di audio usecase:** Elegant design for the voice chat dispatcher.
- **`bootstrap.go` jelas:** The dependency graph is very clear in one place.

---

### AIRepository Leaks Infrastructure Type

**File:** `internal/repository/ai_repository.go` lines 6 & 13

```go
import "github.com/Steven-Tampubolon/Vox-AI/infrastructure/gemini"

type AIRepository interface {
    Generate(ctx context.Context, systemPrompt string, history []gemini.Content) (string, error)
    //                                                          ^^^^^^^^^^^^^^
    //                          infrastructure type leaks into the domain/repository layer
}
```

**Problem:** The `AIRepository` interface in the `internal/repository` layer exposes the `gemini.Content` type from `infrastructure/gemini`. This violates the dependency rule — inner layers should not depend on outer layers. If you want to replace Gemini with Ollama, the interface must also be changed.

**Remedy:** Define a type `ChatMessage` in the domain or repository layer itself:

```go
// di internal/domain atau internal/repository
type ChatMessage struct {
    Role    string
    Content string
}
```

---

### `ConversationHandler` Langsung ke Repository, Skip Usecase

**File:** `internal/delivery/http/handler/conversation_handler.go` lines 12-13

```go
type ConversationHandler struct {
    chatRepo repository.ChatRepository // handler accesses the repository directly, skipping the usecase
}
```

**Problem:** The handler calls `chatRepo` langsung, bypassing the usecase layer. If business validation is needed later (misalnya authorization per-user), it cannot be applied without changing the handler.

**Catatan:** For simple CRUD this is still acceptable, but it should be watched as business complexity grows.

---

### `RAGHandler` Referensi Tipe Konkret, Bukan Interface

**File:** `internal/delivery/http/handler/rag_handler.go` lines 21

```go
type RAGHandler struct {
    useCase *usecase.RAGUseCase // concrete type, not an interface
}
```

**Problem:** The handler knows the concrete usecase implementation, not interface. Ini membuat handler cannot be mocked for unit testing. Other handlers (`BetawiHandler`, `GitHandler`, dll.) likely have the same issue.

---

### Duplicated Logic `getOrCreateConversation` di Setiap Usecase

Architecturally, this is **feature logic that is copy-pasted** rather than a **single source of truth**. If there is a bug in one implementation (such as those found in `betawi_usecase.go` and `git_usecase.go`), the same bug may exist in all copies.

**Remedy:** Create package-level helpers:

```go
// internal/usecase/chat_helpers.go
func getOrCreateConversation(ctx context.Context, req *domain.ChatRequest, char domain.Character, repo repository.ChatRepository) (*domain.Conversation, error)
func buildHistory(ctx context.Context, convID string, repo repository.ChatRepository) ([]gemini.Content, error)
```

---

## 4. Security

### ✅ Good Things

- **File Upload Validation from Bytes:** `validateFile()` di `rag_handler.go` detects the MIME type from the actual bytes (rather than only the extension). This prevents malicious files with fake extensions from being uploaded.
- **API Key Is Not Logged:** Config only reads from an environment variable; it is not hardcoded.
- **Parameterized Queries:** All SQLite queries use the `?` placeholder, safe from SQL injection.

---

### IP-Based Rate Limiter — Vulnerable to Bypass

**File:** `internal/delivery/http/middleware/rate_limiter.go` lines 78

```go
if !rl.isAllowed(c.ClientIP()) {
```

**Problem:** In Gin, `c.ClientIP()` uses the `X-Forwarded-For` header when configured, which can be spoofed by the client. Without proper trusted proxy configuration, an attacker can bypass the rate limiter by spoofing the header `X-Forwarded-For`.

**Optional:** Add `router.SetTrustedProxies([]string{"127.0.0.1"})` in bootstrap for deployment behind a reverse proxy.

---

### Gemini API Errors Returned Raw to Client

**File:** Several handlers

```go
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
```

**Problem:** Errors from the Gemini API (which may contain internal details: model name, quota info, API key hints) are returned directly to the client. This can leak infrastructure information.

**Optional:** Consider sanitizing error messages for production (log detailed errors on the server and return a generic message to the client).

---

### No Authentication/Authorization

**FYI:** All API endpoints can be accessed by anyone without authentication (there is no API key, JWT, or other authentication). For a backend that calls the paid Gemini API, this is high risk if the endpoints are exposed to the public internet. If this is intentional (internal/personal backend), it is not a problem.

---

### No File Upload Size Limit

**File:** `internal/delivery/http/handler/rag_handler.go` lines 62-78

**Problem:** The uploaded file is read entirely into memory (`io.ReadAll`) without a size limit. Large files can cause OOM (Out of Memory) and crash the server.

**Remedy:** Add a limit before `io.ReadAll`:

```go
c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20) // 10MB limit
```

---

## 5. Performancence

### One-by-One Embedding — N Sequential API Calls

**File:** `internal/usecase/rag_usecase.go` lines 98-112

```go
for _, chunkText := range chunks {
    embedding, err := uc.aiRepo.Embed(ctx, chunkText) // N API call ke Gemini
    ...
}
```

**Problem:** If a document produces 50 chunk, there will be 50 sequential API calls to Gemini. This is very slow (50× network round-trip latency). The Gemini API supports batch embedding or this can be run concurrently.

**Consider:** Implement batch embedding or goroutines with a semaphore to limit concurrency.

---

### Rate Limiter Allocates a New Slice Per Request

**File:** `internal/delivery/http/middleware/rate_limiter.go` lines 59-70

```go
var valid []time.Time
for _, t := range rl.requests[ip] {
    if t.After(windowStart) {
        valid = append(valid, t)
    }
}
rl.requests[ip] = append(valid, now)
```

**Problem:** Each request allocates a new slice and copies all timestamps. For high traffic, this creates GC pressure.

**Nit:** For small scale, this is not a problem. If traffic is high, consider a circular buffer or use a proven rate limiter library.

---

### No Index on the Table `messages`

**File:** `infrastructure/sqlite/chat_store.go` lines 33-40

```sql
CREATE TABLE IF NOT EXISTS messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL,
    ...
    -- No index on conversation_id!
);
```

**Problem:** The query `GetMessages` performs a full table scan for every conversation. The more messages there are, the slower it becomes.

**Remedy:** Add an index in the migration:

```sql
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
```

---

### No Index on the Table `chunks`

**File:** `infrastructure/sqlite/document_store.go` lines 31-37

Same as above — `GetChunksByConversation` performs a JOIN without an index on `document_id`. Add:

```sql
CREATE INDEX IF NOT EXISTS idx_chunks_document_id ON chunks(document_id);
```

---

## 6. Testing

### Very Thin Coverage

**Problem:** There is only **satu test file** (`audio_usecase_test.go`) with 3 test cases. There are no tests for:

- All chat use cases (betawi, git, explain, rag)
- HTTP handlers
- Infrastructure layer (gemini client, sqlite store)
- Middleware

The existing test already uses mocks correctly and tests behavior (rather than implementation details) — this is the right pattern. Just extend it to the other modules using the same pattern.

---

### Tests Do Not Cover Error Scenarios

**File:** `internal/usecase/audio_usecase_test.go`

The `Transcribe` and `Synthesize` mocks always succeed. There are no tests for error scenarios (Transcribe fails, Chat fails, Synthesize fails, character not found in the map).

---

## Summary Checklist

```
### Correctness
- [x] Critical bug: error wrapping nil di git_usecase.go ChatStream (err vs streamErr)
- [x] Critical bug: error wrapping nil di betawi_usecase.go ChatStream (err vs saveErr)
- [x] Critical bug: character salah di git_usecase.go ChatStream return (Explain vs Git)
- [x] Bug: model STT "gemini-3.5-flash-lite" kemungkinan tidak valid
- [x] Bug: io.ReadAll error diabaikan di audio_handler.go VoiceChat
- [x] Bug: UPDATE updated_at error diabaikan di chat_store.go

### Readability
- [x] ~400 duplicate lines logika di empat usecase
- [x] Magic number tersebar (20, 500, 3) tanpa named constant
- [x] Penamaan usecase tidak konsisten (UseCase vs Usecase)
- [x] Typo variabel "chatracterVoiceProfiles"
- [x] Komentar nomor langkah tidak sinkron di rag_usecase.go

### Architecture
- [x] AIRepository bocorkan tipe gemini.Content ke layer internal
- [x] RAGHandler referensi tipe konkret *usecase.RAGUseCase, not interface
- [x] ConversationHandler akses repository langsung (skip usecase layer)

### Keamanan
- [ ] No authentication (intentional?)
- [x] Rate limiter rentan IP spoofing tanpa trusted proxy config
- [x] Error Gemini dikembalikan mentah ke client
- [x] No file upload size limit

### Performance
- [x] Embedding N API call berurutan saat index dokumen
- [x] No DB index on messages.conversation_id
- [x] No DB index on chunks.document_id

### Testing
- [x] Coverage is very thin — only 3 tests for the audio usecase
- [x] There are no tests for error scenarios
```

---

## Fix Priorities

### Required (before production)

1. **Fix bug error wrapping** di `git_usecase.go:167` — replace `err` with `streamErr`
2. **Fix bug error wrapping** di `betawi_usecase.go:145` — replace `err` with `saveErr`
3. **Fix character salah** di `git_usecase.go:176` — replace `CharacterExplain` with `CharacterGit`
4. **Verify the STT model name** `gemini-3.5-flash-lite` — replace it if invalid
5. **Add DB indexes** on `messages.conversation_id` and `chunks.document_id`
6. **Add a file upload size limit** untuk mencegah OOM

### Highly Recommended

7. **Extract duplication** `buildHistory` dan `getOrCreateConversation` ke shared helper
8. **Fix variable typo** `chatracterVoiceProfiles` → `characterVoiceProfiles`
9. **Perbaiki `io.ReadAll` error diabaikan** di `VoiceChat` handler
10. **Define named constants** untuk magic numbers (20, 500, 3)
11. **Fix the AIRepository interface** so it does not leak the type `gemini.Content`

### Optional / Consider

12. **Configure trusted proxies** di Gin untuk rate limiter yang lebih aman
13. **Sanitize error messages** sebelum dikembalikan ke client
14. **Add tests** for other use cases using the existing correct mock pattern
15. **Consider batch/concurrent embedding** for large documents in the RAG use case
16. **Standardize naming** struct usecase — choose one: `UseCase` atau `Usecase`

---

## Final Notes

This project demonstrates **a good understanding of architecture** — Clean Architecture properly applied, clean dependency injection, and use of repository interfaces that facilitate testing. SSE streaming and the voice pipeline (STT → AI → TTS) implemented elegantly.

The bugs found are not due to poor design, but are likely due to copying and pasting code between similar use cases without carefully checking each copy. That is the main reason duplication should be eliminated immediately — **a bug in one copy is very likely to exist in the other copies**.

---

*This review covers all available source code. No code changes were made.*