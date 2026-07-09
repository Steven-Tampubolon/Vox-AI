package domain

// Character mendefinisikan 4 karakter AI yang tersedia
type Character string

const (
	CharacterBetawi  Character = "betawi"
	CharacterRAG     Character = "rag"
	CharacterGit     Character = "git"
	CharacterExplain Character = "explain"
)

func (c Character) IsValid() bool {
	switch c {
	case CharacterBetawi, CharacterRAG, CharacterGit, CharacterExplain:
		return true
	}
	return false
}

func (c Character) DisplayName() string {
	switch c {
	case CharacterBetawi:
		return "Abang Betawi"
	case CharacterRAG:
		return "Dokter Dokumen"
	case CharacterGit:
		return "Git Master"
	case CharacterExplain:
		return "Profesor Analogi"
	}
	return "Unknown"
}

// CharacterInfo adalah data lengkap karakter untuk response API
type CharacterInfo struct {
	ID          Character `json:"id"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Capability  []string  `json:"capability"`
}

// GetAllCharacters mengembalikan daftar semua karakter beserta metadata-nya
func GetAllCharacters() []CharacterInfo {
	return []CharacterInfo{
		{
			ID:          CharacterBetawi,
			DisplayName: CharacterBetawi.DisplayName(),
			Description: "Karakter humoris yang fasih berpantun dengan logat betawi",
			Capability:  []string{"balas pantun", "buat pantun", "ngobrol santai"},
		},
		{
			ID:          CharacterRAG,
			DisplayName: CharacterRAG.DisplayName(),
			Description: "Analisis dokumen yang teliti, menjawab hanya dari isi dokumen",
			Capability:  []string{"tanya jawab dokumen", "ringkasan", "kesimpulan"},
		},
		{
			ID:          CharacterGit,
			DisplayName: CharacterGit.DisplayName(),
			Description: "Ahli version control yang generate commit message profesional",
			Capability:  []string{"commit message", "conventional commits", "git tips"},
		},
		{
			ID:          CharacterExplain,
			DisplayName: CharacterExplain.DisplayName(),
			Description: "Profesor yang menjelaskan konsep kompleks dengan analogi sederhana",
			Capability:  []string{"analogi", "ELI5", "contoh nyata"},
		},
	}
}
