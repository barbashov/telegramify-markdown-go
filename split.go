package telegramify

import "strings"

// DefaultMaxLength is Telegram's maximum message length in UTF-16 code units.
const DefaultMaxLength = 4096

// Split breaks text into chunks that each fit within limit UTF-16 code units
// (the unit Telegram measures message length in). If limit is <= 0,
// DefaultMaxLength is used.
//
// Splitting happens at newline boundaries. Fenced code blocks are kept intact
// when they fit within the limit; an oversized block is split by closing the
// fence at the chunk boundary and reopening it (with its language) in the
// next chunk, so every chunk remains valid MarkdownV2. Other multi-line
// entities (e.g. emphasis spanning a line break, blockquotes) are not
// reopened across chunks — keep such blocks below the limit to avoid broken
// markup.
//
// A single line longer than the limit is hard-split at rune boundaries,
// keeping backslash escape pairs together. A chunk may exceed the limit only
// when it is impossible to stay within it: a limit smaller than one rune's
// UTF-16 width (e.g. limit 1 with an emoji) or smaller than one escape pair.
// Chunks consisting solely of whitespace are dropped.
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

	s := &splitter{limit: limit}
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "```") {
			end := -1
			for j := i + 1; j < len(lines); j++ {
				if isClosingFence(lines[j]) {
					end = j
					break
				}
			}
			if end >= 0 {
				block := strings.Join(lines[i:end+1], "\n")
				if bl := utf16Len(block); bl <= limit {
					s.addAtomic(block, bl)
				} else {
					s.addFencedBlock(lines[i:end+1], line)
				}
				i = end
				continue
			}
			// Unterminated fence: fall through to plain line handling.
		}
		s.addLine(line)
	}
	s.flush()
	return s.chunks
}

// isClosingFence reports whether line consists solely of three or more
// backticks.
func isClosingFence(line string) bool {
	return len(line) >= 3 && strings.TrimLeft(line, "`") == ""
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
		s.chunks = append(s.chunks, splitLongLine(line, s.limit)...)
		return
	}
	s.append(line, lineLen)
}

// addAtomic appends a multi-line block that must not be split.
func (s *splitter) addAtomic(block string, blockLen int) {
	s.append(block, blockLen)
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

// addFencedBlock splits a fenced code block that exceeds the limit on its
// own. Each emitted chunk is opened with the original fence line (including
// the language) and closed with ```, so every chunk parses as a complete
// code block.
func (s *splitter) addFencedBlock(lines []string, opener string) {
	s.flush()
	const closing = "\n```"
	openLen := utf16Len(opener)
	closeLen := utf16Len(closing)

	s.cur.WriteString(opener)
	s.curLen = openLen
	s.started = true
	for _, line := range lines[1 : len(lines)-1] {
		pieces := []string{line}
		if lineLen := utf16Len(line); openLen+1+lineLen+closeLen > s.limit {
			avail := s.limit - openLen - 1 - closeLen
			if avail < 1 {
				avail = 1
			}
			pieces = splitLongLine(line, avail)
		}
		for _, piece := range pieces {
			pieceLen := utf16Len(piece)
			if s.curLen+1+pieceLen+closeLen > s.limit {
				s.cur.WriteString(closing)
				s.chunks = append(s.chunks, s.cur.String())
				s.cur.Reset()
				s.cur.WriteString(opener)
				s.curLen = openLen
			}
			s.cur.WriteByte('\n')
			s.cur.WriteString(piece)
			s.curLen += 1 + pieceLen
		}
	}
	s.cur.WriteString(closing)
	s.chunks = append(s.chunks, s.cur.String())
	s.cur.Reset()
	s.curLen = 0
	s.started = false
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
