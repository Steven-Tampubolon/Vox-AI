package domain

import "time"

// Role pengirim pesan
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message adalah satu pesan dalam percakapan
type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           Role      `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

// Conversation adalah satu sesi chat dengan satu karakter
type Conversation struct {
	ID        string    `json:"id"`
	Character Character `json:"character"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatRequest adalah request masuk dari frontend
type ChatRequest struct {
	ConversationID string    `json:"conversation_id"` // kosong = buat baru
	Character      Character `json:"character"`
	Message        string    `json:"message"`
}

// ChatResponse adalah response ke frontend
type ChatResponse struct {
	ConversationID string    `json:"conversation_id"`
	Character      Character `json:"character"`
	Reply          string    `json:"reply"`
	Error          string    `json:"error,omitempty"`
}

// Document untuk karakter RAG
type Document struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Filename       string    `json:"filename"`
	Content        string    `json:"-"` // tidak dikirim ke frontend
	ChunkCount     int       `json:"chunk_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// Chunk adalah potongan dokumen + dokumen + embedding-nya
type Chunk struct {
	ID         int64     `json:"id"`
	DocumentID string    `json:"document_id"`
	Content    string    `json:"content"`
	Embedding  []float64 `json:"-"`
}
