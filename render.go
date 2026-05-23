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
		prefix = s + " "
	}
	if strings.TrimSpace(content) == "" {
		return strings.TrimSpace(prefix)
	}
	var open, close string
	switch {
	case level <= 2:
		open, close = "*__", "__*" // bold + underline
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
		s := escapeText(string(n.Value(r.source)))
		if n.HardLineBreak() || n.SoftLineBreak() {
			s += "\n"
		}
		return s
	case *ast.String:
		return escapeText(string(n.Value))
	case *ast.Emphasis:
		marker := "_" // level 1: italic
		if n.Level >= 2 {
			marker = "*" // level 2: bold
		}
		return marker + r.renderInlineChildren(n) + marker
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
		return r.renderLink(url, escapeText(url))
	case *ast.Image:
		return r.renderImage(n)
	case *east.TaskCheckBox:
		if n.IsChecked {
			return r.cfg.taskDone + " "
		}
		return r.cfg.taskTodo + " "
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
	label := r.cfg.imageSymbol
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
	code := strings.TrimRight(r.linesText(n), "\n")
	var b strings.Builder
	b.WriteString("```")
	b.WriteString(escapeCode(lang))
	b.WriteByte('\n')
	b.WriteString(escapeCode(code))
	b.WriteString("\n```")
	return b.String()
}

func (r *renderer) renderBlockquote(n *ast.Blockquote, depth int) string {
	content := r.renderBlocks(n, depth)
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
		var marker string
		switch {
		case n.IsOrdered():
			marker = escapeText(strconv.Itoa(index)+".") + " "
			index++
		case itemHasTaskCheckbox(item):
			marker = "" // the rendered checkbox symbol acts as the marker
		default:
			marker = escapeText(r.cfg.unorderedMarker) + " "
		}

		var textBlocks []string
		var nested []string
		for ic := item.FirstChild(); ic != nil; ic = ic.NextSibling() {
			if lst, ok := ic.(*ast.List); ok {
				nested = append(nested, r.renderList(lst, depth+1))
				continue
			}
			if s := r.renderBlock(ic, depth); strings.TrimSpace(s) != "" {
				textBlocks = append(textBlocks, s)
			}
		}

		cont := indent + strings.Repeat(" ", utf8.RuneCountInString(marker))
		for i, line := range strings.Split(strings.Join(textBlocks, "\n"), "\n") {
			if i == 0 {
				lines = append(lines, indent+marker+line)
			} else {
				lines = append(lines, cont+line)
			}
		}
		lines = append(lines, nested...)
	}
	return strings.Join(lines, "\n")
}

func (r *renderer) renderTable(n *east.Table) string {
	var rows [][]string
	for rowNode := n.FirstChild(); rowNode != nil; rowNode = rowNode.NextSibling() {
		var cells []string
		for cellNode := rowNode.FirstChild(); cellNode != nil; cellNode = cellNode.NextSibling() {
			cells = append(cells, escapeCode(r.plainInline(cellNode)))
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
			if w := utf8.RuneCountInString(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	b.WriteString("```\n")
	for ri, row := range rows {
		b.WriteByte('|')
		for i := 0; i < ncol; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			pad := widths[i] - utf8.RuneCountInString(cell)
			b.WriteByte(' ')
			b.WriteString(cell)
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
			b.Write(t.Segment.Value(r.source))
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
			b.Write(t.Value(r.source))
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
