package httpclient

import (
	"testing"

	"github.com/MacJediWizard/keldris/internal/config"
)

func TestShouldBypassProxy(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		noProxy string
		want    bool
	}{
		{
			name:    "empty no_proxy",
			host:    "example.com",
			noProxy: "",
			want:    false,
		},
		{
			name:    "exact match",
			host:    "example.com",
			noProxy: "example.com",
			want:    true,
		},
		{
			name:    "exact match with port",
			host:    "example.com:8080",
			noProxy: "example.com",
			want:    true,
		},
		{
			name:    "domain suffix match",
			host:    "api.example.com",
			noProxy: ".example.com",
			want:    true,
		},
		{
			name:    "subdomain match",
			host:    "api.example.com",
			noProxy: "example.com",
			want:    true,
		},
		{
			name:    "no match",
			host:    "other.com",
			noProxy: "example.com",
			want:    false,
		},
		{
			name:    "wildcard match",
			host:    "anything.com",
			noProxy: "*",
			want:    true,
		},
		{
			name:    "multiple entries match",
			host:    "api.internal.com",
			noProxy: "example.com, internal.com, test.com",
			want:    true,
		},
		{
			name:    "case insensitive",
			host:    "API.Example.COM",
			noProxy: "example.com",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldBypassProxy(tt.host, tt.noProxy)
			if got != tt.want {
				t.Errorf("shouldBypassProxy(%q, %q) = %v, want %v", tt.host, tt.noProxy, got, tt.want)
			}
		})
	}
}

func TestMaskProxyURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no auth",
			input: "http://proxy:8080",
			want:  "http://proxy:8080",
		},
		{
			name:  "with auth",
			input: "http://user:password@proxy:8080",
			// URL encoding escapes special characters in passwords
			want: "http://user:%2A%2A%2A%2A@proxy:8080",
		},
		{
			name:  "socks5 with auth",
			input: "socks5://admin:secret@socks-proxy:1080",
			// URL encoding escapes special characters in passwords
			want: "socks5://admin:%2A%2A%2A%2A@socks-proxy:1080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskProxyURL(tt.input)
			if got != tt.want {
				t.Errorf("maskProxyURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProxyInfo(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.ProxyConfig
		want string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: "No proxy configured",
		},
		{
			name: "empty config",
			cfg:  &config.ProxyConfig{},
			want: "No proxy configured",
		},
		{
			name: "http proxy only",
			cfg: &config.ProxyConfig{
				HTTPProxy: "http://proxy:8080",
			},
			want: "HTTP: http://proxy:8080",
		},
		{
			name: "socks5 proxy",
			cfg: &config.ProxyConfig{
				SOCKS5Proxy: "socks5://proxy:1080",
			},
			want: "SOCKS5: socks5://proxy:1080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProxyInfo(tt.cfg)
			if got != tt.want {
				t.Errorf("ProxyInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("no proxy", func(t *testing.T) {
		client, err := New(Options{})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if client == nil {
			t.Fatal("New() returned nil client")
		}
	})

	t.Run("with http proxy", func(t *testing.T) {
		client, err := New(Options{
			ProxyConfig: &config.ProxyConfig{
				HTTPProxy: "http://proxy:8080",
			},
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if client == nil {
			t.Fatal("New() returned nil client")
		}
	})

	t.Run("with socks5 proxy", func(t *testing.T) {
		client, err := New(Options{
			ProxyConfig: &config.ProxyConfig{
				SOCKS5Proxy: "socks5://proxy:1080",
			},
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if client == nil {
			t.Fatal("New() returned nil client")
		}
	})
}
