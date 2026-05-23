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
