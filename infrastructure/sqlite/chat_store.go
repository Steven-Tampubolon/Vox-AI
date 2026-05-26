package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vox-ai/internal/domain"

	_ "modernc.org/sqlite"
)

type ChatStore struct {
	db *sql.DB
}

func NewChatStore(db *sql.DB) *ChatStore {
	return &ChatStore{db: db}
}

func (s *ChatStore) Migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS conversations (
		id          TEXT PRIMARY KEY,
		character   TEXT NOT NULL,
		title       TEXT NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id TEXT NOT NULL,
		role            TEXT NOT NULL,
		content         TEXT NOT NULL,
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id)
	);`

	_, err := s.db.Exec(query)
	return err
}

// SaveConversation menyimpan sesi chat baru
func (s *ChatStore) SaveConversation(ctx context.Context, conv *domain.Conversation) error {
	query := `
	INSERT INTO conversations (id, character, title, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		conv.ID,
		string(conv.Character),
		conv.Title,
		conv.CreatedAt,
		conv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save conversation: %w", err)
	}
	return nil
}

// GetConversation mengambil satu sesi berdasarkan ID
func (s *ChatStore) GetConversation(ctx context.Context, id string) (*domain.Conversation, error) {
	query := `SELECT id, character, title, created_at, updated_at FROM conversations WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, id)

	var conv domain.Conversation
	var char string
	err := row.Scan(&conv.ID, &char, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	conv.Character = domain.Character(char)
	return &conv, nil
}

// ListConversations mengambil semua sesi chat, terbaru di atas
func (s *ChatStore) ListConversations(ctx context.Context) ([]*domain.Conversation, error) {
	query := `SELECT id, character, title, created_at, updated_at 
	          FROM conversations ORDER BY updated_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var result []*domain.Conversation
	for rows.Next() {
		var conv domain.Conversation
		var char string
		if err := rows.Scan(&conv.ID, &char, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, err
		}
		conv.Character = domain.Character(char)
		result = append(result, &conv)
	}
	return result, nil
}

// SaveMessage menyimpan satu pesan
func (s *ChatStore) SaveMessage(ctx context.Context, msg *domain.Message) error {
	query := `
	INSERT INTO messages (conversation_id, role, content, created_at)
	VALUES (?, ?, ?, ?)`

	res, err := s.db.ExecContext(ctx, query,
		msg.ConversationID,
		string(msg.Role),
		msg.Content,
		msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	id, _ := res.LastInsertId()
	msg.ID = id

	// Update updated_at conversation
	_, _ = s.db.ExecContext(ctx,
		`UPDATE conversations SET updated_at = ? WHERE id = ?`,
		time.Now(), msg.ConversationID,
	)

	return nil
}

// GetMessages mengambil semua pesan dalam satu sesi
func (s *ChatStore) GetMessages(ctx context.Context, conversationID string) ([]*domain.Message, error) {
	query := `
	SELECT id, conversation_id, role, content, created_at
	FROM messages WHERE conversation_id = ? ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var result []*domain.Message
	for rows.Next() {
		var msg domain.Message
		var role string
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msg.Role = domain.Role(role)
		result = append(result, &msg)
	}
	return result, nil
}
