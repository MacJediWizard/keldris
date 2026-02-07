// Package httpclient provides HTTP client utilities with proxy support.
package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"golang.org/x/net/proxy"
)

// DefaultTimeout is the default HTTP client timeout.
const DefaultTimeout = 30 * time.Second

// Options configures the HTTP client.
type Options struct {
	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration
	// ProxyConfig contains proxy settings
	ProxyConfig *config.ProxyConfig
}

// New creates a new HTTP client with optional proxy support.
func New(opts Options) (*http.Client, error) {
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Configure proxy if provided
	if opts.ProxyConfig != nil && opts.ProxyConfig.HasProxy() {
		if err := configureProxy(transport, opts.ProxyConfig); err != nil {
			return nil, fmt.Errorf("configure proxy: %w", err)
		}
	}

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}, nil
}

// NewWithConfig creates an HTTP client using the agent configuration.
func NewWithConfig(cfg *config.AgentConfig, timeout time.Duration) (*http.Client, error) {
	var proxyConfig *config.ProxyConfig
	if cfg != nil {
		proxyConfig = cfg.GetProxyConfig()
	}

	return New(Options{
		Timeout:     timeout,
		ProxyConfig: proxyConfig,
	})
}

// NewSimple creates a simple HTTP client with timeout and no proxy.
func NewSimple(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

// configureProxy sets up proxy configuration on the transport.
func configureProxy(transport *http.Transport, cfg *config.ProxyConfig) error {
	// SOCKS5 proxy takes precedence if configured
	if cfg.SOCKS5Proxy != "" {
		return configureSocks5Proxy(transport, cfg.SOCKS5Proxy)
	}

	// HTTP/HTTPS proxy
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxyFunc(req, cfg)
	}

	return nil
}

// configureSocks5Proxy sets up a SOCKS5 proxy dialer.
func configureSocks5Proxy(transport *http.Transport, socks5URL string) error {
	proxyURL, err := url.Parse(socks5URL)
	if err != nil {
		return fmt.Errorf("parse SOCKS5 proxy URL: %w", err)
	}

	// Extract auth if present
	var auth *proxy.Auth
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     proxyURL.User.Username(),
			Password: password,
		}
	}

	// Create SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
	if err != nil {
		return fmt.Errorf("create SOCKS5 dialer: %w", err)
	}

	// Wrap the dialer to implement DialContext
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}

	return nil
}

// proxyFunc returns the proxy URL for the given request.
func proxyFunc(req *http.Request, cfg *config.ProxyConfig) (*url.URL, error) {
	// Check if host should bypass proxy
	if shouldBypassProxy(req.URL.Host, cfg.NoProxy) {
		return nil, nil
	}

	// Use HTTPS proxy for https requests, HTTP proxy for http requests
	var proxyURLStr string
	if req.URL.Scheme == "https" && cfg.HTTPSProxy != "" {
		proxyURLStr = cfg.HTTPSProxy
	} else if cfg.HTTPProxy != "" {
		proxyURLStr = cfg.HTTPProxy
	}

	if proxyURLStr == "" {
		return nil, nil
	}

	return url.Parse(proxyURLStr)
}

// shouldBypassProxy checks if a host should bypass the proxy.
func shouldBypassProxy(host string, noProxy string) bool {
	if noProxy == "" {
		return false
	}

	// Remove port from host if present
	hostOnly, _, err := net.SplitHostPort(host)
	if err != nil {
		hostOnly = host
	}

	// Check against each no_proxy entry
	for _, pattern := range strings.Split(noProxy, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Wildcard match
		if pattern == "*" {
			return true
		}

		// Exact match
		if strings.EqualFold(hostOnly, pattern) {
			return true
		}

		// Domain suffix match (e.g., .example.com)
		if strings.HasPrefix(pattern, ".") {
			if strings.HasSuffix(strings.ToLower(hostOnly), strings.ToLower(pattern)) {
				return true
			}
		}

		// Subdomain match (e.g., example.com matches foo.example.com)
		if strings.HasSuffix(strings.ToLower(hostOnly), "."+strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// TestProxy tests if the proxy configuration is working by making a request.
func TestProxy(ctx context.Context, cfg *config.ProxyConfig, testURL string) error {
	client, err := New(Options{
		Timeout:     10 * time.Second,
		ProxyConfig: cfg,
	})
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, testURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("proxy request failed: %w", err)
	}
	defer resp.Body.Close()

	// Any response means the proxy is working
	return nil
}

// ProxyInfo returns a description of the configured proxy.
func ProxyInfo(cfg *config.ProxyConfig) string {
	if cfg == nil || !cfg.HasProxy() {
		return "No proxy configured"
	}

	var parts []string
	if cfg.SOCKS5Proxy != "" {
		parts = append(parts, fmt.Sprintf("SOCKS5: %s", maskProxyURL(cfg.SOCKS5Proxy)))
	}
	if cfg.HTTPProxy != "" {
		parts = append(parts, fmt.Sprintf("HTTP: %s", maskProxyURL(cfg.HTTPProxy)))
	}
	if cfg.HTTPSProxy != "" {
		parts = append(parts, fmt.Sprintf("HTTPS: %s", maskProxyURL(cfg.HTTPSProxy)))
	}
	if cfg.NoProxy != "" {
		parts = append(parts, fmt.Sprintf("NoProxy: %s", cfg.NoProxy))
	}

	return strings.Join(parts, ", ")
}

// maskProxyURL masks credentials in a proxy URL for display.
func maskProxyURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	if u.User != nil {
		username := u.User.Username()
		if _, hasPass := u.User.Password(); hasPass {
			u.User = url.UserPassword(username, "****")
		}
	}

	return u.String()
}
