package telegramify

import "testing"

func TestEscapeText(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain text 123", "plain text 123"},
		{"_*[]()~`>#+-=|{}.!\\", "\\_\\*\\[\\]\\(\\)\\~\\`\\>\\#\\+\\-\\=\\|\\{\\}\\.\\!\\\\"},
		{"a.b", "a\\.b"},
		{"no specials: ? @ % ^ & / : ;", "no specials: ? @ % ^ & / : ;"},
	}
	for _, tt := range tests {
		if got := escapeText(tt.in); got != tt.want {
			t.Errorf("escapeText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeCode(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain.code-here!", "plain.code-here!"}, // dots/dashes/bangs stay literal in code
		{"a`b", "a\\`b"},
		{"a\\b", "a\\\\b"},
		{"a`b\\c", "a\\`b\\\\c"},
	}
	for _, tt := range tests {
		if got := escapeCode(tt.in); got != tt.want {
			t.Errorf("escapeCode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://example.com/path?a=1&b=2", "https://example.com/path?a=1&b=2"},
		{"https://e.com/x(y)z", "https://e.com/x(y\\)z"},
		{"a\\b", "a\\\\b"},
	}
	for _, tt := range tests {
		if got := escapeURL(tt.in); got != tt.want {
			t.Errorf("escapeURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUTF16Len(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"é", 1},   // BMP, single code unit
		{"€", 1},   // BMP
		{"😀", 2},   // outside BMP -> surrogate pair
		{"a😀b", 4}, // 1 + 2 + 1
		{"日本語", 3}, // BMP CJK
		{"👍🏽", 4},  // emoji + skin tone modifier, both astral
	}
	for _, tt := range tests {
		if got := utf16Len(tt.in); got != tt.want {
			t.Errorf("utf16Len(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
