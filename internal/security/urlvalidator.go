package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// privateNetworks defines CIDR ranges that should be blocked for SSRF prevention.
var privateNetworks = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

var parsedPrivateNets []*net.IPNet

func init() {
	for _, cidr := range privateNetworks {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			parsedPrivateNets = append(parsedPrivateNets, ipNet)
		}
	}
}

// allowedSchemes defines the URL schemes permitted for outbound requests.
var allowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
}

// AllowLocalhost can be set to true in tests to skip localhost blocking.
var AllowLocalhost = false

// ValidateURL checks that a URL is safe to make requests to.
// It rejects private/internal IPs, disallowed schemes, and metadata endpoints.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme.
	scheme := strings.ToLower(parsed.Scheme)
	if !allowedSchemes[scheme] {
		return fmt.Errorf("disallowed URL scheme: %s", scheme)
	}

	// Resolve hostname to IPs.
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("missing hostname in URL")
	}

	// Check for IP-based metadata endpoints.
	if hostname == "169.254.169.254" || hostname == "metadata.google.internal" {
		return fmt.Errorf("cloud metadata endpoint blocked")
	}

	// Check if hostname is a direct IP.
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) && !AllowLocalhost {
			return fmt.Errorf("private/internal IP blocked: %s", ip)
		}
		return nil
	}

	// Resolve and check IPs.
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %w", hostname, err)
	}

	if !AllowLocalhost {
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return fmt.Errorf("private/internal IP blocked: %s resolves to %s", hostname, ip)
			}
		}
	}

	return nil
}

// isPrivateIP checks if an IP address belongs to a private/internal network.
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	for _, ipNet := range parsedPrivateNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// NewSafeHTTPClient creates an HTTP client that blocks requests to private IPs.
func NewSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, fmt.Errorf("invalid address: %w", err)
				}

				ips, err := net.LookupIP(host)
				if err != nil {
					return nil, fmt.Errorf("DNS resolution failed: %w", err)
				}

				for _, ip := range ips {
					if isPrivateIP(ip) {
						return nil, fmt.Errorf("connection to private IP %s blocked (SSRF protection)", ip)
					}
				}

				// Connect using the original address.
				return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
			},
		},
	}
}
