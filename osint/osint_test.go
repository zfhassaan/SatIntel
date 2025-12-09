package osint

import (
	"strings"
	"testing"
)

func TestExtractNorad(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid format with parentheses",
			input:    "ISS (ZARYA) (25544)",
			expected: "ZARYA",
		},
		{
			name:     "Simple format",
			input:    "Satellite Name (12345)",
			expected: "12345",
		},
		{
			name:     "Multiple parentheses - takes first",
			input:    "Name (NORAD_ID) (extra)",
			expected: "NORAD_ID",
		},
		{
			name:     "No parentheses",
			input:    "Satellite Name",
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only opening parenthesis",
			input:    "Name (NORAD",
			expected: "",
		},
		{
			name:     "Only closing parenthesis",
			input:    "Name NORAD)",
			expected: "",
		},
		{
			name:     "Reversed parentheses",
			input:    "Name )NORAD(",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNorad(tt.input)
			if result != tt.expected {
				t.Errorf("extractNorad(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenRowString(t *testing.T) {
	tests := []struct {
		name        string
		intro       string
		input       string
		checkPrefix bool
		checkSuffix bool
		checkLength bool
	}{
		{
			name:        "Normal case",
			intro:       "Name",
			input:       "Test",
			checkPrefix: true,
			checkSuffix: true,
			checkLength: true,
		},
		{
			name:        "Empty input",
			intro:       "Field",
			input:       "",
			checkPrefix: true,
			checkSuffix: true,
			checkLength: true,
		},
		{
			name:        "Short intro and input",
			intro:       "ID",
			input:       "123",
			checkPrefix: true,
			checkSuffix: true,
			checkLength: true,
		},
		{
			name:        "Long intro",
			intro:       "Very Long Field Name That Takes Up Space",
			input:       "Value",
			checkPrefix: true,
			checkSuffix: true,
			checkLength: false, // May exceed 67 chars if too long
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenRowString(tt.intro, tt.input)
			// Check that result has correct structure
			if tt.checkPrefix && !strings.HasPrefix(result, "║ ") {
				t.Errorf("GenRowString() should start with '║ '")
			}
			if tt.checkSuffix && !strings.HasSuffix(result, " ║") {
				t.Errorf("GenRowString() should end with ' ║'")
			}
			if !strings.Contains(result, tt.intro+": "+tt.input) {
				t.Errorf("GenRowString() should contain intro and input")
			}
			// Check total length only if not too long
			if tt.checkLength && len(result) != 67 {
				t.Errorf("GenRowString() length = %d, want 67", len(result))
			}
		})
	}
}

// Benchmark tests
func BenchmarkExtractNorad(b *testing.B) {
	testCases := []string{
		"ISS (ZARYA) (25544)",
		"Satellite Name (12345)",
		"Name (NORAD_ID)",
		"Invalid Format",
	}

	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			extractNorad(tc)
		}
	}
}

func BenchmarkGenRowString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenRowString("Field Name", "Field Value")
	}
}

