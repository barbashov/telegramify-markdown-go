package telegramify

import (
	"strings"

	"github.com/yuin/goldmark/util"
)

// resolveRawText interprets CommonMark backslash escapes and HTML entity
// references in raw source text. goldmark leaves both unprocessed in Text
// nodes and expects the renderer to resolve them (its HTML writer does the
// same at render time).
func resolveRawText(src []byte) string {
	var b strings.Builder
	b.Grow(len(src))
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '\\' && i+1 < len(src) && util.IsPunct(src[i+1]) {
			b.WriteByte(src[i+1])
			i++
			continue
		}
		if c == '&' {
			if s, size := resolveEntity(src[i:]); size > 0 {
				b.WriteString(s)
				i += size - 1
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

// resolveEntity decodes one HTML entity reference at the start of src (which
// begins with '&'). It returns the decoded text and the number of source
// bytes consumed, or ("", 0) if src does not start with a valid entity.
func resolveEntity(src []byte) (string, int) {
	if len(src) < 3 {
		return "", 0
	}
	if src[1] == '#' {
		i := 2
		base := 10
		if i < len(src) && (src[i] == 'x' || src[i] == 'X') {
			base = 16
			i++
		}
		start := i
		v := 0
		for ; i < len(src) && i-start < 7; i++ {
			d := digitValue(src[i], base)
			if d < 0 {
				break
			}
			v = v*base + d
		}
		if i == start || i >= len(src) || src[i] != ';' {
			return "", 0
		}
		return string(util.ToValidRune(rune(v))), i + 1
	}
	i := 1
	for ; i < len(src) && util.IsAlphaNumeric(src[i]); i++ {
	}
	if i == 1 || i >= len(src) || src[i] != ';' {
		return "", 0
	}
	if e, ok := util.LookUpHTML5EntityByName(string(src[1:i])); ok {
		return string(e.Characters), i + 1
	}
	return "", 0
}

func digitValue(c byte, base int) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case base == 16 && c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case base == 16 && c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
