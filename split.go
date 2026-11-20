package telegramify

import "strings"

// DefaultMaxLength is Telegram's maximum message length in UTF-16 code units.
const DefaultMaxLength = 4096

// Split breaks Telegram MarkdownV2 into independently sendable chunks.
// The limit applies to visible UTF-16 units. Escape pairs and runes stay intact.
func Split(text string, limit int) []string {
	if !strings.ContainsAny(text, "_*[`~|>") && !strings.Contains(text, "\\") {
		return splitPlain(text, limit)
	}
	chunks := splitMarkup(text, limit)
	var parts []string
	for _, chunk := range chunks {
		parts = append(parts, chunk.MarkdownV2)
	}
	return parts
}

func splitPlain(text string, limit int) []string {
	if limit <= 0 {
		limit = DefaultMaxLength
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if utf16Len(text) <= limit {
		return []string{text}
	}

	s := &splitter{limit: limit}
	for _, line := range strings.Split(text, "\n") {
		s.addLine(line)
	}
	s.flush()
	return s.chunks
}

// splitter accumulates lines into chunks of at most limit UTF-16 code units.
type splitter struct {
	limit   int
	chunks  []string
	cur     strings.Builder
	curLen  int
	started bool
}

// flush emits the current chunk unless it is empty or whitespace-only.
func (s *splitter) flush() {
	if s.started && strings.TrimSpace(s.cur.String()) != "" {
		s.chunks = append(s.chunks, s.cur.String())
	}
	s.cur.Reset()
	s.curLen = 0
	s.started = false
}

// addLine appends one line (which may be empty) to the current chunk,
// flushing first if it would not fit.
func (s *splitter) addLine(line string) {
	lineLen := utf16Len(line)
	if lineLen > s.limit {
		s.flush()
		for _, p := range splitLongLine(line, s.limit) {
			if strings.TrimSpace(p) != "" {
				s.chunks = append(s.chunks, p)
			}
		}
		return
	}
	s.append(line, lineLen)
}

func (s *splitter) append(part string, partLen int) {
	sep := 0
	if s.started {
		sep = 1 // the '\n' rejoining this part to the current chunk
	}
	if s.curLen+sep+partLen > s.limit {
		s.flush()
		sep = 0
	}
	if sep == 1 {
		s.cur.WriteByte('\n')
		s.curLen++
	}
	s.cur.WriteString(part)
	s.curLen += partLen
	s.started = true
}

// splitLongLine hard-splits a single line into pieces of at most limit UTF-16
// code units, breaking only at rune boundaries and keeping backslash escape
// pairs together so no piece ends with a dangling backslash.
func splitLongLine(line string, limit int) []string {
	var parts []string
	var b strings.Builder
	n := 0
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		unit := string(runes[i])
		unitLen := runeUTF16Len(runes[i])
		if runes[i] == '\\' && i+1 < len(runes) {
			unit += string(runes[i+1])
			unitLen += runeUTF16Len(runes[i+1])
			i++
		}
		if n+unitLen > limit && b.Len() > 0 {
			parts = append(parts, b.String())
			b.Reset()
			n = 0
		}
		b.WriteString(unit)
		n += unitLen
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}

// runeUTF16Len returns the UTF-16 width of a single rune (2 for astral
// runes, which need a surrogate pair).
func runeUTF16Len(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}
