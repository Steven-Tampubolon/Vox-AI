# Menjalankan VOX AI via Docker Desktop (tanpa clone repo)

## Prasyarat
- Docker Desktop terinstal & sedang berjalan
- API key Gemini dari https://aistudio.google.com/app/apikey

## Langkah

1. Download `docker-compose.yml` dan `.env.example` dari sini, atau dari
   [halaman Release](https://github.com/Steven-Tampubolon/vox-ai/releases) versi terbaru
   (kedua file otomatis dilampirkan di setiap rilis).
2. Taruh di satu folder kosong, rename `.env.example` menjadi `.env`, isi `GEMINI_API_KEY`.
3. Jika image di GHCR **private**, login dulu:
```bash
   docker login ghcr.io -u <github-username-anda>
   # password: Personal Access Token dengan scope read:packages
```
   Jika image sudah **public**, langkah ini bisa dilewati.
4. Jalankan:
```bash
   docker compose up -d
```
5. Buka `http://localhost:3000` di browser.

## Cek status
```bash
docker compose ps
docker compose logs -f backend
```

## Update ke versi image terbaru
```bash
docker compose pull
docker compose up -d
```

## Matikan
```bash
docker compose down
```