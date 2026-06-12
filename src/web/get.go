// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"encoding/base64"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/casjay-forks/caspaste/src/lineend"
	"github.com/casjay-forks/caspaste/src/netshare"
)

// File type detection helpers
func isImageMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func isVideoMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}

func isAudioMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "audio/")
}

func isPDFMimeType(mimeType string) bool {
	return mimeType == "application/pdf"
}

func isTextMimeType(mimeType string) bool {
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	// Common text-based MIME types
	textTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/ecmascript",
		"application/x-sh",
		"application/x-csh",
		"application/x-python",
		"application/x-ruby",
		"application/x-perl",
		"application/x-php",
	}
	for _, t := range textTypes {
		if mimeType == t {
			return true
		}
	}
	return false
}

// isTextFileExtension checks if filename has a known text/code extension
func isTextFileExtension(filename string) bool {
	if filename == "" {
		return false
	}
	ext := strings.ToLower(filename)
	// Get the extension part
	if idx := strings.LastIndex(ext, "."); idx >= 0 {
		ext = ext[idx:]
	} else {
		return false
	}

	// Common text/code file extensions
	textExts := map[string]bool{
		// Programming languages
		".py": true, ".rb": true, ".pl": true, ".php": true,
		".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".go": true, ".rs": true, ".java": true, ".kt": true,
		".c": true, ".cpp": true, ".cc": true, ".h": true, ".hpp": true,
		".cs": true, ".swift": true, ".m": true, ".mm": true,
		".scala": true, ".clj": true, ".ex": true, ".exs": true,
		".hs": true, ".ml": true, ".fs": true, ".r": true,
		".lua": true, ".vim": true, ".el": true, ".lisp": true,
		".asm": true, ".s": true, ".d": true, ".nim": true,
		".cr": true, ".v": true, ".zig": true, ".odin": true,
		// Web
		".html": true, ".htm": true, ".css": true, ".scss": true,
		".sass": true, ".less": true, ".vue": true, ".svelte": true,
		// Data/Config
		".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".toml": true, ".ini": true, ".cfg": true, ".conf": true,
		".env": true, ".properties": true,
		// Shell/Script
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".ps1": true, ".bat": true, ".cmd": true,
		// Text/Doc
		".txt": true, ".md": true, ".markdown": true, ".rst": true,
		".tex": true, ".org": true, ".adoc": true,
		// SQL/Database
		".sql": true, ".graphql": true, ".gql": true,
		// Build/Make
		".make": true, ".cmake": true, ".gradle": true,
		// Other
		".diff": true, ".patch": true, ".log": true,
		".csv": true, ".tsv": true,
		".dockerfile": true, ".containerfile": true,
		".gitignore": true, ".dockerignore": true,
	}
	return textExts[ext]
}

type pasteTmpl struct {
	ID         string
	Title      string
	Body       template.HTML
	Syntax     string
	CreateTime int64
	DeleteTime int64
	OneUse     bool

	LineEnd       string
	CreateTimeStr string
	DeleteTimeStr string

	Author      string
	AuthorEmail string
	AuthorURL   string

	// File upload fields
	IsFile   bool
	FileName string
	MimeType string
	FileSize int

	// File type flags for template rendering
	IsImage    bool
	IsVideo    bool
	IsAudio    bool
	IsPDF      bool
	IsText     bool
	IsMarkdown bool

	// Data URL for embedding media (images, video, audio)
	// Using template.URL to mark as safe for embedding
	MediaDataURL template.URL

	User     *AuthUser
	Language string
	Theme    func(string) string

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool

	Translate func(string, ...interface{}) template.HTML
}

type pasteContinueTmpl struct {
	ID       string
	User     *AuthUser
	Language string
	Theme    func(string) string

	// CSRF token for form protection per AI.md PART 11
	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool

	Translate func(string, ...interface{}) template.HTML
}

func (data *Data) handleGetPaste(rw http.ResponseWriter, req *http.Request) error {
	// Check rate limit
	err := data.RateLimitGet.CheckAndUse(netshare.GetClientAddr(req))
	if err != nil {
		return err
	}

	// Get paste ID
	pasteID := string([]rune(req.URL.Path)[1:])

	// Read DB
	paste, err := data.DB.PasteGet(pasteID)
	if err != nil {
		return err
	}

	// If "one use" paste
	if paste.OneUse {
		// If continue button not pressed
		req.ParseForm()

		if req.PostForm.Get("oneUseContinue") != "true" {
			tmplData := pasteContinueTmpl{
				ID:            paste.ID,
				User:          GetAuthUser(req.Context()),
				Language:      getCookie(req, "lang"),
				Theme:         data.getThemeFunc(req),
				CSRFToken:     GetCSRFToken(req, 32),
				UnreadCount:   0,
				Notifications: nil,
				ShowLogin:     data.ShowLogin,
				Translate:     data.Locales.findLocale(req).translate,
			}

			return data.PasteContinue.Execute(rw, tmplData)
		}

		// If continue button pressed delete paste
		err = data.DB.PasteDelete(pasteID)
		if err != nil {
			return err
		}
	}

	// Prepare template data
	createTime := time.Unix(paste.CreateTime, 0).UTC()
	deleteTime := time.Unix(paste.DeleteTime, 0).UTC()

	// Determine body content based on whether this is a file upload
	var bodyContent string
	var fileSize int
	var bodyHTML template.HTML
	var mediaDataURL template.URL
	var isImage, isVideo, isAudio, isPDF, isText bool

	// Detect if content is markdown
	var isMarkdown bool

	if paste.IsFile {
		// File upload: try to decode base64, fall back to raw for legacy data
		var base64Data string
		fileData, err := base64.StdEncoding.DecodeString(paste.Body)
		if err != nil {
			// Legacy data stored without base64 encoding - use as-is
			bodyContent = paste.Body
			fileSize = len(paste.Body)
			base64Data = base64.StdEncoding.EncodeToString([]byte(paste.Body))
		} else {
			bodyContent = string(fileData)
			fileSize = len(fileData)
			base64Data = paste.Body
		}

		// Detect file type from MIME type and extension
		mimeType := paste.MimeType
		isImage = isImageMimeType(mimeType)
		isVideo = isVideoMimeType(mimeType)
		isAudio = isAudioMimeType(mimeType)
		isPDF = isPDFMimeType(mimeType)
		// Check text by MIME type or by file extension (for application/octet-stream)
		isText = isTextMimeType(mimeType) || isTextFileExtension(paste.FileName)

		// Check for markdown content
		isMarkdown = IsMarkdownMimeType(mimeType) || IsMarkdownFile(paste.FileName) || IsMarkdownSyntax(paste.Syntax)

		// For media files, create data URL for embedding
		if isImage || isVideo || isAudio || isPDF {
			mediaDataURL = template.URL("data:" + mimeType + ";base64," + base64Data)
			// Don't syntax highlight media - body will be empty
			bodyHTML = ""
		} else if isMarkdown {
			// Render markdown to HTML
			bodyHTML = RenderMarkdown(bodyContent)
		} else if isText {
			// Detect syntax from filename if not explicitly set
			syntax := paste.Syntax
			if syntax == "" || syntax == "plaintext" {
				if detected := DetectSyntaxFromFilename(paste.FileName); detected != "" {
					syntax = detected
				}
			}
			// Text files can be syntax highlighted
			bodyHTML = data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight(bodyContent, syntax)
		} else {
			// Binary files - show file info, don't try to display content
			bodyHTML = ""
		}
	} else {
		bodyContent = paste.Body

		// Check for markdown content by syntax
		isMarkdown = IsMarkdownSyntax(paste.Syntax)

		if isMarkdown {
			// Render markdown to HTML
			bodyHTML = RenderMarkdown(bodyContent)
		} else {
			bodyHTML = data.Themes.findTheme(req, data.UiDefaultTheme).tryHighlight(bodyContent, paste.Syntax)
		}
	}

	tmplData := pasteTmpl{
		ID:         paste.ID,
		Title:      paste.Title,
		Body:       bodyHTML,
		Syntax:     paste.Syntax,
		CreateTime: paste.CreateTime,
		DeleteTime: paste.DeleteTime,
		OneUse:     paste.OneUse,

		CreateTimeStr: createTime.Format("Mon, 02 Jan 2006 15:04:05 -0700"),
		DeleteTimeStr: deleteTime.Format("Mon, 02 Jan 2006 15:04:05 -0700"),

		Author:      paste.Author,
		AuthorEmail: paste.AuthorEmail,
		AuthorURL:   paste.AuthorURL,

		IsFile:       paste.IsFile,
		FileName:     paste.FileName,
		MimeType:     paste.MimeType,
		FileSize:     fileSize,
		IsImage:      isImage,
		IsVideo:      isVideo,
		IsAudio:      isAudio,
		IsPDF:        isPDF,
		IsText:       isText,
		IsMarkdown:   isMarkdown,
		MediaDataURL: mediaDataURL,

		User:          GetAuthUser(req.Context()),
		Language:      getCookie(req, "lang"),
		Theme:         data.getThemeFunc(req),
		CSRFToken:     data.buildCSRFToken(req),
		UnreadCount:   0,
		Notifications: nil,
		ShowLogin:     data.ShowLogin,
		Translate:     data.Locales.findLocale(req).translate,
	}

	// Get body line end (only for text content)
	if !paste.IsFile || isText {
		switch lineend.GetLineEnd(bodyContent) {
		case "\r\n":
			tmplData.LineEnd = "CRLF"
		case "\r":
			tmplData.LineEnd = "CR"
		default:
			tmplData.LineEnd = "LF"
		}
	}

	// Show paste
	return data.PastePage.Execute(rw, tmplData)
}
