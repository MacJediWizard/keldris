package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimitMiddleware_UnderLimit(t *testing.T) {
	r := gin.New()
	r.Use(BodyLimitMiddleware(1024))
	r.POST("/test", func(c *gin.Context) {
		buf := make([]byte, 512)
		_, err := c.Request.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := strings.NewReader(strings.Repeat("a", 512))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestBodyLimitMiddleware_OverLimit(t *testing.T) {
	r := gin.New()
	r.Use(BodyLimitMiddleware(1024))
	r.POST("/test", func(c *gin.Context) {
		buf := make([]byte, 2048)
		_, err := c.Request.Body.Read(buf)
		if err != nil {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := strings.NewReader(strings.Repeat("a", 2048))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", w.Code)
	}
}

func TestBodyLimitMiddleware_ExactLimit(t *testing.T) {
	r := gin.New()
	r.Use(BodyLimitMiddleware(1024))
	r.POST("/test", func(c *gin.Context) {
		buf := make([]byte, 1025)
		_, err := c.Request.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := strings.NewReader(strings.Repeat("a", 1024))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestBodyLimitMiddleware_NilBody(t *testing.T) {
	r := gin.New()
	r.Use(BodyLimitMiddleware(1024))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
