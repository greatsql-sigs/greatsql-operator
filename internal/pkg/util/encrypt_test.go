package util

import (
	"strings"
	"testing"
)

func TestBase64Encode(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty string",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "simple string",
			input:    []byte("hello"),
			expected: "aGVsbG8=",
		},
		{
			name:     "string with special characters",
			input:    []byte("hello@123!"),
			expected: "aGVsbG9AMTIzIQ==",
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0x01, 0x02, 0xff},
			expected: "AAEC/w==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Base64Encode(tt.input)
			if result != tt.expected {
				t.Errorf("Base64Encode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBase64Decode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []byte
		expectErr bool
	}{
		{
			name:      "empty string",
			input:     "",
			expected:  []byte(""),
			expectErr: false,
		},
		{
			name:      "valid base64",
			input:     "aGVsbG8=",
			expected:  []byte("hello"),
			expectErr: false,
		},
		{
			name:      "valid base64 with special chars",
			input:     "aGVsbG9AMTIzIQ==",
			expected:  []byte("hello@123!"),
			expectErr: false,
		},
		{
			name:      "invalid base64",
			input:     "invalid!!!base64",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Base64Decode(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Base64Decode() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Base64Decode() unexpected error: %v", err)
				}
				if string(result) != string(tt.expected) {
					t.Errorf("Base64Decode() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestBase64EncodeDecodeRoundTrip(t *testing.T) {
	testCases := []string{
		"",
		"hello world",
		"GreatSQL@2025",
		"密码测试123",
		"!@#$%^&*()_+-=[]{}|;':\",./<>?",
	}

	for _, original := range testCases {
		t.Run(original, func(t *testing.T) {
			encoded := Base64Encode([]byte(original))
			decoded, err := Base64Decode(encoded)
			if err != nil {
				t.Errorf("Base64Decode() error: %v", err)
			}
			if string(decoded) != original {
				t.Errorf("Round trip failed: got %v, want %v", string(decoded), original)
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero length", 0},
		{"negative length", -1},
		{"short password", 8},
		{"medium password", 16},
		{"long password", 32},
		{"very long password", 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := GeneratePassword(tt.length)

			// 验证长度
			if tt.length <= 0 {
				if password != "" {
					t.Errorf("GeneratePassword(%d) should return empty string, got %s", tt.length, password)
				}
				return
			}

			if len(password) != tt.length {
				t.Errorf("GeneratePassword(%d) length = %d, want %d", tt.length, len(password), tt.length)
			}

			// 验证只包含允许的字符
			for _, char := range password {
				if !strings.ContainsRune(alphanum, char) {
					t.Errorf("GeneratePassword() contains invalid character: %c", char)
				}
			}
		})
	}
}

func TestGeneratePasswordRandomness(t *testing.T) {
	// 生成多个密码，确保它们不相同（随机性测试）
	length := 16
	passwords := make(map[string]bool)
	iterations := 100

	for range iterations {
		password := GeneratePassword(length)
		if passwords[password] {
			t.Errorf("GeneratePassword() generated duplicate password: %s", password)
		}
		passwords[password] = true
	}

	// 确保生成了足够多的不同密码
	if len(passwords) != iterations {
		t.Errorf("GeneratePassword() should generate %d unique passwords, got %d", iterations, len(passwords))
	}
}

func TestGeneratePasswordCharacterDistribution(t *testing.T) {
	// 生成一个长密码，检查字符分布
	length := 1000
	password := GeneratePassword(length)

	hasLower := false
	hasUpper := false
	hasDigit := false

	for _, char := range password {
		if char >= 'a' && char <= 'z' {
			hasLower = true
		}
		if char >= 'A' && char <= 'Z' {
			hasUpper = true
		}
		if char >= '0' && char <= '9' {
			hasDigit = true
		}
	}

	// 在足够长的密码中，应该包含所有类型的字符
	if !hasLower {
		t.Error("GeneratePassword() should contain lowercase letters")
	}
	if !hasUpper {
		t.Error("GeneratePassword() should contain uppercase letters")
	}
	if !hasDigit {
		t.Error("GeneratePassword() should contain digits")
	}
}

// Benchmark tests
func BenchmarkBase64Encode(b *testing.B) {
	data := []byte("GreatSQL@2025_test_password_benchmark")
	b.ResetTimer()
	for b.Loop() {
		_ = Base64Encode(data)
	}
}

func BenchmarkBase64Decode(b *testing.B) {
	encoded := "R3JlYXRTUUxAMjAyNV90ZXN0X3Bhc3N3b3JkX2JlbmNobWFyaw=="
	
	for b.Loop() {
		_, _ = Base64Decode(encoded)
	}
}

func BenchmarkGeneratePassword(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		_ = GeneratePassword(16)
	}
}
