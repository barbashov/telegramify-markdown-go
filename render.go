package telegramify

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// renderer walks a parsed goldmark AST and produces a Telegram MarkdownV2
// string. Block-level nodes return their rendered string; inline-level nodes
// append to a strings.Builder via renderInlineChildren.
type renderer struct {
	cfg    *config
	source []byte
	// inItalic / inBold track open emphasis entities: Telegram cannot nest
	// entities of the same type, so redundant inner markers are dropped.
	inItalic bool
	inBold   bool
}

// render renders the document node and returns the trimmed MarkdownV2 output.
func (r *renderer) render(doc ast.Node) string {
	return strings.TrimSpace(r.renderBlocks(doc, 0))
}

// renderBlocks renders the block-level children of parent, joining them with a
// blank line. depth is the current list nesting depth (used for indentation).
func (r *renderer) renderBlocks(parent ast.Node, depth int) string {
	var parts []string
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		s := r.renderBlock(c, depth)
		if strings.TrimSpace(s) == "" {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n\n")
}

func (r *renderer) renderBlock(n ast.Node, depth int) string {
	switch n := n.(type) {
	case *ast.Heading:
		return r.renderHeading(n)
	case *ast.Paragraph:
		return r.renderInlineChildren(n)
	case *ast.TextBlock:
		return r.renderInlineChildren(n)
	case *ast.Blockquote:
		return r.renderBlockquote(n, depth)
	case *ast.List:
		return r.renderList(n, depth)
	case *ast.FencedCodeBlock:
		return r.renderCode(string(n.Language(r.source)), n)
	case *ast.CodeBlock:
		return r.renderCode("", n)
	case *ast.ThematicBreak:
		return escapeText(r.cfg.horizontalRule)
	case *east.Table:
		return r.renderTable(n)
	case *ast.HTMLBlock:
		return escapeText(strings.TrimRight(r.linesText(n), "\n"))
	default:
		return r.renderBlocks(n, depth)
	}
}

func (r *renderer) renderHeading(n *ast.Heading) string {
	content := r.renderInlineChildren(n)
	level := n.Level
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	prefix := ""
	if s := r.cfg.headingSymbols[level-1]; s != "" {
		prefix = escapeText(s) + " "
	}
	if strings.TrimSpace(content) == "" {
		return strings.TrimSpace(prefix)
	}
	var open, close string
	switch {
	case level <= 2:
		open, close = "*__", "__*" // bold + underline
		if strings.HasSuffix(content, "_") {
			// Telegram matches __ greedily; \r disambiguates an italic close
			// immediately followed by the underline close.
			close = "\r__*"
		}
	case level <= 4:
		open, close = "*", "*" // bold
	default:
		open, close = "_", "_" // italic
	}
	return prefix + open + content + close
}

func (r *renderer) renderInlineChildren(n ast.Node) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b.WriteString(r.renderInline(c))
	}
	return b.String()
}

func (r *renderer) renderInline(n ast.Node) string {
	switch n := n.(type) {
	case *ast.Text:
		value := n.Value(r.source)
		var s string
		if n.IsRaw() {
			s = escapeText(string(value))
		} else {
			s = escapeText(resolveRawText(value))
		}
		if n.HardLineBreak() || n.SoftLineBreak() {
			s += "\n"
		}
		return s
	case *ast.String:
		return escapeText(string(n.Value))
	case *ast.Emphasis:
		if n.Level >= 2 { // bold
			if r.inBold {
				return r.renderInlineChildren(n)
			}
			r.inBold = true
			s := "*" + r.renderInlineChildren(n) + "*"
			r.inBold = false
			return s
		}
		if r.inItalic {
			return r.renderInlineChildren(n)
		}
		r.inItalic = true
		s := "_" + r.renderInlineChildren(n) + "_"
		r.inItalic = false
		return s
	case *east.Strikethrough:
		return "~" + r.renderInlineChildren(n) + "~"
	case *spoilerNode:
		return "||" + r.renderInlineChildren(n) + "||"
	case *ast.CodeSpan:
		return "`" + escapeCode(r.codeSpanText(n)) + "`"
	case *ast.Link:
		return r.renderLink(string(n.Destination), r.renderInlineChildren(n))
	case *ast.AutoLink:
		url := string(n.URL(r.source))
		dest := url
		if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(dest, "mailto:") {
			dest = "mailto:" + dest
		}
		return r.renderLink(dest, escapeText(url))
	case *ast.Image:
		return r.renderImage(n)
	case *east.TaskCheckBox:
		if n.IsChecked {
			return escapeText(r.cfg.taskDone) + " "
		}
		return escapeText(r.cfg.taskTodo) + " "
	case *ast.RawHTML:
		return escapeText(r.rawHTMLText(n))
	default:
		return r.renderInlineChildren(n)
	}
}

func (r *renderer) renderLink(dest, text string) string {
	if strings.TrimSpace(text) == "" {
		text = escapeText(dest)
	}
	return "[" + text + "](" + escapeURL(dest) + ")"
}

func (r *renderer) renderImage(n *ast.Image) string {
	dest := string(n.Destination)
	alt := r.renderInlineChildren(n)
	// Telegram custom emoji are written with image syntax; keep it intact.
	if strings.HasPrefix(dest, "tg://emoji") {
		return "![" + alt + "](" + escapeURL(dest) + ")"
	}
	label := escapeText(r.cfg.imageSymbol)
	if strings.TrimSpace(alt) != "" {
		if label != "" {
			label += " " + alt
		} else {
			label = alt
		}
	}
	if strings.TrimSpace(label) == "" {
		label = escapeText(dest)
	}
	return "[" + label + "](" + escapeURL(dest) + ")"
}

func (r *renderer) renderCode(lang string, n ast.Node) string {
	// Trim only the structural newline of the last line; further trailing
	// blank lines are code content.
	code := strings.TrimSuffix(r.linesText(n), "\n")
	var b strings.Builder
	b.WriteString("```")
	b.WriteString(escapeCode(lang))
	b.WriteByte('\n')
	b.WriteString(escapeCode(code))
	b.WriteString("\n```")
	return b.String()
}

func (r *renderer) renderBlockquote(n *ast.Blockquote, depth int) string {
	content := r.renderQuoteBlocks(n, depth)
	lines := strings.Split(content, "\n")
	expandable := r.cfg.citeExpandable && utf16Len(content) > r.cfg.expandableThreshold
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == 0 && expandable {
			b.WriteString("**>")
		} else {
			b.WriteByte('>')
		}
		b.WriteString(line)
	}
	if expandable {
		b.WriteString("||")
	}
	return b.String()
}

// renderQuoteBlocks renders blockquote children, flattening nested
// blockquotes into the parent: Telegram forbids nested blockquote entities.
func (r *renderer) renderQuoteBlocks(parent ast.Node, depth int) string {
	var parts []string
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		var s string
		if bq, ok := c.(*ast.Blockquote); ok {
			s = r.renderQuoteBlocks(bq, depth)
		} else {
			s = r.renderBlock(c, depth)
		}
		if strings.TrimSpace(s) == "" {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n\n")
}

func (r *renderer) renderList(n *ast.List, depth int) string {
	indent := strings.Repeat("  ", depth)
	var lines []string
	index := n.Start
	if index == 0 {
		index = 1
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		var marker, visibleMarker string
		switch {
		case n.IsOrdered():
			raw := strconv.Itoa(index) + "."
			marker = escapeText(raw) + " "
			visibleMarker = raw + " "
			index++
		case itemHasTaskCheckbox(item):
			// The rendered checkbox symbol acts as the marker.
		default:
			marker = escapeText(r.cfg.unorderedMarker) + " "
			visibleMarker = r.cfg.unorderedMarker + " "
		}

		// Continuation lines align under the visible marker width, which
		// excludes escape backslashes Telegram strips when rendering.
		cont := indent + strings.Repeat(" ", utf8.RuneCountInString(visibleMarker))
		first := true
		for ic := item.FirstChild(); ic != nil; ic = ic.NextSibling() {
			if lst, ok := ic.(*ast.List); ok {
				if s := r.renderList(lst, depth+1); strings.TrimSpace(s) != "" {
					lines = append(lines, strings.Split(s, "\n")...)
					first = false
				}
				continue
			}
			s := r.renderBlock(ic, depth)
			if strings.TrimSpace(s) == "" {
				continue
			}
			if _, isQuote := ic.(*ast.Blockquote); isQuote {
				// Telegram only recognises '>' at the start of a line, so a
				// blockquote cannot be indented into the item.
				lines = append(lines, strings.Split(s, "\n")...)
				first = false
				continue
			}
			for _, line := range strings.Split(s, "\n") {
				if first {
					lines = append(lines, indent+marker+line)
					first = false
				} else {
					lines = append(lines, cont+line)
				}
			}
		}
		if first { // empty item
			lines = append(lines, indent+strings.TrimRight(marker, " "))
		}
	}
	return strings.Join(lines, "\n")
}

func (r *renderer) renderTable(n *east.Table) string {
	// Widths and padding are computed on the raw text (what Telegram shows
	// after stripping escapes); the escaped text is what gets written.
	type tableCell struct {
		raw, escaped string
	}
	var rows [][]tableCell
	for rowNode := n.FirstChild(); rowNode != nil; rowNode = rowNode.NextSibling() {
		var cells []tableCell
		for cellNode := rowNode.FirstChild(); cellNode != nil; cellNode = cellNode.NextSibling() {
			raw := r.plainInline(cellNode)
			cells = append(cells, tableCell{raw: raw, escaped: escapeCode(raw)})
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return ""
	}
	ncol := 0
	for _, row := range rows {
		if len(row) > ncol {
			ncol = len(row)
		}
	}
	widths := make([]int, ncol)
	for _, row := range rows {
		for i, cell := range row {
			if w := utf8.RuneCountInString(cell.raw); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	b.WriteString("```\n")
	for ri, row := range rows {
		b.WriteByte('|')
		for i := 0; i < ncol; i++ {
			var cell tableCell
			if i < len(row) {
				cell = row[i]
			}
			pad := widths[i] - utf8.RuneCountInString(cell.raw)
			b.WriteByte(' ')
			b.WriteString(cell.escaped)
			b.WriteString(strings.Repeat(" ", pad))
			b.WriteString(" |")
		}
		b.WriteByte('\n')
		if ri == 0 { // header separator
			b.WriteByte('|')
			for i := 0; i < ncol; i++ {
				b.WriteString(strings.Repeat("-", widths[i]+2))
				b.WriteByte('|')
			}
			b.WriteByte('\n')
		}
	}
	b.WriteString("```")
	return b.String()
}

// itemHasTaskCheckbox reports whether the first inline child of a list item is
// a GFM task checkbox.
func itemHasTaskCheckbox(item *ast.ListItem) bool {
	first := item.FirstChild()
	if first == nil {
		return false
	}
	fi := first.FirstChild()
	if fi == nil {
		return false
	}
	_, ok := fi.(*east.TaskCheckBox)
	return ok
}

// codeSpanText extracts the literal text of an inline code span.
func (r *renderer) codeSpanText(n *ast.CodeSpan) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			v := t.Segment.Value(r.source)
			// CommonMark: line endings inside a code span become spaces.
			if len(v) > 0 && v[len(v)-1] == '\n' {
				b.Write(v[:len(v)-1])
				b.WriteByte(' ')
			} else {
				b.Write(v)
			}
		case *ast.String:
			b.Write(t.Value)
		}
	}
	return b.String()
}

// rawHTMLText returns the literal source of an inline raw-HTML node.
func (r *renderer) rawHTMLText(n *ast.RawHTML) string {
	var b strings.Builder
	for i := 0; i < n.Segments.Len(); i++ {
		seg := n.Segments.At(i)
		b.Write(seg.Value(r.source))
	}
	return b.String()
}

// linesText returns the concatenated source lines of a block node (code blocks,
// HTML blocks).
func (r *renderer) linesText(n ast.Node) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(r.source))
	}
	return b.String()
}

// plainInline returns the unformatted text of an inline subtree, used for table
// cells which are rendered inside a monospace block.
func (r *renderer) plainInline(n ast.Node) string {
	var b strings.Builder
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := node.(type) {
		case *ast.Text:
			if t.IsRaw() {
				b.Write(t.Value(r.source))
			} else {
				b.WriteString(resolveRawText(t.Value(r.source)))
			}
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		case *ast.CodeSpan:
			b.WriteString(r.codeSpanText(t))
			return ast.WalkSkipChildren, nil
		case *ast.AutoLink:
			b.Write(t.URL(r.source))
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}
