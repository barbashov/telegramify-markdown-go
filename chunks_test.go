package telegramify

import (
	"strings"
	"testing"
)

func TestRenderChunksKeepsFormatting(t *testing.T) {
	tests := []struct{ markdown, want string }{
		{"**" + strings.Repeat("x", 4100) + "**", strings.Repeat("x", 4100)},
		{"[source](https://example.org/" + strings.Repeat("x", 33000) + ")", "source"},
		{strings.Repeat("!", 3000), strings.Repeat("!", 3000)},
		{"> " + strings.Repeat("x", 4100), strings.Repeat("x", 4100)},
	}
	for _, tt := range tests {
		var visible strings.Builder
		for _, chunk := range RenderChunks(tt.markdown, DefaultMaxLength) {
			if utf16Len(chunk.Text) > DefaultMaxLength {
				t.Fatal("visible text exceeds limit")
			}
			reparsed := splitMarkup(chunk.MarkdownV2, DefaultMaxLength)
			if len(reparsed) != 1 || reparsed[0].Text != chunk.Text {
				t.Fatalf("rendered chunk is not self-contained: %q", chunk.Text[:min(32, len(chunk.Text))])
			}
			visible.WriteString(chunk.Text)
		}
		if visible.String() != tt.want {
			t.Fatalf("visible length=%d want %d", visible.Len(), len(tt.want))
		}
	}
}

func TestRenderNormalizesLinks(t *testing.T) {
	got := Markdownify("[![schema](https://example.org/img)](https://example.org/?a=1&amp;b=2)")
	if got != "[schema](https://example.org/?a=1&b=2)" {
		t.Fatalf("got %q", got)
	}
}
