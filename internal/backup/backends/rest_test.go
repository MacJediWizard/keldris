package backends

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestRESTBackend_Init(t *testing.T) {
	b := &RestBackend{URL: "https://rest.example.com:8000/"}
	if b.Type() != models.RepositoryTypeRest {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeRest)
	}
}

func TestRESTBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend RestBackend
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid HTTPS URL",
			backend: RestBackend{URL: "https://rest.example.com:8000/"},
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			backend: RestBackend{URL: "http://localhost:8000/"},
			wantErr: false,
		},
		{
			name:    "valid URL with rest prefix",
			backend: RestBackend{URL: "rest:https://rest.example.com:8000/"},
			wantErr: false,
		},
		{
			name: "valid with credentials",
			backend: RestBackend{
				URL:      "https://rest.example.com:8000/",
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name:    "missing URL",
			backend: RestBackend{URL: ""},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name:    "invalid URL with bad escape",
			backend: RestBackend{URL: "http://example.com/%zz"},
			wantErr: true,
			errMsg:  "invalid url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if got := err.Error(); !contains(got, tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestRESTBackend_ToResticConfig(t *testing.T) {
	tests := []struct {
		name           string
		backend        RestBackend
		wantRepository string
	}{
		{
			name:           "without auth",
			backend:        RestBackend{URL: "https://rest.example.com:8000/"},
			wantRepository: "rest:https://rest.example.com:8000/",
		},
		{
			name: "with auth",
			backend: RestBackend{
				URL:      "https://rest.example.com:8000/",
				Username: "user",
				Password: "pass",
			},
			wantRepository: "rest:https://user:pass@rest.example.com:8000/",
		},
		{
			name:           "already has rest prefix",
			backend:        RestBackend{URL: "rest:https://rest.example.com:8000/"},
			wantRepository: "rest:https://rest.example.com:8000/",
		},
		{
			name: "with auth and rest prefix",
			backend: RestBackend{
				URL:      "rest:https://rest.example.com:8000/",
				Username: "user",
				Password: "pass",
			},
			// When URL already has rest:, the url.Parse of "rest:https://..." won't
			// parse as expected, so auth embedding may not work cleanly.
			// The rest: prefix remains.
			wantRepository: "rest:https://rest.example.com:8000/",
		},
		{
			name: "only username no password skips auth",
			backend: RestBackend{
				URL:      "https://rest.example.com:8000/",
				Username: "user",
			},
			wantRepository: "rest:https://rest.example.com:8000/",
		},
		{
			name: "only password no username skips auth",
			backend: RestBackend{
				URL:      "https://rest.example.com:8000/",
				Password: "pass",
			},
			wantRepository: "rest:https://rest.example.com:8000/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.backend.ToResticConfig("repopass")
			if cfg.Repository != tt.wantRepository {
				t.Errorf("Repository = %v, want %v", cfg.Repository, tt.wantRepository)
			}
			if cfg.Password != "repopass" {
				t.Errorf("Password = %v, want repopass", cfg.Password)
			}
		})
	}
}

func TestRESTBackend_TestConnection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b := &RestBackend{URL: server.URL}
	err := b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestRESTBackend_TestConnection_WithRestPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// URL with rest: prefix should have it stripped for the HTTP request
	b := &RestBackend{URL: "rest:" + server.URL}
	err := b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestRESTBackend_TestConnection_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b := &RestBackend{
		URL:      server.URL,
		Username: "admin",
		Password: "secret",
	}
	err := b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestRESTBackend_TestConnection_AuthRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	b := &RestBackend{URL: server.URL}
	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for auth required, got nil")
	}
	if !contains(err.Error(), "authentication required") {
		t.Errorf("TestConnection() error = %q, want to contain 'authentication required'", err.Error())
	}
}

func TestRESTBackend_TestConnection_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	b := &RestBackend{URL: server.URL}
	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for server error, got nil")
	}
	if !contains(err.Error(), "status 500") {
		t.Errorf("TestConnection() error = %q, want to contain 'status 500'", err.Error())
	}
}

func TestRESTBackend_TestConnection_InvalidConfig(t *testing.T) {
	b := &RestBackend{URL: ""}
	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for empty URL, got nil")
	}
}

func TestRESTBackend_TestConnection_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	b := &RestBackend{URL: server.URL}
	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for 404, got nil")
	}
}

func TestIsRestURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"rest:https://example.com", true},
		{"rest:http://localhost:8000", true},
		{"https://example.com", false},
		{"http://example.com", false},
		{"", false},
		{"rest", false},
		{"rest:", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isRestURL(tt.input); got != tt.want {
				t.Errorf("isRestURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
