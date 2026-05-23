package telegramify

// config holds the rendering options. It is populated from defaultConfig and
// then mutated by the functional Options passed to Markdownify.
type config struct {
	// headingSymbols holds the emoji/string prefix prepended to headings of
	// level 1..6 (index 0 is H1). Telegram has no native headings, so they are
	// emulated with a prefix plus bold/underline/italic styling.
	headingSymbols [6]string
	// imageSymbol is prepended to the link text rendered for images, since
	// Telegram messages cannot embed images via MarkdownV2.
	imageSymbol string
	// horizontalRule is emitted in place of a Markdown thematic break (---).
	horizontalRule string
	// taskDone / taskTodo are the markers used for completed / pending GFM task
	// list items.
	taskDone string
	taskTodo string
	// unorderedMarker is the bullet used for unordered list items.
	unorderedMarker string
	// citeExpandable controls whether long blockquotes are rendered as Telegram
	// expandable blockquotes.
	citeExpandable bool
	// expandableThreshold is the blockquote length, in UTF-16 code units, above
	// which a blockquote becomes expandable (only when citeExpandable is true).
	expandableThreshold int
}

func defaultConfig() *config {
	return &config{
		headingSymbols:      [6]string{"\U0001F4CC", "✏️", "\U0001F4DA", "\U0001F516", "", ""},
		imageSymbol:         "\U0001F5BC",
		horizontalRule:      "————————",
		taskDone:            "✅",
		taskTodo:            "☐",
		unorderedMarker:     "•",
		citeExpandable:      true,
		expandableThreshold: 200,
	}
}

// Option configures how Markdownify renders Markdown into Telegram MarkdownV2.
type Option func(*config)

// WithHeadingSymbols overrides the prefixes used for headings. Up to six
// symbols may be supplied, mapping to heading levels 1..6; missing levels keep
// their default. Pass an empty string to render a level with no prefix.
func WithHeadingSymbols(symbols ...string) Option {
	return func(c *config) {
		for i := 0; i < len(symbols) && i < len(c.headingSymbols); i++ {
			c.headingSymbols[i] = symbols[i]
		}
	}
}

// WithImageSymbol sets the prefix used for the link that replaces an image.
func WithImageSymbol(symbol string) Option {
	return func(c *config) { c.imageSymbol = symbol }
}

// WithHorizontalRule sets the string emitted for a Markdown thematic break.
func WithHorizontalRule(rule string) Option {
	return func(c *config) { c.horizontalRule = rule }
}

// WithTaskSymbols sets the markers used for completed and pending task list
// items.
func WithTaskSymbols(done, todo string) Option {
	return func(c *config) {
		c.taskDone = done
		c.taskTodo = todo
	}
}

// WithUnorderedMarker sets the bullet used for unordered list items.
func WithUnorderedMarker(marker string) Option {
	return func(c *config) { c.unorderedMarker = marker }
}

// WithCiteExpandable controls whether long blockquotes render as Telegram
// expandable blockquotes.
func WithCiteExpandable(expandable bool) Option {
	return func(c *config) { c.citeExpandable = expandable }
}

// WithExpandableThreshold sets the blockquote length, in UTF-16 code units,
// above which a blockquote becomes expandable. Has no effect unless
// WithCiteExpandable is enabled (the default).
func WithExpandableThreshold(units int) Option {
	return func(c *config) { c.expandableThreshold = units }
}
