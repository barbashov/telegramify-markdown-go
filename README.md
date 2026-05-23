# telegramify-markdown-go

[![Go Reference](https://pkg.go.dev/badge/github.com/barbashov/telegramify-markdown-go.svg)](https://pkg.go.dev/github.com/barbashov/telegramify-markdown-go)
[![test](https://github.com/barbashov/telegramify-markdown-go/actions/workflows/test.yml/badge.svg)](https://github.com/barbashov/telegramify-markdown-go/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/barbashov/telegramify-markdown-go)](https://goreportcard.com/report/github.com/barbashov/telegramify-markdown-go)

Convert standard Markdown into the **MarkdownV2** dialect understood by the
[Telegram Bot API](https://core.telegram.org/bots/api#markdownv2-style).

Telegram's MarkdownV2 supports only a small subset of Markdown and requires many
ASCII punctuation characters (``_ * [ ] ( ) ~ ` > # + - = | { } . !``) to be
backslash-escaped — getting any of it wrong makes the Bot API reject your
message with `400 Bad Request`. This library takes arbitrary Markdown, escapes
it correctly, and maps unsupported constructs (headings, tables, images, task
lists, …) onto the closest Telegram-compatible representation, so you can just
send the result with `parse_mode = "MarkdownV2"`.

This is a Go port of the Python library
[telegramify-markdown](https://github.com/sudoskys/telegramify-markdown),
covering its core Markdown → MarkdownV2 string conversion plus message
splitting. Parsing is done with [goldmark](https://github.com/yuin/goldmark).

## Install

```sh
go get github.com/barbashov/telegramify-markdown-go
```

Requires Go 1.22+.

## Quick start

```go
package main

import (
	"fmt"

	telegramify "github.com/barbashov/telegramify-markdown-go"
)

func main() {
	md := "## Shopping list\n\nDon't forget **milk** and `eggs`! Cost: $3.50."
	out := telegramify.Markdownify(md)
	fmt.Println(out)
	// ✏️ *__Shopping list__*
	//
	// Don't forget *milk* and `eggs`\! Cost: $3\.50\.
}
```

Then send it with your Telegram bot library of choice using
`parse_mode = "MarkdownV2"`. For example, with
[go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api):

```go
msg := tgbotapi.NewMessage(chatID, telegramify.Markdownify(text))
msg.ParseMode = tgbotapi.ModeMarkdownV2
bot.Send(msg)
```

## Splitting long messages

Telegram rejects messages longer than 4096 UTF-16 code units. `Split` breaks a
rendered string into chunks that each fit, cutting at newline boundaries so
markup is not broken across chunks (as long as each block is itself under the
limit):

```go
out := telegramify.Markdownify(longText)
for _, chunk := range telegramify.Split(out, telegramify.DefaultMaxLength) {
	msg := tgbotapi.NewMessage(chatID, chunk)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	bot.Send(msg)
}
```

`Split` measures length in **UTF-16 code units** — the same unit Telegram uses —
so emoji and other astral-plane characters are counted correctly and never cut
mid-rune.

## Supported Markdown

| Markdown | Rendered as |
| --- | --- |
| `**bold**` | `*bold*` |
| `_italic_` / `*italic*` | `_italic_` |
| `~~strike~~` | `~strike~` |
| `` `code` `` | `` `code` `` (only `` ` `` and `\` escaped inside) |
| ```` ```lang … ``` ```` | fenced code block, language preserved |
| `[text](url)` | `[text](url)` (only `)` and `\` escaped in the URL) |
| `![alt](url)` | link with an image prefix, e.g. `[🖼 alt](url)` |
| `![👍](tg://emoji?id=…)` | Telegram custom emoji (kept as `![…](…)`) |
| `# H1` … `###### H6` | emoji prefix + bold/underline/italic styling per level |
| `- item` / `1. item` | `• item` / `1\. item` (with nesting) |
| `- [x] done` | task list with ✅ / ☐ markers |
| `> quote` | blockquote; long quotes become **expandable** |
| tables | aligned monospace block |
| `---` | a horizontal-rule string (`————————`) |
| `\|\|spoiler\|\|` | `\|\|spoiler\|\|` (Telegram spoiler) |

Bare URLs in text are **not** auto-linkified — they are left as plain (escaped)
text, matching how most chat content is authored.

## Configuration

`Markdownify` accepts functional options:

```go
out := telegramify.Markdownify(md,
	telegramify.WithHeadingSymbols("➤"),          // prefixes for H1..H6
	telegramify.WithImageSymbol("🏞"),             // prefix for image links
	telegramify.WithHorizontalRule("──────"),     // thematic break string
	telegramify.WithTaskSymbols("✔", "▢"),        // done / todo markers
	telegramify.WithUnorderedMarker("·"),         // bullet for unordered lists
	telegramify.WithCiteExpandable(false),        // disable expandable blockquotes
	telegramify.WithExpandableThreshold(120),     // length (UTF-16) to expand at
)
```

## Scope

This port focuses on the most widely used part of the original library: direct
Markdown → MarkdownV2 **string** conversion (Python's `markdownify`) and message
splitting. The Python library's entity-based output (`convert`) and the async
pipeline (`telegramify` with code-file extraction and Mermaid image rendering)
are out of scope for this release.

## License

[MIT](LICENSE). Port of [telegramify-markdown](https://github.com/sudoskys/telegramify-markdown)
(© 2024 Jasmine), which is also MIT-licensed.
