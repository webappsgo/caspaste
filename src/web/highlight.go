// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// DetectSyntaxFromFilename detects programming language from file extension
// Returns the lexer name or empty string if not detected
func DetectSyntaxFromFilename(filename string) string {
	if filename == "" {
		return ""
	}

	// Try Chroma's built-in filename matching
	l := lexers.Match(filename)
	if l != nil {
		return l.Config().Name
	}

	// Additional common extensions not in Chroma
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "plaintext"
	case ".log":
		return "plaintext"
	case ".cfg", ".conf", ".ini":
		return "INI"
	case ".env":
		return "Bash"
	case ".gitignore", ".dockerignore":
		return "plaintext"
	}

	return ""
}

func tryHighlight(source string, lexer string, theme string) template.HTML {
	// Determine lexer
	var l chroma.Lexer

	if lexer == "autodetect" || lexer == "" {
		// Auto-detect language from source code
		l = lexers.Analyse(source)
		if l == nil {
			// Couldn't detect, fallback to plaintext
			l = lexers.Get("plaintext")
		}
	} else {
		l = lexers.Get(lexer)
	}

	if l == nil {
		return template.HTML(source)
	}

	l = chroma.Coalesce(l)

	// Determine formatter
	f := html.New(
		html.Standalone(false),
		html.WithClasses(false),
		html.TabWidth(4),
		html.WithLineNumbers(true),
		html.WrapLongLines(true),
	)

	s := styles.Get(theme)

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return template.HTML(source)
	}

	// Format
	var buf bytes.Buffer

	err = f.Format(&buf, s, it)
	if err != nil {
		return template.HTML(source)
	}

	return template.HTML(buf.String())
}
