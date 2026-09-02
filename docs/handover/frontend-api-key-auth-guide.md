# 🔐 Vox-AI Frontend Handover — API Key Authentication Guide

Dokumen ini ditujukan untuk tim Frontend yang perlu mengintegrasikan perubahan
autentikasi yang diimplementasikan pada security hardening Finding 1.
Baca ini sebelum deploy atau menjalankan BE versi terbaru.

---

## Latar Belakang

Sebelum security hardening, semua endpoint `/api/v1/*` bersifat **publik tanpa
autentikasi**. Siapapun yang mengetahui base URL bisa:

- Membaca dan menghapus riwayat percakapan siapapun
- Menghabiskan Gemini API quota developer secara gratis
- Melakukan operasi destruktif tanpa bisa di-trace

**Fix yang diimplementasikan**: Static API Key middleware pada semua route
`/api/v1/*`. Setiap request wajib menyertakan header `X-API-Key`.

---

## Konsep: Siapa yang Memegang Key?

```
┌─────────────────────────────────────────────────────────────┐
│                        ARSITEKTUR AUTH                      │
│                                                             │
│   User (Browser)                                            │
│        │                                                    │
│        │  klik tombol, ketik pesan                         │
│        ▼                                                    │
│   Frontend App  ──── X-API-Key: <key dari .env FE> ──────▶ │
│   (React/Next)                                  │           │
│                                                 ▼           │
│                                          Backend API         │
│                                          memvalidasi key     │
│                                          vs APP_API_KEY      │
│                                          di .env BE          │
└─────────────────────────────────────────────────────────────┘
```

**Key TIDAK berasal dari input user.** User tidak pernah tahu key ini ada.
Key dikonfigurasi sekali oleh developer/ops saat setup, dan FE membacanya
dari environment variable saat build/runtime.

---

## ⚠️ Anti-Pattern yang Harus Dihindari

**JANGAN** membuat kolom input di UI untuk memasukkan API key:

```
❌  [ Masukkan API Key Anda: _________________ ]  ← JANGAN DIBUAT
```

Alasannya:
1. Key akan **terekspos di browser** (DevTools → Network tab, localStorage)
2. Siapapun yang membuka DevTools bisa mencuri key dan bypass auth
3. Key yang bocor berarti **semua proteksi Finding 1 gugur**
4. `APP_API_KEY` bukan kredensial per-user — ini adalah shared secret
   antara FE app dan BE app, bukan antara user dan BE

---

## Setup untuk Development Lokal

### 1. Backend — tambah `APP_API_KEY` ke `.env`

```dotenv
# .env (di root project BE)
GEMINI_API_KEY=your-gemini-key-here
APP_API_KEY=voxai-dev-secret-2026       # ← tambahkan ini
PORT=8080
DB_PATH=./voxai.db
ALLOW_ORIGINS=http://localhost:3000
```

> Generate key yang aman untuk production:
> ```bash
> openssl rand -hex 32
> ```

### 2. Frontend — tambah env variable

Tambahkan key yang **sama persis** dengan `APP_API_KEY` di BE ke env FE:

```dotenv
# .env.local (di root project FE)
VITE_API_KEY=voxai-dev-secret-2026
```

> Nama variable bergantung framework: `VITE_` untuk Vite/React,
> `NEXT_PUBLIC_` untuk Next.js. Sesuaikan dengan setup FE kalian.

---

## Integrasi di Kode Frontend

### Cara yang Benar — HTTP Client Terpusat

Buat satu wrapper/client terpusat yang otomatis menyisipkan header di setiap
request. Jangan copy-paste header ini ke setiap `fetch` call.

#### Contoh Vite + React

```typescript
// src/lib/api-client.ts

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1';
const API_KEY = import.meta.env.VITE_API_KEY ?? '';

if (!API_KEY) {
  console.error('[VoxAI] VITE_API_KEY tidak ditemukan di .env — semua request ke BE akan gagal 401');
}

/**
 * Wrapper fetch yang otomatis menyisipkan X-API-Key header.
 * Gunakan fungsi ini untuk SEMUA request ke BE, bukan fetch langsung.
 */
export async function apiFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const url = `${API_BASE_URL}${path}`;

  const headers = new Headers(options.headers);
  headers.set('X-API-Key', API_KEY);

  // Jangan set Content-Type untuk multipart/form-data —
  // biarkan browser yang handle boundary secara otomatis
  if (!(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  return fetch(url, {
    ...options,
    headers,
  });
}
```

#### Contoh Penggunaan di Komponen

```typescript
// src/hooks/useChat.ts
import { apiFetch } from '@/lib/api-client';

// Text chat
export async function sendMessage(message: string, conversationId: string) {
  const res = await apiFetch('/chat/betawi', {
    method: 'POST',
    body: JSON.stringify({ message, conversation_id: conversationId }),
  });

  if (res.status === 401) {
    // Ini error konfigurasi, bukan error user — log ke monitoring
    console.error('API key tidak valid atau tidak ditemukan');
    throw new Error('Konfigurasi aplikasi bermasalah, hubungi administrator');
  }

  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error ?? 'Request gagal');
  }

  return res.json();
}

// Upload dokumen (multipart)
export async function uploadDocument(file: File, conversationId?: string) {
  const formData = new FormData();
  formData.append('file', file);
  if (conversationId) formData.append('conversation_id', conversationId);

  const res = await apiFetch('/document/upload', {
    method: 'POST',
    body: formData, // Content-Type otomatis multipart/form-data
  });

  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error ?? 'Upload gagal');
  }

  return res.json();
}
```

---

## Error Baru yang Perlu Dihandle

Setelah middleware aktif, ada dua error baru yang mungkin muncul:

| HTTP Status | `error` | Penyebab | Tindakan FE |
|---|---|---|---|
| `401` | `"API key diperlukan, sertakan header X-API-Key"` | Header `X-API-Key` tidak dikirim | Cek `apiFetch` wrapper — pastikan header terpasang |
| `401` | `"API key tidak valid"` | Key salah / tidak cocok dengan BE | Cek nilai `VITE_API_KEY` vs `APP_API_KEY` di BE |
| `500` | `"server tidak terkonfigurasi dengan benar..."` | BE jalan tanpa `APP_API_KEY` | Lapor ke BE/ops untuk cek `.env` BE |

**Panduan UX untuk 401:**
Jangan tampilkan pesan teknis ke user. Tampilkan pesan generic:
```
"Terjadi kesalahan konfigurasi. Silakan hubungi administrator."
```

---

## Setup Docker (Deploy)

### Perubahan pada `deploy/docker-compose.yml`

```yaml
services:
  backend:
    image: ghcr.io/steven-tampubolon/vox-ai:latest
    container_name: voxai-backend
    ports:
      - "8080:8080"
    environment:
      GEMINI_API_KEY: ${GEMINI_API_KEY}
      APP_API_KEY: ${APP_API_KEY}        # ← tambahkan ini
      PORT: 8080
      DB_PATH: /app/data/voxai.db
      ALLOW_ORIGINS: http://localhost:3000
    volumes:
      - voxai_data:/app/data
    restart: unless-stopped

  frontend:
    image: ghcr.io/steven-tampubolon/vox-ai-frontend:latest
    container_name: voxai-frontend
    ports:
      - "3000:80"
    environment:
      VITE_API_KEY: ${APP_API_KEY}       # ← tambahkan ini (nilai SAMA dengan BE)
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  voxai_data:
```

> **Kenapa nilainya sama?** `APP_API_KEY` adalah shared secret antara FE dan BE.
> FE mengirim key ini, BE memvalidasinya. Dengan memakai satu variable
> `APP_API_KEY` di `.env` untuk keduanya, tidak ada risiko typo atau nilai
> berbeda antara dua service.

### `deploy/.env` (rename dari `.env.example`, isi sebelum deploy)

```dotenv
# ── Gemini API ─────────────────────────────────────────────
# Dapatkan dari https://aistudio.google.com/app/apikey
GEMINI_API_KEY=your-gemini-key-here

# ── App Authentication ─────────────────────────────────────
# Shared secret antara Frontend dan Backend.
# Generate dengan: openssl rand -hex 32
# WAJIB diisi — BE tidak akan start jika kosong.
APP_API_KEY=ganti-dengan-random-string-aman
```

> **Catatan untuk Next.js:** Sesuaikan nama variable di
> `frontend.environment` pada docker-compose menjadi
> `NEXT_PUBLIC_API_KEY: ${APP_API_KEY}` dan update `api-client.ts`
> untuk menggunakan `process.env.NEXT_PUBLIC_API_KEY`.

---

## Flow Lengkap dari Sisi User

```
User membuka browser
        │
        ▼
FE App di-load (React bundle)
   └─ VITE_API_KEY sudah "terpanggang" ke dalam bundle saat build
   └─ User tidak tahu key ini ada
        │
        ▼
User mengetik pesan / upload dokumen / tekan tombol rekam
        │
        ▼
FE memanggil apiFetch()
   └─ Otomatis menambahkan header:
      X-API-Key: voxai-dev-secret-2026
        │
        ▼
BE menerima request
   └─ Middleware APIKeyAuth membandingkan header vs APP_API_KEY
   └─ Cocok     → lanjut ke handler
   └─ Tidak cocok / kosong → 401 Unauthorized
        │
        ▼
Handler memproses request (Gemini, SQLite, dll)
        │
        ▼
Response dikembalikan ke FE → ditampilkan ke user
```

---

## Checklist Integrasi

Sebelum push ke staging/production, pastikan semua item ini sudah dicentang:

- [ ] `VITE_API_KEY` (atau prefix sesuai framework) ada di `.env.local` FE
- [ ] Nilai `VITE_API_KEY` di FE **sama persis** dengan `APP_API_KEY` di BE
- [ ] Semua request ke BE menggunakan `apiFetch()` — bukan `fetch()` langsung
- [ ] Error `401` ditangani dengan pesan generic (tidak bocorkan detail teknis ke user)
- [ ] `VITE_API_KEY` **tidak** di-commit ke git (sudah masuk `.gitignore`)
- [ ] `APP_API_KEY` di production menggunakan nilai dari `openssl rand -hex 32`
- [ ] `deploy/.env` sudah diisi `APP_API_KEY` untuk deployment Docker

---

*Dokumen ini bagian dari seri security hardening VOX AI. Lihat juga:*
- [`docs/handover/frontend-voice-guide.md`](./frontend-voice-guide.md) — integrasi voice & endpoint lengkap
- [`docs/security-review-and-gap-analysis-report/`](../security-review-and-gap-analysis-report/) — laporan lengkap semua finding
