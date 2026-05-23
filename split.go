package telegramify

import "strings"

// DefaultMaxLength is Telegram's maximum message length in UTF-16 code units.
const DefaultMaxLength = 4096

// Split breaks text into chunks that each fit within limit UTF-16 code units
// (the unit Telegram measures message length in). If limit is <= 0,
// DefaultMaxLength is used.
//
// Splitting happens at newline boundaries so that, as long as every individual
// block is shorter than the limit, no MarkdownV2 markup is broken across
// chunks. A single line longer than the limit is hard-split at rune boundaries
// as a last resort, which may break markup — keep individual blocks below the
// limit to avoid that.
func Split(text string, limit int) []string {
	if limit <= 0 {
		limit = DefaultMaxLength
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if utf16Len(text) <= limit {
		return []string{text}
	}

	var chunks []string
	var cur strings.Builder
	curLen := 0

	flush := func() {
		if cur.Len() > 0 {
			chunks = append(chunks, cur.String())
			cur.Reset()
			curLen = 0
		}
	}

	for _, line := range strings.Split(text, "\n") {
		lineLen := utf16Len(line)

		if lineLen > limit {
			// The line alone exceeds the limit; emit what we have and split it.
			flush()
			chunks = append(chunks, splitLongLine(line, limit)...)
			continue
		}

		sep := 0
		if cur.Len() > 0 {
			sep = 1 // the '\n' that will rejoin this line to the current chunk
		}
		if curLen+sep+lineLen > limit {
			flush()
			sep = 0
		}
		if sep == 1 {
			cur.WriteByte('\n')
			curLen++
		}
		cur.WriteString(line)
		curLen += lineLen
	}
	flush()
	return chunks
}

// splitLongLine hard-splits a single line into pieces of at most limit UTF-16
// code units, breaking only at rune boundaries.
func splitLongLine(line string, limit int) []string {
	var parts []string
	var b strings.Builder
	n := 0
	for _, r := range line {
		rl := 1
		if r > 0xFFFF {
			rl = 2
		}
		if n+rl > limit && b.Len() > 0 {
			parts = append(parts, b.String())
			b.Reset()
			n = 0
		}
		b.WriteRune(r)
		n += rl
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}
