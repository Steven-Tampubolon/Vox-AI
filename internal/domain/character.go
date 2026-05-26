package domain

// Character mendefinisikan 4 karakter AI yang tersedia
type Character string

const (
	CharacterBetawi   Character = "betawi"
	CharacterRAG      Character = "rag"
	CharacterGit      Character = "git"
	CharacterExplaine Character = "explaine"
)

func (c Character) Isvalid() bool {
	switch c {
	case CharacterBetawi, CharacterRAG, CharacterGit, CharacterExplaine:
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
	case CharacterExplaine:
		return "Profesor Analogi"
	}
	return "Unknown"
}
