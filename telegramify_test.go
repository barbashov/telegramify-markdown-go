package telegramify

import "testing"

func TestMarkdownify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "inline emphasis and code",
			in:   "Hello **bold** and _italic_ and ~~strike~~ and `code`.",
			want: "Hello *bold* and _italic_ and ~strike~ and `code`\\.",
		},
		{
			name: "nested emphasis",
			in:   "**bold _italic_ inside**",
			want: "*bold _italic_ inside*",
		},
		{
			name: "spoiler",
			in:   "A spoiler: ||hidden|| here.",
			want: "A spoiler: ||hidden|| here\\.",
		},
		{
			name: "escape special characters",
			in:   "a.b-c=d+e (f) [g] {h} #i !j",
			want: "a\\.b\\-c\\=d\\+e \\(f\\) \\[g\\] \\{h\\} \\#i \\!j",
		},
		{
			name: "question mark and slash not escaped",
			in:   "what? a/b",
			want: "what? a/b",
		},
		{
			name: "headings by level",
			in:   "# H1\n## H2\n### H3\n##### H5",
			want: "📌 *__H1__*\n\n✏️ *__H2__*\n\n📚 *H3*\n\n_H5_",
		},
		{
			name: "fenced code keeps content unescaped",
			in:   "```python\nprint('a.b')\nx = 1\n```",
			want: "```python\nprint('a.b')\nx = 1\n```",
		},
		{
			name: "backslash and backtick escaped inside code",
			in:   "`a\\b`",
			want: "`a\\\\b`",
		},
		{
			name: "link keeps url unescaped except paren",
			in:   "[link](https://example.com/path?a=1)",
			want: "[link](https://example.com/path?a=1)",
		},
		{
			name: "link escapes closing paren in url",
			in:   "[a](https://e.com/x(y)z)",
			want: "[a](https://e.com/x(y\\)z)",
		},
		{
			name: "image becomes link with symbol",
			in:   "![alt text](https://img.example.com/x.png)",
			want: "[🖼 alt text](https://img.example.com/x.png)",
		},
		{
			name: "custom emoji preserved",
			in:   "![👍](tg://emoji?id=123)",
			want: "![👍](tg://emoji?id=123)",
		},
		{
			name: "unordered nested list",
			in:   "- one\n- two\n  - nested\n- three",
			want: "• one\n• two\n  • nested\n• three",
		},
		{
			name: "ordered list escapes dot",
			in:   "1. first\n2. second",
			want: "1\\. first\n2\\. second",
		},
		{
			name: "task list",
			in:   "- [x] done\n- [ ] todo",
			want: "✅ done\n☐ todo",
		},
		{
			name: "short blockquote",
			in:   "> quote line one\n> quote line two",
			want: ">quote line one\n>quote line two",
		},
		{
			name: "table rendered as monospace block",
			in:   "| a | bb | ccc |\n|---|----|-----|\n| 1 | 22 | 333 |",
			want: "```\n| a | bb | ccc |\n|---|----|-----|\n| 1 | 22 | 333 |\n```",
		},
		{
			name: "thematic break",
			in:   "---",
			want: "————————",
		},
		{
			name: "bare url is not auto-linked",
			in:   "see https://example.com now",
			want: "see https://example\\.com now",
		},
		{
			name: "paragraph spacing",
			in:   "para one\n\npara two",
			want: "para one\n\npara two",
		},
		{
			name: "soft line break preserved",
			in:   "line a\nline b",
			want: "line a\nline b",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Markdownify(tt.in)
			if got != tt.want {
				t.Errorf("Markdownify(%q)\n got: %q\nwant: %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMarkdownifyExpandableBlockquote(t *testing.T) {
	var sb string
	for i := 0; i < 60; i++ {
		sb += "word "
	}
	got := Markdownify("> " + sb)
	if len(got) < 3 || got[:3] != "**>" {
		t.Fatalf("expected expandable blockquote to start with **>, got prefix %q", got[:3])
	}
	if got[len(got)-2:] != "||" {
		t.Fatalf("expected expandable blockquote to end with ||, got suffix %q", got[len(got)-2:])
	}
}

func TestMarkdownifyOptions(t *testing.T) {
	t.Run("custom heading symbol", func(t *testing.T) {
		got := Markdownify("# Title", WithHeadingSymbols("➤"))
		want := "➤ *__Title__*"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

	t.Run("custom unordered marker", func(t *testing.T) {
		got := Markdownify("- item", WithUnorderedMarker("-"))
		// "-" is a MarkdownV2 special char and must be escaped.
		want := "\\- item"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})

	t.Run("disable expandable blockquote", func(t *testing.T) {
		var sb string
		for i := 0; i < 60; i++ {
			sb += "word "
		}
		got := Markdownify("> "+sb, WithCiteExpandable(false))
		if got[:3] == "**>" {
			t.Errorf("expected non-expandable blockquote, got %q", got[:10])
		}
	})

	t.Run("custom task symbols", func(t *testing.T) {
		got := Markdownify("- [x] done", WithTaskSymbols("[DONE]", "[TODO]"))
		want := "[DONE] done"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
