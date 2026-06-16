// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// markdownRenderer is the global markdown renderer with syntax highlighting
var markdownRenderer goldmark.Markdown

func init() {
	// Initialize goldmark with extensions and syntax highlighting
	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(
			// GitHub Flavored Markdown (tables, strikethrough, autolinks)
			extension.GFM,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
				highlighting.WithFormatOptions(
					html.WithLineNumbers(true),
					html.WithClasses(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
			gmhtml.WithXHTML(),
			// Allow raw HTML in markdown
			gmhtml.WithUnsafe(),
		),
	)
}

// RenderMarkdown converts markdown text to HTML
func RenderMarkdown(markdown string) template.HTML {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(markdown), &buf); err != nil {
		// On error, return escaped HTML
		return template.HTML("<pre>" + template.HTMLEscapeString(markdown) + "</pre>")
	}
	return template.HTML(buf.String())
}

// IsMarkdownSyntax checks if the syntax indicates markdown content
func IsMarkdownSyntax(syntax string) bool {
	syntaxLower := strings.ToLower(syntax)
	return syntaxLower == "markdown" || syntaxLower == "md"
}

// IsMarkdownFile checks if a filename indicates markdown content
func IsMarkdownFile(filename string) bool {
	filenameLower := strings.ToLower(filename)
	return strings.HasSuffix(filenameLower, ".md") ||
		strings.HasSuffix(filenameLower, ".markdown") ||
		strings.HasSuffix(filenameLower, ".mdown") ||
		strings.HasSuffix(filenameLower, ".mkd")
}

// IsMarkdownMimeType checks if MIME type indicates markdown
func IsMarkdownMimeType(mimeType string) bool {
	return mimeType == "text/markdown" ||
		mimeType == "text/x-markdown" ||
		mimeType == "text/plain; charset=utf-8" && strings.Contains(mimeType, "markdown")
}

