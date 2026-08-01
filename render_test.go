package telegramify

import "testing"

// Caller-supplied symbols are display text and must be escaped.
func TestConfigSymbolsEscaped(t *testing.T) {
	t.Run("heading symbol", func(t *testing.T) {
		got := Markdownify("# Title", WithHeadingSymbols("*"))
		want := `\* *__Title__*`
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("image symbol", func(t *testing.T) {
		got := Markdownify("![alt](https://img.example.com/x.png)", WithImageSymbol("[img]"))
		want := `[\[img\] alt](https://img.example.com/x.png)`
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
	t.Run("task symbols", func(t *testing.T) {
		got := Markdownify("- [x] done\n- [ ] todo", WithTaskSymbols("[DONE]", "[TODO]"))
		want := "\\[DONE\\] done\n\\[TODO\\] todo"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}

// Telegram forbids nested blockquote entities; nested quotes are flattened.
func TestNestedBlockquoteFlattened(t *testing.T) {
	got := Markdownify("> outer\n> > inner")
	want := ">outer\n>\n>inner"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// A blockquote inside a list item must start at column 0 to be valid.
func TestBlockquoteInListNotIndented(t *testing.T) {
	got := Markdownify("- item\n\n  > quoted")
	want := "• item\n>quoted"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// List-item children keep their source order (nested list not moved last).
func TestListItemChildOrderPreserved(t *testing.T) {
	got := Markdownify("- para1\n\n  para2\n\n  - sub\n\n  para3")
	want := "• para1\n  para2\n  • sub\n  para3"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Heading content ending in italic needs \r to disambiguate _ from __.
func TestHeadingTrailingItalicDisambiguated(t *testing.T) {
	got := Markdownify("# Title _em_")
	want := "📌 *__Title _em_\r__*"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Email autolinks need a mailto: destination.
func TestEmailAutolinkMailto(t *testing.T) {
	got := Markdownify("<user@example.com>")
	want := `[user@example\.com](mailto:user@example.com)`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Only the structural final newline is trimmed from code blocks; intentional
// trailing blank lines are content.
func TestCodeBlockKeepsTrailingBlankLine(t *testing.T) {
	got := Markdownify("```\nline\n\n```")
	want := "```\nline\n\n```"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Telegram cannot nest same-type entities; redundant nesting is dropped.
func TestNestedSameEmphasisFlattened(t *testing.T) {
	got := Markdownify("_a _b_ c_")
	want := "_a b c_"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// CommonMark converts code-span line endings to spaces.
func TestCodeSpanNewlineBecomesSpace(t *testing.T) {
	got := Markdownify("`foo\nbar`")
	want := "`foo bar`"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Inline raw HTML is rendered as escaped literal text.
func TestInlineRawHTMLEscaped(t *testing.T) {
	got := Markdownify("text <br> more")
	want := "text <br\\> more"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Table cells reduce code spans and autolinks to their plain text.
func TestTableCellsWithCodeSpanAndAutolink(t *testing.T) {
	got := Markdownify("| `code.x` | link |\n|---|---|\n| <https://e.com> | y |")
	want := "```\n| code.x        | link |\n|---------------|------|\n| https://e.com | y    |\n```"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// An empty list item still renders its marker.
func TestEmptyListItem(t *testing.T) {
	got := Markdownify("- a\n-\n- b")
	want := "• a\n•\n• b"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Table alignment must use visible (unescaped) cell widths.
func TestTableAlignmentUsesVisibleWidth(t *testing.T) {
	got := Markdownify("| a\\z | bb |\n|---|----|\n| c | d |")
	want := "```\n| a\\\\z | bb |\n|-----|----|\n| c   | d  |\n```"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// Continuation lines align to the visible marker width ("1. " = 3 cols),
// not the escaped width ("1\. " = 4 runes).
func TestListContinuationIndentVisibleWidth(t *testing.T) {
	got := Markdownify("1. first\n\n   second")
	want := "1\\. first\n   second"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
