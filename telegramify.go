// Package telegramify converts standard (CommonMark/GFM) Markdown into the
// MarkdownV2 dialect understood by the Telegram Bot API.
//
// Telegram's MarkdownV2 supports only a small subset of Markdown and requires
// many ASCII punctuation characters to be backslash-escaped; getting this wrong
// makes the Bot API reject the message. This package handles the escaping and
// maps unsupported constructs (headings, tables, images, task lists, ...) onto
// the closest Telegram-compatible representation, so you can feed it arbitrary
// Markdown and send the result with parse_mode = "MarkdownV2".
//
// It is a Go port of the Python library telegramify-markdown
// (https://github.com/sudoskys/telegramify-markdown), covering the core
// Markdown -> MarkdownV2 string conversion and message splitting.
//
// Basic usage:
//
//	text := telegramify.Markdownify("# Title\n\nHello **world**!")
//	// send text with parse_mode = "MarkdownV2"
package telegramify

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Markdownify converts a Markdown string into Telegram MarkdownV2.
//
// The result is safe to send to the Telegram Bot API with
// parse_mode = "MarkdownV2". Rendering can be tuned with Options such as
// WithHeadingSymbols or WithCiteExpandable.
func Markdownify(markdown string, opts ...Option) string {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			spoilerExt,
		),
	)

	src := []byte(markdown)
	doc := md.Parser().Parse(text.NewReader(src))

	r := &renderer{cfg: cfg, source: src}
	return r.render(doc)
}
