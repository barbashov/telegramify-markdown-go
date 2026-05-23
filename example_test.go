package telegramify_test

import (
	"fmt"

	telegramify "github.com/barbashov/telegramify-markdown-go"
)

func ExampleMarkdownify() {
	md := "## Shopping list\n\nDon't forget **milk** and `eggs`! Cost: $3.50."
	fmt.Println(telegramify.Markdownify(md))
	// Output:
	// ✏️ *__Shopping list__*
	//
	// Don't forget *milk* and `eggs`\! Cost: $3\.50\.
}

func ExampleMarkdownify_options() {
	out := telegramify.Markdownify("# Title", telegramify.WithHeadingSymbols("➤"))
	fmt.Println(out)
	// Output: ➤ *__Title__*
}

func ExampleSplit() {
	long := "first line\nsecond line\nthird line"
	for _, chunk := range telegramify.Split(long, 22) {
		fmt.Printf("%q\n", chunk)
	}
	// Output:
	// "first line\nsecond line"
	// "third line"
}
