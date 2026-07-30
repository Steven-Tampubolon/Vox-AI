# 🎙️ Vox-AI Frontend Handover & Voice Integration Guide

Dokumen ini berisi panduan lengkap untuk tim Frontend yang akan membangun antarmuka web / mobile untuk Vox-AI Backend API.

---

## 🌐 Base URL & Konfigurasi Server

- **Development Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json` (untuk text chat & STT json) / `multipart/form-data` (untuk upload audio & file)

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
| `/audio/synthesize` | `POST` | JSON: `{"text": "..."}` | Convert teks AI menjadi audio WAV binary (TTS) |
| `/voice/chat` | `POST` | Multipart: `file`, `character`, `conversation_id` | End-to-end Voice Chat (rekaman suara → STT → AI → TTS) |

---

## 🎤 Panduan Integrasi Voice Chat di Frontend

### 1. Merekam Suara Pengguna (Browser MediaRecorder API)

```javascript
let mediaRecorder;
let audioChunks = [];

async function startRecording() {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  mediaRecorder = new MediaRecorder(stream, { mimeType: 'audio/webm' });

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
      const audioBlob = new Blob(audioChunks, { type: 'audio/webm' });
      audioChunks = [];
      resolve(audioBlob);
    };
    mediaRecorder.stop();
  });
}
```

### 2. Mengirimkan Rekaman ke Backend (`POST /api/v1/voice/chat`)

```javascript
async function sendVoiceMessage(audioBlob, characterSlug = "betawi", convId = "") {
  const formData = new FormData();
  formData.append("file", audioBlob, "user_voice.webm");
  formData.append("character", characterSlug);
  formData.append("conversation_id", convId);

  const response = await fetch("http://localhost:8080/api/v1/voice/chat", {
    method: "POST",
    body: formData,
  });

  const data = await response.json();
  console.log("User Transcribed Text:", data.user_text);
  console.log("AI Response Text:", data.ai_text);

  // Play audio response dari base64
  if (data.audio_base64) {
    const audio = new Audio(`data:${data.mime_type};base64,${data.audio_base64}`);
    audio.play();
  }

  return data;
}
```

---

## 📄 OpenAPI Specification File
File spesifikasi OpenAPI 3.0 lengkap tersedia di: [`docs/api/openapi.yaml`](../api/openapi.yaml).
Gunakan Swagger UI / Postman / Orval untuk auto-generate TypeScript SDK.
