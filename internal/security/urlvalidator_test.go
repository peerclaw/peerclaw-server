package security

import (
	"net"
	"testing"
)

func TestValidateURL_RejectsPrivateIPs(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"http://127.0.0.1/path", true},
		{"http://10.0.0.1/path", true},
		{"http://172.16.0.1/path", true},
		{"http://192.168.1.1/path", true},
		{"http://169.254.169.254/latest/meta-data/", true},
		{"http://[::1]/path", true},
		{"file:///etc/passwd", true},
		{"gopher://localhost", true},
		{"ftp://internal.server", true},
		{"", true},
		{"not-a-url", true},
		{"http:///no-host", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateURL(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateURL(%q) = %v, want nil", tt.url, err)
			}
		})
	}
}

func TestValidateURL_AllowsPublicURLs(t *testing.T) {
	// Only test with IPs we know are public (not requiring DNS resolution).
	tests := []string{
		"https://8.8.8.8/dns-query",
		"http://93.184.216.34/", // example.com IP
	}

	for _, u := range tests {
		t.Run(u, func(t *testing.T) {
			err := ValidateURL(u)
			if err != nil {
				t.Errorf("ValidateURL(%q) = %v, want nil", u, err)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"::1", true},
		{"8.8.8.8", false},
		{"93.184.216.34", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("invalid IP: %s", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}
