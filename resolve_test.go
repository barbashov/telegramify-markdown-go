package telegramify

import "testing"

// Backslash escapes and HTML entities in the Markdown source must be
// resolved before MarkdownV2 escaping, not escaped literally.
func TestResolveSourceEscapesAndEntities(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"backslash escape", `a \* b`, `a \* b`},
		{"named entity", "AT&amp;T", "AT&T"},
		{"numeric entity", "x &#35; y", `x \# y`},
		{"hex entity", "x &#x21; y", `x \! y`},
		{"entity resolving to special char", "a &gt; b", `a \> b`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Markdownify(tt.in); got != tt.want {
				t.Errorf("Markdownify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveRawText(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`\*`, `*`},
		{`\a`, `\a`}, // backslash before non-punct stays literal
		{`\`, `\`},
		{"&amp;", "&"},
		{"&#65;", "A"},
		{"&#x41;", "A"},
		{"&#X41;", "A"}, // uppercase hex marker
		{"&#x4A;", "J"}, // hex digit A-F
		{"&#x4a;", "J"}, // hex digit a-f
		{"&#xFF;", "ÿ"}, // uppercase hex digits
		{"&#0;", "�"},   // invalid code point -> replacement char
		{"&notanentity;", "&notanentity;"},
		{"&missingsemicolon", "&missingsemicolon"},
		{"plain", "plain"},
	}
	for _, tt := range tests {
		if got := resolveRawText([]byte(tt.in)); got != tt.want {
			t.Errorf("resolveRawText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
