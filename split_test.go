package telegramify

import (
	"strings"
	"testing"
)

func TestSplitShortReturnsSingleChunk(t *testing.T) {
	in := "one\ntwo\nthree"
	got := Split(in, DefaultMaxLength)
	if len(got) != 1 || got[0] != in {
		t.Fatalf("expected single unchanged chunk, got %#v", got)
	}
}

func TestSplitEmpty(t *testing.T) {
	if got := Split("", 100); got != nil {
		t.Fatalf("expected nil for empty input, got %#v", got)
	}
	if got := Split("   \n  ", 100); got != nil {
		t.Fatalf("expected nil for whitespace-only input, got %#v", got)
	}
}

func TestSplitRespectsLimitAndReassembles(t *testing.T) {
	limit := 50
	in := strings.TrimSuffix(strings.Repeat("abcde\n", 200), "\n")
	parts := Split(in, limit)
	if len(parts) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(parts))
	}
	for i, p := range parts {
		if utf16Len(p) > limit {
			t.Errorf("chunk %d length %d exceeds limit %d", i, utf16Len(p), limit)
		}
	}
	if got := strings.Join(parts, "\n"); got != in {
		t.Errorf("rejoined chunks do not match original\n got: %q\nwant: %q", got, in)
	}
}

func TestSplitLongSingleLine(t *testing.T) {
	limit := 10
	in := strings.Repeat("x", 95) // single line, no newlines
	parts := Split(in, limit)
	for i, p := range parts {
		if utf16Len(p) > limit {
			t.Errorf("chunk %d length %d exceeds limit %d", i, utf16Len(p), limit)
		}
	}
	if got := strings.Join(parts, ""); got != in {
		t.Errorf("rejoined long line mismatch")
	}
}

func TestSplitDefaultLimit(t *testing.T) {
	// limit <= 0 falls back to DefaultMaxLength.
	in := strings.Repeat("a", DefaultMaxLength+100)
	parts := Split(in, 0)
	for _, p := range parts {
		if utf16Len(p) > DefaultMaxLength {
			t.Errorf("chunk exceeds DefaultMaxLength: %d", utf16Len(p))
		}
	}
}

// Split must not drop blank lines at chunk boundaries.
func TestSplitKeepsBlankLinesAtBoundaries(t *testing.T) {
	got := Split("abc\n\nx", 3)
	want := []string{"abc", "\nx"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestSplitLosslessAcrossParagraphs(t *testing.T) {
	in := "para one\n\npara two\n\npara three"
	parts := Split(in, 10)
	if len(parts) < 2 {
		t.Fatalf("expected multiple chunks, got %#v", parts)
	}
	if got := strings.Join(parts, "\n"); got != in {
		t.Errorf("rejoined chunks lost content\n got: %q\nwant: %q", got, in)
	}
}

// A fenced code block that fits in a chunk is kept atomic.
func TestSplitKeepsFenceAtomic(t *testing.T) {
	parts := Split("12345\n\n```\na\n```", 10)
	want := []string{"12345\n", "```\na\n```"}
	if len(parts) != len(want) {
		t.Fatalf("got %#v want %#v", parts, want)
	}
	for i := range want {
		if parts[i] != want[i] {
			t.Fatalf("got %#v want %#v", parts, want)
		}
	}
}

// An oversized fenced block is split with the fence closed and reopened
// (including the language) so every chunk is valid MarkdownV2.
func TestSplitReopensOversizedFence(t *testing.T) {
	in := "```go\nline one is long\nline two is long\nline three\n```"
	parts := Split(in, 30)
	want := []string{
		"```go\nline one is long\n```",
		"```go\nline two is long\n```",
		"```go\nline three\n```",
	}
	if len(parts) != len(want) {
		t.Fatalf("got %#v want %#v", parts, want)
	}
	for i := range want {
		if parts[i] != want[i] {
			t.Fatalf("chunk %d: got %q want %q", i, parts[i], want[i])
		}
		if utf16Len(parts[i]) > 30 {
			t.Errorf("chunk %d exceeds limit: %d", i, utf16Len(parts[i]))
		}
	}
}

// Every chunk of a mixed document has balanced code fences.
func TestSplitChunksHaveBalancedFences(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("some paragraph text here\n\n```python\ncode line a\ncode line b\n```\n\n")
	}
	parts := Split(strings.TrimSpace(sb.String()), 60)
	for i, p := range parts {
		fences := 0
		for _, line := range strings.Split(p, "\n") {
			if strings.HasPrefix(line, "```") {
				fences++
			}
		}
		if fences%2 != 0 {
			t.Errorf("chunk %d has unbalanced fences (%d):\n%q", i, fences, p)
		}
	}
}

// A single code line longer than the limit is hard-split, with every
// resulting chunk still opened and closed as a fence.
func TestSplitFenceWithOverlongLine(t *testing.T) {
	code := strings.Repeat("x", 50)
	in := "```\n" + code + "\n```"
	limit := 20
	parts := Split(in, limit)
	if len(parts) < 3 {
		t.Fatalf("expected several chunks, got %#v", parts)
	}
	var rejoined strings.Builder
	for i, p := range parts {
		if utf16Len(p) > limit {
			t.Errorf("chunk %d exceeds limit: %d", i, utf16Len(p))
		}
		if !strings.HasPrefix(p, "```\n") || !strings.HasSuffix(p, "\n```") {
			t.Errorf("chunk %d is not a complete fence: %q", i, p)
		}
		rejoined.WriteString(strings.TrimSuffix(strings.TrimPrefix(p, "```\n"), "\n```"))
	}
	if rejoined.String() != code {
		t.Errorf("code content lost: got %q want %q", rejoined.String(), code)
	}
}

// Hard-splitting never separates a backslash from its escaped character.
func TestSplitLongLineKeepsEscapePairs(t *testing.T) {
	// An odd limit would land a boundary in the middle of an escape pair.
	parts := Split(strings.Repeat(`\.`, 10), 3)
	want := []string{`\.`, `\.`, `\.`, `\.`, `\.`, `\.`, `\.`, `\.`, `\.`, `\.`}
	if len(parts) != len(want) {
		t.Fatalf("got %#v want %#v", parts, want)
	}
	for i := range want {
		if parts[i] != want[i] {
			t.Fatalf("got %#v want %#v", parts, want)
		}
	}
	for i, p := range parts {
		if strings.HasSuffix(p, `\`) && !strings.HasSuffix(p, `\\`) {
			t.Errorf("chunk %d ends with a dangling backslash: %q", i, p)
		}
	}
}

// A limit smaller than one rune's UTF-16 width cannot be honored; the rune
// is emitted whole and reassembly stays lossless (documented behavior).
func TestSplitTinyLimitAstralRune(t *testing.T) {
	in := strings.Repeat("😀", 3)
	parts := Split(in, 1)
	if strings.Join(parts, "") != in {
		t.Errorf("astral reassembly mismatch: %#v", parts)
	}
	for _, p := range parts {
		if p == "" {
			t.Errorf("empty chunk emitted")
		}
	}
}

func TestSplitAstralNotBroken(t *testing.T) {
	// Each 😀 is 2 UTF-16 units; a limit that is odd must not split a rune.
	limit := 5
	in := strings.Repeat("😀", 10)
	parts := Split(in, limit)
	for _, p := range parts {
		if utf16Len(p) > limit {
			t.Errorf("chunk exceeds limit: %d", utf16Len(p))
		}
		// Reassembling must reproduce valid runes (no broken surrogate halves).
	}
	if strings.Join(parts, "") != in {
		t.Errorf("astral reassembly mismatch")
	}
}
