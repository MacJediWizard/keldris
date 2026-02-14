package notifications

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// blockedCIDRs contains private and reserved IP ranges that must not be used as webhook targets.
var blockedCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
}

// parsedBlockedNets holds the pre-parsed blocked networks.
var parsedBlockedNets []*net.IPNet

func init() {
	for _, cidr := range blockedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid blocked CIDR %q: %v", cidr, err))
		}
		parsedBlockedNets = append(parsedBlockedNets, ipNet)
	}
}

// ValidateWebhookURL validates that a webhook URL is safe to call.
// It blocks private IP ranges, localhost, link-local addresses, and cloud metadata endpoints.
// When requireHTTPS is true, only HTTPS URLs are allowed.
func ValidateWebhookURL(urlStr string, requireHTTPS bool) error {
	if strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("webhook URL is required")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Validate scheme
	switch parsed.Scheme {
	case "https":
		// always allowed
	case "http":
		if requireHTTPS {
			return fmt.Errorf("webhook URL must use HTTPS")
		}
	default:
		return fmt.Errorf("webhook URL must use HTTP or HTTPS scheme")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}

	// Resolve hostname to IP addresses
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("failed to resolve webhook host %q: %w", host, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		for _, blocked := range parsedBlockedNets {
			if blocked.Contains(ip) {
				return fmt.Errorf("webhook URL resolves to blocked address %s", ipStr)
			}
		}

		// Block unspecified addresses (0.0.0.0, ::)
		if ip.IsUnspecified() {
			return fmt.Errorf("webhook URL resolves to blocked address %s", ipStr)
		}
	}

	return nil
}

// isBlockedIP returns true if the IP falls within a private or reserved range.
func isBlockedIP(ip net.IP) bool {
	for _, blocked := range parsedBlockedNets {
		if blocked.Contains(ip) {
			return true
		}
	}
	return ip.IsUnspecified()
}

// ValidatingDialer returns a DialContext function that resolves hostnames and
// checks every resolved IP against blocked ranges before connecting. This
// prevents DNS rebinding attacks where a hostname resolves to a safe IP during
// pre-flight validation but to a private IP at connection time.
func ValidatingDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host %q: %w", host, err)
		}

		// Filter to only safe IPs
		var safeAddrs []string
		for _, ipAddr := range ips {
			if isBlockedIP(ipAddr.IP) {
				continue
			}
			safeAddrs = append(safeAddrs, net.JoinHostPort(ipAddr.IP.String(), port))
		}

		if len(safeAddrs) == 0 {
			return nil, fmt.Errorf("all resolved IPs for %q are blocked (private/reserved)", host)
		}

		// Connect using the first safe IP
		return dialer.DialContext(ctx, network, safeAddrs[0])
	}
}
