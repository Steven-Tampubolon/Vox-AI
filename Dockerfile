# =========================================
# Stage 1 - Build binary Go
# =========================================
FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# modernc.org/sqlite itu pure-Go (tanpa CGO), jadi binary bisa fully static
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /voxai ./cmd/main.go

# =========================================
# Stage 2 - Runtime image (kecil & aman)
# =========================================
FROM alpine:3.20 AS runner
WORKDIR /app

# ca-certificates wajib karena app manggil Gemini API via HTTPS
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 appuser

COPY --from=builder /voxai /app/voxai

RUN mkdir -p /app/data && chown -R appuser /app/data
USER appuser

ENV PORT=8080 \
    DB_PATH=/app/data/voxai.db

EXPOSE 8080
VOLUME ["/app/data"]

ENTRYPOINT ["/app/voxai"]