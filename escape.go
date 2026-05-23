package telegramify

import "strings"

// textEscapeChars are the characters that must be backslash-escaped in normal
// Telegram MarkdownV2 text.
//
// See https://core.telegram.org/bots/api#markdownv2-style.
var textEscapeChars = map[rune]struct{}{
	'_': {}, '*': {}, '[': {}, ']': {}, '(': {}, ')': {}, '~': {}, '`': {},
	'>': {}, '#': {}, '+': {}, '-': {}, '=': {}, '|': {}, '{': {}, '}': {},
	'.': {}, '!': {}, '\\': {},
}

// escapeText escapes all MarkdownV2 special characters in regular text.
func escapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/8 + 1)
	for _, r := range s {
		if _, ok := textEscapeChars[r]; ok {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// escapeCode escapes the characters that are special inside MarkdownV2 `code`
// and ```pre``` entities. Only backtick and backslash need escaping there.
func escapeCode(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		if r == '`' || r == '\\' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// escapeURL escapes the characters that are special inside the URL part of a
// MarkdownV2 inline link, i.e. `[text](url)`. Only `)` and `\` need escaping.
func escapeURL(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		if r == ')' || r == '\\' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// utf16Len returns the length of s measured in UTF-16 code units, which is the
// unit Telegram uses for message length and entity offsets. Runes outside the
// Basic Multilingual Plane (e.g. most emoji) require a surrogate pair and count
// as two units.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}
