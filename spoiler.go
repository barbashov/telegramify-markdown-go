package telegramify

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// kindSpoiler is the node kind for Telegram spoilers (||hidden||).
var kindSpoiler = gast.NewNodeKind("TelegramSpoiler")

// spoilerNode is an inline node representing a Telegram spoiler. It has no
// extra data; its children hold the spoilered inline content.
type spoilerNode struct {
	gast.BaseInline
}

func (n *spoilerNode) Kind() gast.NodeKind { return kindSpoiler }

func (n *spoilerNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// spoilerDelimiterProcessor matches paired `||` delimiters, mirroring the
// approach goldmark's strikethrough extension uses for `~~`.
type spoilerDelimiterProcessor struct{}

func (p *spoilerDelimiterProcessor) IsDelimiter(b byte) bool { return b == '|' }

func (p *spoilerDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *spoilerDelimiterProcessor) OnMatch(consumes int) gast.Node { return &spoilerNode{} }

var defaultSpoilerDelimiterProcessor = &spoilerDelimiterProcessor{}

// spoilerParser is an inline parser that recognises the `||` delimiter.
type spoilerParser struct{}

func (s *spoilerParser) Trigger() []byte { return []byte{'|'} }

func (s *spoilerParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	// Require exactly two pipes so a single `|` (e.g. a table cell separator
	// that slipped through, or ordinary text) is left untouched.
	node := parser.ScanDelimiter(line, before, 2, defaultSpoilerDelimiterProcessor)
	if node == nil || node.OriginalLength != 2 || before == '|' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (s *spoilerParser) CloseBlock(parent gast.Node, pc parser.Context) {}

// spoilerExtension registers the spoiler inline parser. It deliberately adds no
// renderer because this package walks the AST itself.
type spoilerExtension struct{}

// spoilerExt enables parsing of `||spoiler||` syntax.
var spoilerExt = &spoilerExtension{}

func (e *spoilerExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&spoilerParser{}, 499),
	))
}
