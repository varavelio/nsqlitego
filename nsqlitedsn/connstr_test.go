package nsqlitedsn

import (
	"testing"
)

func TestConnStr(t *testing.T) {
	t.Run("NewConnStrFromText", func(t *testing.T) {
		tests := []struct {
			name        string
			input       string
			expected    ConnStr
			expectError bool
		}{
			{
				name:  "Valid HTTP without port and authToken",
				input: "http://example.com",
				expected: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "9876",
				},
				expectError: false,
			},
			{
				name:  "Valid HTTPS with port and authToken",
				input: "https://example.com:8080?authToken=secret",
				expected: ConnStr{
					Protocol:  "https",
					Host:      "example.com",
					Port:      "8080",
					AuthToken: "secret",
				},
				expectError: false,
			},
			{
				name:        "Invalid protocol",
				input:       "ftp://example.com",
				expected:    ConnStr{},
				expectError: true,
			},
			{
				name:        "Missing protocol",
				input:       "://example.com",
				expected:    ConnStr{},
				expectError: true,
			},
			{
				name:        "Missing host",
				input:       "http://:8080",
				expected:    ConnStr{},
				expectError: true,
			},
			{
				name:  "Valid HTTP with port but without authToken",
				input: "http://example.com:9090",
				expected: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "9090",
				},
				expectError: false,
			},
			{
				name:  "Valid HTTPS without port but with authToken",
				input: "https://example.com?authToken=token123",
				expected: ConnStr{
					Protocol:  "https",
					Host:      "example.com",
					Port:      "9876",
					AuthToken: "token123",
				},
				expectError: false,
			},
			{
				name:        "Invalid URL format",
				input:       "http//example.com",
				expected:    ConnStr{},
				expectError: true,
			},
			{
				name:  "Host with IP address",
				input: "http://192.168.1.1:8000?authToken=abc",
				expected: ConnStr{
					Protocol:  "http",
					Host:      "192.168.1.1",
					Port:      "8000",
					AuthToken: "abc",
				},
				expectError: false,
			},
			{
				name:  "Host with subdomain",
				input: "https://sub.domain.example.com",
				expected: ConnStr{
					Protocol: "https",
					Host:     "sub.domain.example.com",
					Port:     "9876",
				},
				expectError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := NewConnStrFromText(tt.input)
				if tt.expectError {
					if err == nil {
						t.Errorf("expected an error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("did not expect an error but got: %v", err)
					}
					if *result != tt.expected {
						t.Errorf("expected: %+v, got: %+v", tt.expected, result)
					}
				}
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name     string
			connStr  ConnStr
			expected string
		}{
			{
				name: "All fields set without authToken",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "example.com",
					Port:     "8443",
				},
				expected: "https://example.com:8443",
			},
			{
				name: "All fields set with authToken",
				connStr: ConnStr{
					Protocol:  "http",
					Host:      "localhost",
					Port:      "8080",
					AuthToken: "secret",
				},
				expected: "http://localhost:8080?authToken=***REDACTED***",
			},
			{
				name:     "Empty ConnStr uses defaults without authToken",
				connStr:  ConnStr{},
				expected: "http://localhost:9876",
			},
			{
				name: "Empty ConnStr with AuthToken",
				connStr: ConnStr{
					AuthToken: "token",
				},
				expected: "http://localhost:9876?authToken=***REDACTED***",
			},
			{
				name: "Partial fields set without authToken",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "example.com",
				},
				expected: "https://example.com:9876",
			},
			{
				name: "Partial fields set with authToken",
				connStr: ConnStr{
					Protocol:  "https",
					Host:      "example.com",
					AuthToken: "abc123",
				},
				expected: "https://example.com:9876?authToken=***REDACTED***",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.connStr.String()
				if result != tt.expected {
					t.Errorf("expected: %s, got: %s", tt.expected, result)
				}
			})
		}
	})

	t.Run("BaseUrlStr", func(t *testing.T) {
		tests := []struct {
			name     string
			connStr  ConnStr
			expected string
		}{
			{
				name: "All fields set without authToken",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "example.com",
					Port:     "8443",
				},
				expected: "https://example.com:8443",
			},
			{
				name: "All fields set with authToken",
				connStr: ConnStr{
					Protocol:  "http",
					Host:      "localhost",
					Port:      "8080",
					AuthToken: "secret",
				},
				expected: "http://localhost:8080",
			},
			{
				name:     "Empty ConnStr uses defaults",
				connStr:  ConnStr{},
				expected: "http://localhost:9876",
			},
			{
				name: "Partial fields set without authToken",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "example.com",
				},
				expected: "https://example.com:9876",
			},
			{
				name: "Partial fields set with authToken",
				connStr: ConnStr{
					Protocol:  "https",
					Host:      "example.com",
					AuthToken: "abc123",
				},
				expected: "https://example.com:9876",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.connStr.BaseUrlStr()
				if result != tt.expected {
					t.Errorf("expected: %s, got: %s", tt.expected, result)
				}
			})
		}
	})

	t.Run("CreateUrlStr", func(t *testing.T) {
		tests := []struct {
			name        string
			connStr     ConnStr
			path        string
			expectedURL string
			expectError bool
		}{
			{
				name: "Valid path without leading slash",
				connStr: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "8080",
				},
				path:        "api/v1/resource",
				expectedURL: "http://example.com:8080/api/v1/resource",
				expectError: false,
			},
			{
				name: "Valid path with leading slash",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "localhost",
					Port:     "443",
				},
				path:        "/dashboard",
				expectedURL: "https://localhost:443/dashboard",
				expectError: false,
			},
			{
				name: "Empty path",
				connStr: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "80",
				},
				path:        "",
				expectedURL: "http://example.com:80",
				expectError: false,
			},
			{
				name: "Path with special characters",
				connStr: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "80",
				},
				path:        "search?q=golang testing",
				expectedURL: "http://example.com:80/search?q=golang testing",
				expectError: false,
			},
			{
				name:    "Empty ConnStr with path",
				connStr: ConnStr{
					// Defaults will be used
				},
				path:        "home",
				expectedURL: "http://localhost:9876/home",
				expectError: false,
			},
			{
				name: "Path resulting in root",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "example.com",
					Port:     "443",
				},
				path:        "/",
				expectedURL: "https://example.com:443/",
				expectError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := tt.connStr.CreateUrlStr(tt.path)
				if tt.expectError {
					if err == nil {
						t.Errorf("expected an error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("did not expect an error but got: %v", err)
					} else if result != tt.expectedURL {
						t.Errorf("expected: %s, got: %s", tt.expectedURL, result)
					}
				}
			})
		}
	})

	t.Run("CreateUrl", func(t *testing.T) {
		tests := []struct {
			name        string
			connStr     ConnStr
			path        string
			expectedURL string
			expectError bool
		}{
			{
				name: "Valid path without leading slash",
				connStr: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "8080",
				},
				path:        "api/v1/resource",
				expectedURL: "http://example.com:8080/api/v1/resource",
				expectError: false,
			},
			{
				name: "Valid path with leading slash",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "localhost",
					Port:     "443",
				},
				path:        "/dashboard",
				expectedURL: "https://localhost:443/dashboard",
				expectError: false,
			},
			{
				name: "Empty path",
				connStr: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "80",
				},
				path:        "",
				expectedURL: "http://example.com:80",
				expectError: false,
			},
			{
				name: "Path with special characters",
				connStr: ConnStr{
					Protocol: "http",
					Host:     "example.com",
					Port:     "80",
				},
				path:        "search?q=golang testing",
				expectedURL: "http://example.com:80/search?q=golang testing",
				expectError: false,
			},
			{
				name:    "Empty ConnStr with path",
				connStr: ConnStr{
					// Defaults will be used
				},
				path:        "home",
				expectedURL: "http://localhost:9876/home",
				expectError: false,
			},
			{
				name: "Path resulting in root",
				connStr: ConnStr{
					Protocol: "https",
					Host:     "example.com",
					Port:     "443",
				},
				path:        "/",
				expectedURL: "https://example.com:443/",
				expectError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := tt.connStr.CreateUrl(tt.path)
				if tt.expectError {
					if err == nil {
						t.Errorf("expected an error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("did not expect an error but got: %v", err)
					} else if result.String() != tt.expectedURL {
						t.Errorf("expected: %s, got: %s", tt.expectedURL, result.String())
					}
				}
			})
		}
	})

	t.Run("setDefaultsIfEmpty", func(t *testing.T) {
		tests := []struct {
			name        string
			initial     ConnStr
			expected    ConnStr
			methodToUse string // "String", "BaseUrlStr", "CreateUrlStr", "CreateUrl"
		}{
			{
				name:        "Empty ConnStr defaults via String",
				initial:     ConnStr{},
				expected:    ConnStr{Protocol: "http", Host: "localhost", Port: "9876"},
				methodToUse: "String",
			},
			{
				name:        "Partial ConnStr defaults via BaseUrlStr",
				initial:     ConnStr{Protocol: "https"},
				expected:    ConnStr{Protocol: "https", Host: "localhost", Port: "9876"},
				methodToUse: "BaseUrlStr",
			},
			{
				name:        "Partial ConnStr defaults via CreateUrlStr",
				initial:     ConnStr{Host: "example.com"},
				expected:    ConnStr{Protocol: "http", Host: "example.com", Port: "9876"},
				methodToUse: "CreateUrlStr",
			},
			{
				name:        "Partial ConnStr defaults via CreateUrl",
				initial:     ConnStr{Port: "8000"},
				expected:    ConnStr{Protocol: "http", Host: "localhost", Port: "8000"},
				methodToUse: "CreateUrl",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				switch tt.methodToUse {
				case "String":
					_ = tt.initial.String()
				case "BaseUrlStr":
					_ = tt.initial.BaseUrlStr()
				case "CreateUrlStr":
					_, _ = tt.initial.CreateUrlStr("testpath")
				case "CreateUrl":
					_, _ = tt.initial.CreateUrl("testpath")
				}

				if tt.initial.Protocol != tt.expected.Protocol {
					t.Errorf(
						"expected Protocol: %s, got: %s",
						tt.expected.Protocol, tt.initial.Protocol,
					)
				}
				if tt.initial.Host != tt.expected.Host {
					t.Errorf(
						"expected Host: %s, got: %s",
						tt.expected.Host, tt.initial.Host,
					)
				}
				if tt.initial.Port != tt.expected.Port {
					t.Errorf(
						"expected Port: %s, got: %s",
						tt.expected.Port, tt.initial.Port,
					)
				}
			})
		}
	})
}
