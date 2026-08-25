# Security Review & Gap Analysis Report — Vox-AI

> **Review Date:** August 25, 2026  
> **Reviewer:** Antigravity (AI Security Auditor)  
> **Scope:** Backend Repository `/vox-ai`  
> **Programming Language:** Go (Golang)  
> **Implementation Status:** **REVIEW ONLY** (No source code was modified per user instructions)

---

## 1. Executive Summary

This security review was conducted on the Vox-AI backend repository, an AI assistant service featuring 4 unique characters (Betawi, RAG, Git, Explain). The backend is built using the Go programming language following Clean Architecture principles.

In general, the architectural foundation of Vox-AI is well-designed, structured, and clean. Several baseline security controls are functioning properly, such as file size restriction on PDF uploads (10 MB), dynamic Cross-Origin Resource Sharing (CORS) separation from configuration files, internal error handling to prevent leaking infrastructure details via the `respondInternalError` function, and migrating the Gemini API Key from URL query parameters to HTTP Headers.

However, several **critical vulnerabilities** and **design gaps** were identified that must be remediated immediately before the application can be declared production-ready.

**Key Recommendation: Immediate Action Required (Request Changes)**  
The application currently lacks authentication and authorization mechanisms, is vulnerable to Denial of Service (DoS) attacks on audio uploads, and leaves orphaned records in the database when conversation sessions are deleted.

---

## 2. Threat Modeling (STRIDE)

Based on the current Vox-AI backend architecture, the trust boundary lies at the interaction between the client (frontend) and the API endpoints at `/api/v1/*`. Any external data (audio files, document files, chat messages, conversation UUIDs) crossing this boundary must be treated as a potential threat (*hostile input*).

Below is the threat mapping using the **STRIDE** methodology:

| Threat | Abuse Case Description | Mitigation Status |
| :--- | :--- | :--- |
| **S**poofing | An attacker spoofs a user's identity or hijacks another user's conversation session ID due to the absence of authentication tokens. | **UNMITIGATED** (All API routes are public) |
| **T**ampering | An attacker manipulates conversation titles (`PATCH /conversations/:id`) or deletes conversations (`DELETE /conversations/:id`) belonging to other users. | **UNMITIGATED** (No ownership or authorization checks) |
| **R**epudiation | A user performs destructive actions or exhausts API quotas, but the system lacks user-based audit logs to trace the attacker. | **UNMITIGATED** (Only global IP logs exist) |
| **I**nformation disclosure | Sensitive data from uploaded documents (`.pdf` / `.txt`) is leaked due to guessable or exposed session UUIDs, and leftover document chunks remain in the database after conversations are deleted. | **PARTIALLY MITIGATED** (Internal errors are hidden, but RAG data leaks / is not deleted) |
| **D**enial of service | An attacker uploads a massive audio file (e.g., 2GB) to the transcription endpoint to exhaust server memory (OOM), or uploads a hundred-page PDF to drain Gemini API quotas and strain CPU vector search. | **UNMITIGATED** (No size limit on audio, no page limit on PDF) |
| **E**levation of privilege | An attacker accesses admin functionality or freely manipulates other users' conversations without valid access rights. | **UNMITIGATED** (No role-based access control) |

---

## 3. Key Security Findings

### 🚨 Finding 1: Absence of Authentication and Authorization (Broken Access Control)
* **OWASP Category:** **A01:2021-Broken Access Control**
* **Related File:** [router.go](file:///home/steven/projects/vox-ai/internal/delivery/http/router.go#L51-L84)
* **Description:** 
  All API routes under the `/api/v1` group are public and require no authentication (such as JWT or session tokens). Additionally, conversation manipulation endpoints such as `DELETE /api/v1/conversations/:id` and `PATCH /api/v1/conversations/:id` rely solely on the `:id` sent by the client without verifying whether the client is the legitimate owner of that conversation.
* **Impact:** 
  External attackers can randomly or systematically query, modify, or delete conversations belonging to other users if they obtain or guess conversation UUIDs. Attackers can also misuse the AI assistant and audio APIs free of charge, incurring costs and consuming the application developer's Gemini API quota.

---

### 🚨 Finding 2: OOM / Denial of Service (DoS) Vulnerability in Audio Uploads
* **OWASP Category:** **A05:2021-Security Misconfiguration (Denial of Service)**
* **Related Files:** [audio_handler.go](file:///home/steven/projects/vox-ai/internal/delivery/http/handler/audio_handler.go#L22-L56) (`Transcribe` function) and [audio_handler.go](file:///home/steven/projects/vox-ai/internal/delivery/http/handler/audio_handler.go#L85-L122) (`VoiceChat` function)
* **Description:** 
  Unlike `UploadDocument` in `rag_handler.go` which restricts the request body to a maximum of 10MB using `http.MaxBytesReader`, the `/audio/transcribe` and `/voice/chat` endpoints enforce neither request body size limits nor audio file size limits at all.
* **Impact:** 
  An attacker can transmit extremely large audio files (e.g., hundreds of megabytes or several gigabytes). Because the application directly invokes `io.ReadAll(file)` to load the entire audio payload into RAM, the server can easily run Out of Memory (OOM) and crash, causing a Denial of Service for all users.

---

### ⚠️ Finding 3: Data Leakage & Orphaned Records Upon Conversation Deletion
* **Category:** **Data Privacy & Storage Exhaustion**
* **Related File:** [chat_store.go](file:///home/steven/projects/vox-ai/infrastructure/sqlite/chat_store.go#L173-L191) (`DeleteConversation` function)
* **Description:** 
  When the `DELETE /api/v1/conversations/:id` endpoint is invoked, the handler triggers the deletion of conversation metadata and messages only from the database:
  ```go
  _, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE conversation_id = ?`, id)
  ...
  _, err = s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
  ```
  However, document metadata in the `documents` table and text fragments with vector embeddings in the `chunks` table associated with that conversation **are not deleted**.
* **Impact:** 
  Sensitive data extracted from users' PDF/TXT documents remains stored in the database indefinitely even after users delete their conversation sessions. This violates data privacy principles (*Data Erasure / Right to be Forgotten*) and causes the database file size (`voxai.db`) to grow continuously without clear limits (*Storage Exhaustion*).

---

### ⚠️ Finding 4: Missing Native SQLite Foreign Key Enforcement Activation
* **Category:** **Database Integrity Misconfiguration**
* **Related File:** [bootstrap.go](file:///home/steven/projects/vox-ai/bootstrap/bootstrap.go#L27-L39) (`connectDB` function)
* **Description:** 
  Although table schemas define foreign key relationships (e.g., `FOREIGN KEY (conversation_id) REFERENCES conversations(id)`), the SQLite engine **does not enforce** foreign key rules by default unless explicitly activated via PRAGMA commands or DSN configuration. In the `connectDB` function, the SQLite database is opened using only `sql.Open("sqlite", dbPath)` without additional settings.
* **Impact:** 
  Inter-table relationship constraints are not natively enforced at the database level. If manual data deletions relying on `ON DELETE CASCADE` in SQLite are performed in the future, they will not execute, risking relational data integrity corruption.

---

### ⚠️ Finding 5: Quota & CPU Exploitation via Unbounded PDF RAG (Algorithmic Complexity)
* **OWASP LLM Category:** **LLM10:2025-Unbounded Consumption**
* **Related File:** [rag_usecase.go](file:///home/steven/projects/vox-ai/internal/usecase/rag_usecase.go#L63-L120) (`IndexDocument` & `embedChunksConcurrently` functions)
* **Description:** 
  While PDF file sizes are capped at 10MB, there are no checks on page counts or the volume of text/characters extracted from the document. A 10MB PDF consisting purely of uncompressed text can yield thousands of pages and tens of thousands of small chunks.
* **Impact:** 
  1. **Quota Exhaustion**: Parallel embedding processing for thousands of chunks will instantly consume the Gemini API rate limit, triggering HTTP 429 errors across the application.
  2. **CPU & Latency Spike**: All chunks belonging to a conversation session are loaded into RAM and evaluated for cosine similarity using pure Go code sequentially/linearly for every message sent by the user. Processing tens of thousands of chunks on each chat input severely strains server CPU resources and causes assistant response times to degrade significantly (timeout).

---

### 🔍 Finding 6: Internal Detail Information Leakage via PDF Error Responses
* **OWASP Category:** **A03:2021-Injection (Information Disclosure)**
* **Related File:** [rag_handler.go](file:///home/steven/projects/vox-ai/internal/delivery/http/handler/rag_handler.go#L105-L109)
* **Description:** 
  When PDF text extraction fails, the handler returns the raw error string from the parser library directly to the client:
  ```go
  textContent, err = extractPDFText(rawBytes)
  if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to extract text from PDF: %s", err.Error())})
      return
  }
  ```
* **Impact:** 
  Attackers can leverage detailed error messages to gain insight into library versions, internal server structure, or underlying dependencies (`ledongthuc/pdf`), facilitating the search for specific library exploits.

---

## 4. Gap Analysis Against Security Standards

### A. OWASP Top 10 (2021) Compliance
1. **A01: Broken Access Control**: **MAJOR GAP**. Lacks token authentication and session ownership validation.
2. **A02: Cryptographic Failures**: Met. API Keys are sent via POST in HTTPS headers rather than URL query parameters. However, sensitive document data stored in the database is unencrypted.
3. **A03: Injection**: Partially met in the database layer (all SQLite queries utilize parameterized queries). However, information disclosure exists through detailed parser errors.
4. **A04: Insecure Design**: **GAP**. Application design assumes a local single-tenant environment while exposing multi-tenant HTTP API endpoints that are vulnerable to abuse if deployed to a public server.
5. **A05: Security Misconfiguration**: **GAP**. Absence of payload size restrictions on audio HTTP handlers.
6. **A06: Vulnerable and Outdated Components**: Fair. Dependencies are clearly declared in `go.mod`. However, regular execution of `go list -m -u all` or dependency audits is recommended.
7. **A07: Identification and Authentication Failures**: **MAJOR GAP**. No user identification system present.
8. **A08: Software and Data Integrity Failures**: Fair. However, SQLite is not configured to enforce relational integrity via foreign keys natively.
9. **A09: Security Logging and Monitoring Failures**: Partially met. Application logs IP addresses and request durations, but fails to capture unique user identities due to the lack of an authentication system.
10. **A10: Server-Side Request Forgery (SSRF)**: Secure. Application does not permit dynamic external URL fetching based on user input.

### B. OWASP Top 10 for LLM Applications (2025) Compliance
1. **LLM01: Prompt Injection**: **GAP**. Input from uploaded PDF documents is injected directly into system prompts without strong structural delimiters. Malicious documents containing instructions such as "Ignore previous instructions..." can potentially hijack AI behavior.
2. **LLM02: Sensitive Data Disclosure**: Partially met. Terminal logs do not record raw message content. However, conversation history in SQLite is stored in plaintext.
3. **LLM05: Improper Output Handling**: Partially met on the backend (transmits plain text only). However, it is **Critical** for the frontend assistant to secure text rendering (enforce HTML sanitization/escaping) to prevent Stored XSS attacks if the AI is coerced into generating malicious HTML/JS code via uploaded PDF documents.
4. **LLM06: Excessive Agency**: Met. Models are not equipped with destructive external tools (such as automated shell execution or server file deletion).
5. **LLM10: Unbounded Consumption**: **MAJOR GAP**. PDF documents are not restricted by page count or chunk limits, risking Gemini API quota exhaustion and heavy server CPU load.

---

## 5. Hardening Recommendations and Remediation Steps

The following mitigation measures are recommended to remediate the identified security gaps without modifying core business logic:

### A. Short-Term Recommendations (High Priority)

1. **Restrict Payload Size on Audio Handlers**
   Apply `http.MaxBytesReader` in `Transcribe` and `VoiceChat` within [audio_handler.go](file:///home/steven/projects/vox-ai/internal/delivery/http/handler/audio_handler.go) to limit incoming HTTP body sizes to a maximum of 10MB, consistent with the implementation in `UploadDocument`.
   
2. **Enable Native Foreign Keys in SQLite**
   Modify the `connectDB` function in [bootstrap.go](file:///home/steven/projects/vox-ai/bootstrap/bootstrap.go#L27) to execute the following pragma command immediately after establishing the database connection:
   ```go
   _, err = db.Exec("PRAGMA foreign_keys = ON;")
   ```
   Alternatively, if using modern CGO/non-CGO drivers, append the pragma query parameter to the database DSN in the `.env` file or configuration, e.g., `./voxai.db?_pragma=foreign_keys(1)`.

3. **Delete Associated Documents and Chunks on Conversation Deletion**
   Update the `DeleteConversation` function in [chat_store.go](file:///home/steven/projects/vox-ai/infrastructure/sqlite/chat_store.go#L173). Integrate calls to delete documents and chunks by invoking corresponding document deletion logic from the document repository, preventing orphaned sensitive data in the database.

4. **Limit PDF Page Count and Extracted Character Length**
   Within the `extractPDFText` function in [rag_handler.go](file:///home/steven/projects/vox-ai/internal/delivery/http/handler/rag_handler.go#L178), impose limits on the maximum number of PDF pages processed (e.g., max 50 pages) and total extracted character length (e.g., max 100,000 characters) to prevent excessive consumption of Gemini API quotas and local CPU resources.

5. **Sanitize PDF Extraction Error Messages for Clients**
   Modify the error response in [rag_handler.go](file:///home/steven/projects/vox-ai/internal/delivery/http/handler/rag_handler.go#L107) to avoid returning raw `err.Error()` details to the frontend. Display a generic, user-friendly error message, while logging detailed execution errors internally to server logs.

### B. Long-Term Recommendations (Architectural)

1. **Implement JWT / OIDC Authentication**
   Introduce global authentication middleware for sensitive and state-modifying API routes. Users should authenticate first and supply an `Authorization: Bearer <token>` header with every request.
   
2. **Enforce Ownership Verification**
   Modify the `conversations` table schema to include a `user_id` column. For every request targeting `/conversations/:id`, verify that the `user_id` extracted from the JWT token matches the `user_id` of the conversation owner in the database.

3. **Use Structural Delimiters in RAG Prompts**
   To mitigate Prompt Injection risks, wrap document chunks injected into the system prompt using explicit XML/Markdown tags, for example:
   ```markdown
   Below is the relevant document context:
   <document_content>
   {{.DocText}}
   </document_content>
   ```
   Include explicit system prompt instructions directing the model to ignore operational commands contained within `<document_content>` tags.