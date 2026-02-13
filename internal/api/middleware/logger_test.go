package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func TestRequestLogger(t *testing.T) {
	mw := RequestLogger(zerolog.Nop())

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail"})
	})
	r.GET("/bad-request", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad"})
	})

	t.Run("successful request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?q=hello", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("server error request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/error", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("client error request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/bad-request", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}

func TestRedactQueryString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys map[string]string // expected key -> value after redaction
	}{
		{
			name:  "empty query",
			input: "",
		},
		{
			name:     "no sensitive params",
			input:    "page=1&limit=10",
			wantKeys: map[string]string{"page": "1", "limit": "10"},
		},
		{
			name:     "single sensitive param",
			input:    "token=abc123",
			wantKeys: map[string]string{"token": "[REDACTED]"},
		},
		{
			name:     "multiple sensitive params",
			input:    "token=abc&secret=xyz&password=hunter2",
			wantKeys: map[string]string{"token": "[REDACTED]", "secret": "[REDACTED]", "password": "[REDACTED]"},
		},
		{
			name:     "mixed safe and sensitive params",
			input:    "page=1&token=abc&limit=10&code=authcode",
			wantKeys: map[string]string{"page": "1", "token": "[REDACTED]", "limit": "10", "code": "[REDACTED]"},
		},
		{
			name:     "all sensitive param names",
			input:    "token=a&key=b&secret=c&password=d&code=e&state=f",
			wantKeys: map[string]string{"token": "[REDACTED]", "key": "[REDACTED]", "secret": "[REDACTED]", "password": "[REDACTED]", "code": "[REDACTED]", "state": "[REDACTED]"},
		},
		{
			name:     "case insensitive param names",
			input:    "Token=abc&KEY=xyz&Secret=123",
			wantKeys: map[string]string{"Token": "[REDACTED]", "KEY": "[REDACTED]", "Secret": "[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactQueryString(tt.input)

			if tt.input == "" {
				if result != "" {
					t.Fatalf("expected empty string, got %q", result)
				}
				return
			}

			parsed, err := url.ParseQuery(result)
			if err != nil {
				t.Fatalf("failed to parse redacted query: %v", err)
			}

			for key, wantVal := range tt.wantKeys {
				got := parsed.Get(key)
				if got != wantVal {
					t.Errorf("param %q: expected %q, got %q", key, wantVal, got)
				}
			}

			if len(parsed) != len(tt.wantKeys) {
				t.Errorf("expected %d params, got %d", len(tt.wantKeys), len(parsed))
			}
		})
	}
}
