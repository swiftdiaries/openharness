package tools

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var blockedHostnames = map[string]bool{
	"localhost":                 true,
	"metadata.google.internal": true,
}

func isBlockedHostname(hostname string) bool {
	hostname = strings.ToLower(hostname)
	if blockedHostnames[hostname] {
		return true
	}
	if strings.HasSuffix(hostname, ".localhost") ||
		strings.HasSuffix(hostname, ".local") ||
		strings.HasSuffix(hostname, ".internal") {
		return true
	}
	return false
}

func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	privateRanges := []string{
		"0.0.0.0/8", "10.0.0.0/8", "127.0.0.0/8",
		"169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16",
		"100.64.0.0/10",
		"::0/128", "::1/128", "fe80::/10", "fec0::/10", "fc00::/7",
	}

	for _, cidrStr := range privateRanges {
		_, cidr, _ := net.ParseCIDR(cidrStr)
		if cidr != nil && cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// CheckSSRF validates a URL against SSRF attacks.
// Returns an error if the URL targets a private/blocked host.
func CheckSSRF(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("missing hostname")
	}

	if isBlockedHostname(hostname) {
		return fmt.Errorf("blocked hostname: %s", hostname)
	}

	// Check if hostname is already an IP.
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(hostname) {
			return fmt.Errorf("private IP address not allowed: %s", hostname)
		}
		return nil
	}

	// DNS pinning: resolve hostname and verify all IPs are public.
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %w", hostname, err)
	}
	for _, addr := range addrs {
		if isPrivateIP(addr) {
			return fmt.Errorf("hostname %s resolves to private IP %s", hostname, addr)
		}
	}

	return nil
}
