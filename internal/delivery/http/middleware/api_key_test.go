package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAPIKeyRouter(validKey string) *gin.Engine {
	r := gin.New()
	r.Use(APIKeyAuth(validKey))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	return r
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	r := setupAPIKeyRouter("secret-key")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_WrongKey(t *testing.T) {
	r := setupAPIKeyRouter("secret-key")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(APIKeyHeader, "wrong-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_CorrectKey(t *testing.T) {
	r := setupAPIKeyRouter("secret-key")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(APIKeyHeader, "secret-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — key yang benar seharusnya lolos auth", w.Code)
	}
}

func TestAPIKeyAuth_EmptyConfiguredKey(t *testing.T) {
	r := setupAPIKeyRouter("")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(APIKeyHeader, "any-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when APP_API_KEY is empty, got %d — server seharusnya menolak konfigurasi tidak valid", w.Code)
	}
}

func TestAPIKeyAuth_EmptySubmittedKey(t *testing.T) {
	r := setupAPIKeyRouter("secret-key")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(APIKeyHeader, "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty submitted key, got %d", w.Code)
	}
}

func TestSecureCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"abc", "ab", false}, // panjang berbeda
		{"", "", true},       // dua string kosong secara teknis equal —
		// middleware sudah menangani key=="" sebelum sampai ke sini
		{"secret", "secret", true},
		{"Secret", "secret", false}, // case-sensitive
	}

	for _, tc := range cases {
		got := secureCompare(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("secureCompare(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
