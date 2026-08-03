# 🎙️ Vox-AI Frontend Handover & Voice Integration Guide

Dokumen ini berisi panduan lengkap untuk tim Frontend yang akan membangun antarmuka web / mobile untuk Vox-AI Backend API.

---

## 🌐 Base URL & Konfigurasi Server

- **Development Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json` (untuk text chat & TTS) / `multipart/form-data` (untuk upload audio & file)

---

## 🏛️ Daftar Endpoints Handover

| Endpoint | Method | Payload / Content-Type | Deskripsi |
|---|---|---|---|
| `/characters` | `GET` | - | Mengambil daftar 4 karakter AI beserta slug & prompt info |
| `/chat/:slug` | `POST` | JSON: `{"message": "...", "conversation_id": "..."}` | Text chat per karakter (`betawi`, `rag`, `git`, `explain`) |
| `/document/upload` | `POST` | Multipart: `file` | Upload PDF/TXT dokumen untuk RAG chat (`/chat/rag`) |
| `/conversations` | `GET` | Query `?character=betawi` | Mengambil daftar riwayat percakapan |
| `/conversations/:id/messages` | `GET` | - | Mengambil isi pesan dalam satu sesi percakapan |
| `/audio/transcribe` | `POST` | Multipart: `file` | Convert rekaman suara user (STT) menjadi teks |
| `/audio/synthesize` | `POST` | JSON: `{"text": "...", "voice_id": "..."}` | Convert teks menjadi audio WAV binary (TTS) — response binary, bukan JSON |
| `/voice/chat` | `POST` | Multipart: `file`, `character`, `conversation_id` | End-to-end Voice Chat (rekaman suara → STT → AI → TTS) |

---

## 🎤 Panduan Integrasi Voice Chat di Frontend

### 1. Merekam Suara Pengguna (Browser MediaRecorder API)

```javascript
let mediaRecorder;
let audioChunks = [];

async function startRecording() {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });

  // ⚠️ Format audio/webm (default Chrome) belum terverifikasi didukung
  // oleh Gemini STT. Gunakan audio/ogg;codecs=opus sebagai format yang
  // lebih aman -- didukung Chrome & Firefox dan termasuk daftar format
  // resmi Gemini. Kalau mau tetap pakai webm, wajib diuji dulu.
  const mimeType = MediaRecorder.isTypeSupported('audio/ogg;codecs=opus')
    ? 'audio/ogg;codecs=opus'
    : 'audio/webm';

  mediaRecorder = new MediaRecorder(stream, { mimeType });

  mediaRecorder.ondataavailable = (event) => {
    if (event.data.size > 0) {
      audioChunks.push(event.data);
    }
  };

  mediaRecorder.start();
}

function stopRecording() {
  return new Promise((resolve) => {
    mediaRecorder.onstop = () => {
      const mimeType = mediaRecorder.mimeType;
      const audioBlob = new Blob(audioChunks, { type: mimeType });
      audioChunks = [];
      resolve({ blob: audioBlob, mimeType });
    };
    mediaRecorder.stop();
  });
}
```

### 2. Mengirimkan Rekaman ke Backend (`POST /api/v1/voice/chat`)

```javascript
async function sendVoiceMessage(audioBlob, mimeType, characterSlug = "betawi", convId = "") {
  // Tentukan ekstensi file berdasarkan mimeType supaya BE bisa
  // normalize mime type dengan benar di sisi server.
  const ext = mimeType.includes('ogg') ? 'ogg' : 'webm';
  const filename = `user_voice.${ext}`;

  const formData = new FormData();
  formData.append("file", audioBlob, filename);
  formData.append("character", characterSlug);

  // conversation_id boleh kosong untuk memulai sesi baru.
  // BE akan membuat conversation baru dan mengembalikan ID-nya di response.
  if (convId) {
    formData.append("conversation_id", convId);
  }

  const response = await fetch("http://localhost:8080/api/v1/voice/chat", {
    method: "POST",
    body: formData,
  });

  if (!response.ok) {
    const err = await response.json();
    throw new Error(err.error ?? "voice chat gagal");
  }

  const data = await response.json();

  // Jangan console.log(data) langsung -- audio_base64 bisa 5-10MB
  // dan akan membanjiri console. Log hanya field teks.
  console.log("User:", data.user_text);
  console.log("AI:", data.ai_text);
  console.log("Conversation ID:", data.conversation_id);

  return data;
}
```

### 3. Memutar Audio Respons AI

```javascript
function playAudioResponse(data) {
  if (!data.audio_base64) return;

  const audio = new Audio(`data:${data.mime_type};base64,${data.audio_base64}`);

  audio.onerror = (e) => {
    console.error("Gagal memutar audio:", e);
  };

  audio.play();
  return audio; // kembalikan instance supaya UI bisa pause/stop kalau perlu
}
```

### 4. Contoh Penggunaan Lengkap

```javascript
// State percakapan -- simpan conversation_id antar giliran bicara
let currentConvId = "";
let currentCharacter = "betawi";
let currentAudio = null;

async function handleVoiceButton() {
  // Mulai rekam
  await startRecording();

  // ... user bicara ...

  // Stop rekam
  const { blob, mimeType } = await stopRecording();

  // Hentikan audio sebelumnya kalau masih main
  if (currentAudio) {
    currentAudio.pause();
    currentAudio = null;
  }

  try {
    // Kirim ke BE -- latency normal 2-5 detik (STT + AI + TTS serial)
    const data = await sendVoiceMessage(blob, mimeType, currentCharacter, currentConvId);

    // Simpan conversation_id untuk giliran berikutnya
    currentConvId = data.conversation_id;

    // Play balasan AI
    currentAudio = playAudioResponse(data);

    return data;
  } catch (err) {
    console.error("Voice chat error:", err.message);
    throw err;
  }
}
```

---

## 📋 Shape Response & Error

### `POST /voice/chat` — Response 200

```json
{
  "user_text": "halo, apa itu sandbox?",
  "ai_text": "Sandbox itu seperti ruang bermain terbatas...",
  "audio_base64": "UklGRiQ...",
  "mime_type": "audio/wav",
  "conversation_id": "0955b295-3d7a-44b3-9c24-c640d2f1bd34"
}
```

> ⚠️ `audio_base64` bisa sangat besar (5–10MB terenkode base64 untuk respons panjang). **Jangan render langsung ke DOM atau log ke console.** Gunakan langsung di `new Audio(...)`.

### `POST /audio/synthesize` — Response 200

Response-nya **binary WAV**, bukan JSON. Gunakan `response.arrayBuffer()` atau `response.blob()`:

```javascript
const response = await fetch("http://localhost:8080/api/v1/audio/synthesize", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ text: "Halo bang!", voice_id: "Kore" }),
});

const blob = await response.blob();
const url = URL.createObjectURL(blob);
const audio = new Audio(url);
audio.play();
```

### Error Response (semua endpoint)

Semua error dikembalikan dalam format JSON berikut:

```json
{ "error": "pesan error dari server" }
```

Contoh error yang mungkin muncul:

| HTTP Status | `error` | Penyebab |
|---|---|---|
| `400` | `"file field required"` | Field `file` tidak ada di form data |
| `500` | `"karakter 'xxx' tidak dikenal..."` | Slug karakter salah (bukan betawi/rag/git/explain) |
| `500` | `"tidak ada ucapan yang terdeteksi..."` | File audio kosong atau tidak ada suara |
| `500` | `"gemini stt error: status 400..."` | Format audio tidak didukung Gemini STT |

---

## ⚡ Catatan Performa & UX

- **Latency**: Pipeline `/voice/chat` berjalan serial (STT → AI → TTS), ekspektasi latency **2–5 detik** tergantung panjang respons AI. Tambahkan loading state / skeleton yang jelas di UI.
- **Suara per karakter**: Setiap karakter sudah memiliki suara dan persona audio yang berbeda di sisi BE — FE tidak perlu kirim konfigurasi suara apapun, cukup kirim `character` slug.
- **History otomatis**: Setiap voice chat otomatis tersimpan ke riwayat percakapan dan bisa diakses lewat `GET /conversations/:id/messages` — sama seperti text chat.
- **Conversation baru**: Kosongkan atau hapus `conversation_id` untuk memulai sesi percakapan baru.

---

## 🎙️ Karakter yang Tersedia

| Slug | Persona | Contoh Penggunaan |
|---|---|---|
| `betawi` | Teman lokal Jakarta yang hangat dan kasual | Percakapan sehari-hari |
| `rag` | Spesialis dokumen yang presisi dan profesional | Tanya jawab berbasis dokumen upload |
| `git` | Developer antusias yang passionate soal clean code | Review kode, git workflow |
| `explain` | Profesor bijak yang menjelaskan lewat analogi | Belajar konsep teknis/umum |

---

## 📄 OpenAPI Specification File

File spesifikasi OpenAPI 3.0 lengkap tersedia di: [`docs/api/openapi.yaml`](../api/openapi.yaml).
Gunakan Swagger UI / Postman / Orval untuk auto-generate TypeScript SDK.