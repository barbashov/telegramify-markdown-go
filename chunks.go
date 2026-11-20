package telegramify

import (
	"strings"
	"unicode/utf8"
)

const telegramByteLimit = 32768

// Chunk contains a MarkdownV2 part and its unformatted text.
type Chunk struct {
	MarkdownV2 string
	Text       string
}

// RenderChunks converts CommonMark into independently sendable Telegram messages.
// The limit is measured in visible UTF-16 units. A nonpositive limit uses
// DefaultMaxLength. An indivisible rune can exceed a limit of one unit.
func RenderChunks(markdown string, limit int, opts ...Option) []Chunk {
	return splitMarkup(Markdownify(markdown, opts...), limit)
}

type marker struct {
	open, close string
	kind        string
}

type chunkWriter struct {
	limit, units int
	visible      strings.Builder
	output       strings.Builder
	stack        []marker
	chunks       []Chunk
}

func (w *chunkWriter) current() marker {
	if len(w.stack) == 0 {
		return marker{}
	}
	return w.stack[len(w.stack)-1]
}

func (w *chunkWriter) write(markup string) {
	s := w.output.String()
	if strings.HasSuffix(s, "_") && strings.HasPrefix(markup, "_") && !strings.HasSuffix(s, "\\_") {
		w.output.WriteByte('\r')
	}
	w.output.WriteString(markup)
}

func (w *chunkWriter) flush() {
	plain := w.visible.String()
	if strings.TrimSpace(plain) != "" {
		for i := len(w.stack) - 1; i >= 0; i-- {
			w.write(w.stack[i].close)
		}
		w.chunks = append(w.chunks, Chunk{MarkdownV2: w.output.String(), Text: plain})
	}
	w.output.Reset()
	w.visible.Reset()
	w.units = 0
	for _, m := range w.stack {
		w.write(m.open)
	}
}

func (w *chunkWriter) push(m marker) {
	w.stack = append(w.stack, m)
	w.write(m.open)
}

func (w *chunkWriter) pop() {
	w.write(w.current().close)
	w.stack = w.stack[:len(w.stack)-1]
}

func (w *chunkWriter) put(raw, plain string) {
	width := utf16Len(plain)
	if w.units > 0 && (w.units+width > w.limit || w.output.Len()+len(raw) > telegramByteLimit-1024) {
		w.flush()
	}
	w.write(raw)
	w.visible.WriteString(plain)
	w.units += width
}

func splitMarkup(text string, limit int) []Chunk {
	if limit <= 0 {
		limit = DefaultMaxLength
	}
	w := chunkWriter{limit: min(limit, DefaultMaxLength)}
	for i := 0; i < len(text); {
		current := w.current()
		isCode := current.kind == "code" || current.kind == "pre"
		switch {
		case text[i] == '\\' && i+1 < len(text) && text[i+1] >= 1 && text[i+1] <= 126:
			w.put(text[i:i+2], text[i+1:i+2])
			i += 2
			continue
		case current.kind == "pre" && strings.HasPrefix(text[i:], "\n```"):
			w.pop()
			i += len("\n```")
			continue
		case isCode:
			if current.kind == "pre" && (i == 0 || text[i-1] == '\n') {
				line, _, _ := strings.Cut(text[i:], "\n")
				width := utf16Len(unescapeCode(line))
				if w.units > 0 && width <= w.limit && w.units+width > w.limit {
					w.flush()
				}
			}
			if text[i] == '`' && current.kind == "code" {
				w.pop()
				i++
				continue
			}
			if strings.HasPrefix(text[i:], "```") && current.kind == "pre" {
				w.pop()
				i += 3
				continue
			}
		default:
			if n := w.consumeMarkup(text, i); n > 0 {
				i += n
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(text[i:])
		if r != '\r' {
			raw := string(r)
			if !isCode && r != '\n' {
				raw = escapeText(raw)
			}
			w.put(raw, string(r))
		}
		i += size
	}
	w.flush()
	return w.chunks
}

func (w *chunkWriter) consumeMarkup(text string, i int) int {
	rest := text[i:]
	atLineStart := i == 0 || text[i-1] == '\n'
	if atLineStart && (strings.HasPrefix(rest, ">") || strings.HasPrefix(rest, "**>")) {
		if len(w.stack) > 0 && w.stack[0].kind == "quote" {
			w.output.WriteByte('>')
			return 1
		}
		if len(w.stack) == 0 {
			if strings.HasPrefix(rest, "**>") {
				w.push(marker{"**>", "||", "quote"})
				return 3
			}
			w.push(marker{">", "", "quote"})
			return 1
		}
	}
	if len(w.stack) > 0 && w.stack[0].kind == "quote" {
		if w.current().kind == "quote" && w.current().close == "||" &&
			strings.HasPrefix(rest, "||") && (len(rest) == 2 || rest[2] == '\n') {
			w.pop()
			return 2
		}
		if rest[0] == '\n' && (len(rest) == 1 || rest[1] != '>') {
			for len(w.stack) > 0 {
				w.pop()
			}
		}
	}
	if strings.HasPrefix(rest, "```") {
		if end := strings.IndexByte(rest, '\n'); end >= 3 && end <= 128 {
			w.push(marker{rest[:end+1], "\n```", "pre"})
			return end + 1
		}
	}
	if rest[0] == '[' || strings.HasPrefix(rest, "![") {
		if end, label := linkEnd(rest); end > 0 {
			var plain strings.Builder
			for _, part := range splitMarkup(label, DefaultMaxLength) {
				plain.WriteString(part.Text)
			}
			if utf16Len(plain.String()) <= w.limit && end <= telegramByteLimit-2048 {
				w.put(rest[:end], plain.String())
			} else {
				for _, r := range plain.String() {
					w.put(escapeText(string(r)), string(r))
				}
			}
			return end
		}
	}
	for _, delim := range []string{"__", "_", "*", "~", "||", "`"} {
		if !strings.HasPrefix(rest, delim) {
			continue
		}
		if w.current().kind == delim || delim == "`" && w.current().kind == "code" {
			w.pop()
		} else {
			kind := delim
			if delim == "`" {
				kind = "code"
			}
			w.push(marker{delim, delim, kind})
		}
		return len(delim)
	}
	return 0
}

func linkEnd(s string) (int, string) {
	start := 1
	if s[0] == '!' {
		start++
	}
	for i := start; i+1 < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '\n' || s[i] == '[' {
			return 0, ""
		}
		if s[i] != ']' || s[i+1] != '(' {
			continue
		}
		for j := i + 2; j < len(s); j++ {
			if s[j] == '\\' {
				j++
			} else if s[j] == ')' {
				return j + 1, s[start:i]
			} else if s[j] == '\n' {
				break
			}
		}
		return 0, ""
	}
	return 0, ""
}

func unescapeCode(s string) string {
	return strings.NewReplacer("\\`", "`", "\\\\", "\\").Replace(s)
}
