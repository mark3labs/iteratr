package tui

import (
	"strings"
	"testing"
)

func TestSanitizePaste_StripsANSIEscapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "color codes",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "bold formatting",
			input:    "\x1b[1mbold\x1b[22m",
			expected: "bold",
		},
		{
			name:     "256 colors",
			input:    "\x1b[38;5;196mred\x1b[0m",
			expected: "red",
		},
		{
			name:     "cursor control",
			input:    "\x1b[2K\x1b[1Gclear line",
			expected: "clear line",
		},
		{
			name:     "multiple ANSI sequences",
			input:    "\x1b[1m\x1b[31m\x1b[4mbold red underline\x1b[0m",
			expected: "bold red underline",
		},
		{
			name:     "no ANSI sequences",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePaste(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePaste(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePaste_RemovesNullBytes(t *testing.T) {
	input := "hello\x00world\x00"
	expected := "helloworld"
	result := SanitizePaste(input)
	if result != expected {
		t.Errorf("SanitizePaste(%q) = %q, want %q", input, result, expected)
	}
}

func TestSanitizePaste_RemovesNonPrintableControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SOH to BS (1-8)",
			input:    "a\x01b\x02c\x03d\x04e\x05f\x06g\x07h\x08",
			expected: "abcdefgh",
		},
		{
			name:     "VT and FF (11-12)",
			input:    "a\x0bb\x0c",
			expected: "ab",
		},
		{
			name:     "SO to US (14-31)",
			input:    "a\x0eb\x0fc\x10d\x11e\x12f\x13g\x14h\x15i\x16j\x17k\x18l\x19m\x1an\x1bo\x1cp\x1dq\x1er\x1f",
			expected: "abcdefghijklmnopqr",
		},
		{
			name:     "DEL (127)",
			input:    "a\x7fb",
			expected: "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePaste(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePaste(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePaste_PreservesValidWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserves newlines",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "preserves tabs",
			input:    "col1\tcol2\tcol3",
			expected: "col1\tcol2\tcol3",
		},
		{
			name:     "preserves carriage returns (before normalization)",
			input:    "line1\rline2",
			expected: "line1\rline2",
		},
		{
			name:     "preserves regular spaces",
			input:    "hello world  test",
			expected: "hello world  test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePaste(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePaste(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePaste_NormalizesCRLF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CRLF to LF",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "mixed line endings",
			input:    "line1\nline2\r\nline3\nline4",
			expected: "line1\nline2\nline3\nline4",
		},
		{
			name:     "only LF remains LF",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePaste(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePaste(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePaste_TrimsTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trailing spaces",
			input:    "hello world   ",
			expected: "hello world",
		},
		{
			name:     "trailing tabs",
			input:    "hello world\t\t",
			expected: "hello world",
		},
		{
			name:     "trailing newlines",
			input:    "hello world\n\n",
			expected: "hello world",
		},
		{
			name:     "trailing carriage returns",
			input:    "hello world\r\r",
			expected: "hello world",
		},
		{
			name:     "mixed trailing whitespace",
			input:    "hello world \t\n\r",
			expected: "hello world",
		},
		{
			name:     "no trailing whitespace",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "only trailing whitespace on last line",
			input:    "line1\nline2   ",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePaste(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePaste(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePaste_ComplexCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ANSI + null + control + CRLF",
			input:    "\x1b[31mhello\x00\r\n\x01world\x1b[0m\t\n",
			expected: "hello\nworld",
		},
		{
			name:     "terminal output with escapes",
			input:    "\x1b[2K\x1b[1G\x1b[1;32m✓\x1b[0m Success\x00",
			expected: "✓ Success",
		},
		{
			name:     "large content with mixed issues",
			input:    strings.Repeat("\x1b[31m\x00test\x01\r\n\x1b[0m", 100),
			expected: strings.Repeat("test\n", 99) + "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePaste(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePaste(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizePaste_EmptyString(t *testing.T) {
	result := SanitizePaste("")
	if result != "" {
		t.Errorf("SanitizePaste(\"\") = %q, want empty string", result)
	}
}
