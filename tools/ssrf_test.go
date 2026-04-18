package tools

import "testing"

func TestCheckSSRF(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		blocked bool
	}{
		{"public URL", "https://example.com/page", false},
		{"localhost", "http://localhost:8080/secret", true},
		{"127.0.0.1", "http://127.0.0.1:18790/health", true},
		{"private 10.x", "http://10.0.0.1/admin", true},
		{"private 192.168.x", "http://192.168.1.1/config", true},
		{"private 172.16.x", "http://172.16.0.1/api", true},
		{"metadata", "http://metadata.google.internal/v1/token", true},
		{".local suffix", "http://myhost.local/api", true},
		{".internal suffix", "http://service.internal/api", true},
		{"IPv6 loopback", "http://[::1]:8080/", true},
		{"empty hostname", "http:///path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSSRF(tt.url)
			if (err != nil) != tt.blocked {
				t.Errorf("CheckSSRF(%q) error=%v, want blocked=%v", tt.url, err, tt.blocked)
			}
		})
	}
}
