package notifications

import (
	"fmt"
	"net"
	"net/url"
	"strings"
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
