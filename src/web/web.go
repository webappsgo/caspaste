// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"embed"
	"html/template"
	"net/http"
	"os"
	"strings"
	textTemplate "text/template"
	"time"

	chromaLexers "github.com/alecthomas/chroma/v2/lexers"

	"github.com/casjay-forks/caspaste/src/caspasswd"
	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/logger"
	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/storage"
)

//go:embed data/*
var embFS embed.FS

type Data struct {
	DB  storage.DB
	Log logger.Logger

	RateLimitNew *netshare.RateLimitSystem
	RateLimitGet *netshare.RateLimitSystem

	Lexers      []string
	Locales     Locales
	LocalesList LocalesList
	Themes      Themes
	ThemesList  ThemesList

	StyleCSS       *textTemplate.Template
	ErrorPage      *template.Template
	Main           *template.Template
	MainJS         *[]byte
	ToastJS        *[]byte
	NavJS          *[]byte
	SettingsJS     *[]byte
	HistoryJS      *textTemplate.Template
	CodeJS         *textTemplate.Template
	PastePage      *template.Template
	PasteJS        *textTemplate.Template
	PasteContinue  *template.Template
	Settings       *template.Template
	ListPage       *template.Template
	About          *template.Template
	TermsOfUse     *template.Template
	Authors        *template.Template
	License        *template.Template
	SourceCodePage *template.Template
	SecurityPolicy *template.Template

	HealthzPage *template.Template

	Docs            *template.Template
	DocsApiV1       *template.Template
	DocsLibraries   *template.Template
	DocsCustomize   *template.Template
	DocsCliExamples *template.Template

	EmbeddedPage     *template.Template
	EmbeddedHelpPage *template.Template
	Login            *template.Template
	StubPage         *template.Template

	Version     string
	Mode        string
	BuildCommit string
	BuildDate   string

	// DataDir is the application data directory path; used for disk health check
	DataDir string

	ServerTagline     string
	ServerDescription string

	TitleMaxLen int
	BodyMaxLen  int
	MaxLifeTime int64

	ServerAbout      string
	ServerRules      string
	ServerTermsExist bool
	ServerTermsOfUse string
	SecurityTxt      string

	// Server info
	FQDN        string
	ServerTitle string
	AdminName   string
	AdminMail   string

	// Security contact
	SecurityContactEmail string
	SecurityContactName  string

	// Robots
	SiteRobotsAllow      string
	SiteRobotsDeny       string
	SiteRobotsAgentsDeny []string

	// Branding
	Logo    string
	Favicon string

	// true = open/public (no auth), false = auth required
	Public        bool
	CasPasswdFile string

	// Brute force protection for login (5 attempts, 15 min lockout per AI.md)
	BruteForce *caspasswd.BruteForceProtection

	UiDefaultLifeTime string
	UiDefaultTheme    string

	// ShowLogin controls whether the login link appears in the nav.
	// True when multi-user support is enabled (i.e., accounts exist to log into).
	ShowLogin bool

	// ShowRegister controls whether the register link appears in the nav.
	// True when multi-user is enabled and registration mode is open or public.
	ShowRegister bool
}

// LoadContentWithOverride loads content from embedded FS or overrides from file
// If overridePath is specified and file exists, uses that; otherwise uses embedded
func LoadContentWithOverride(embeddedPath, overridePath string) (string, error) {
	var content []byte
	var err error

	// Try override file first if specified
	if overridePath != "" {
		content, err = os.ReadFile(overridePath)
		if err == nil {
			return string(content), nil
		}
		// File doesn't exist or error, fall back to embedded
	}

	// Use embedded content
	content, err = embFS.ReadFile(embeddedPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func Load(db storage.DB, cfg config.Config) (*Data, error) {
	var data Data
	var err error

	// Setup base info
	data.DB = db
	data.Log = cfg.Log

	data.RateLimitNew = cfg.RateLimitNew
	data.RateLimitGet = cfg.RateLimitGet

	data.Version = cfg.Version
	data.Mode = cfg.Mode
	data.BuildCommit = cfg.BuildCommit
	data.BuildDate = cfg.BuildDate
	data.DataDir = cfg.DataDir
	data.ServerTagline = cfg.ServerTagline
	data.ServerDescription = cfg.ServerDescription

	data.TitleMaxLen = cfg.TitleMaxLen
	data.BodyMaxLen = cfg.BodyMaxLen
	data.MaxLifeTime = cfg.MaxLifeTime
	data.UiDefaultLifeTime = cfg.UiDefaultLifetime
	data.UiDefaultTheme = cfg.UiDefaultTheme
	data.Public = cfg.Public
	// Show login link only when the multi-user feature is enabled (accounts exist to log into)
	data.ShowLogin = cfg.Users.Enabled
	// Show register link when multi-user is enabled and registration is open
	data.ShowRegister = cfg.Users.Enabled && (cfg.Users.Registration.Mode == "open" || cfg.Users.Registration.Mode == "public")
	data.CasPasswdFile = cfg.CasPasswdFile

	// Initialize brute force protection for login
	// Per AI.md PART 11: 5 failed attempts = 15-minute lockout
	data.BruteForce = caspasswd.NewBruteForceProtection(5, 15*time.Minute)

	data.ServerAbout = cfg.ServerAbout
	data.ServerRules = cfg.ServerRules
	data.ServerTermsOfUse = cfg.ServerTermsOfUse

	serverTermsExist := false
	if cfg.ServerTermsOfUse != "" {
		serverTermsExist = true
	}
	data.ServerTermsExist = serverTermsExist

	data.AdminName = cfg.AdminName
	data.AdminMail = cfg.AdminMail

	data.FQDN = cfg.FQDN
	data.ServerTitle = cfg.ServerTitle
	data.SecurityContactEmail = cfg.SecurityContactEmail
	data.SecurityContactName = cfg.SecurityContactName
	data.SecurityTxt = cfg.SecurityTxt
	data.SiteRobotsAllow = cfg.SiteRobotsAllow
	data.SiteRobotsDeny = cfg.SiteRobotsDeny
	data.SiteRobotsAgentsDeny = cfg.SiteRobotsAgentsDeny
	data.Logo = cfg.Logo
	data.Favicon = cfg.Favicon

	// Get Chroma lexers
	data.Lexers = chromaLexers.Names(false)

	// Load locales
	data.Locales, data.LocalesList, err = loadLocales(embFS, "data/locale")
	if err != nil {
		return nil, err
	}

	// Load themes
	data.Themes, data.ThemesList, err = loadThemes(cfg.UiThemesDir, data.LocalesList, data.UiDefaultTheme)
	if err != nil {
		return nil, err
	}

	// style.css file
	data.StyleCSS, err = textTemplate.ParseFS(embFS, "data/style.css")
	if err != nil {
		return nil, err
	}

	// main.tmpl
	data.Main, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/main.tmpl")
	if err != nil {
		return nil, err
	}

	// main.js
	mainJS, err := embFS.ReadFile("data/main.js")
	if err != nil {
		return nil, err
	}
	data.MainJS = &mainJS

	// toast.js per AI.md PART 16
	toastJS, err := embFS.ReadFile("data/toast.js")
	if err != nil {
		return nil, err
	}
	data.ToastJS = &toastJS

	// nav.js
	navJS, err := embFS.ReadFile("data/nav.js")
	if err != nil {
		return nil, err
	}
	data.NavJS = &navJS

	// settings.js
	settingsJS, err := embFS.ReadFile("data/settings.js")
	if err != nil {
		return nil, err
	}
	data.SettingsJS = &settingsJS

	// history.js
	data.HistoryJS, err = textTemplate.ParseFS(embFS, "data/history.js")
	if err != nil {
		return nil, err
	}

	// code.js
	data.CodeJS, err = textTemplate.ParseFS(embFS, "data/code.js")
	if err != nil {
		return nil, err
	}

	// paste.tmpl
	data.PastePage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/paste.tmpl")
	if err != nil {
		return nil, err
	}

	// paste.js
	data.PasteJS, err = textTemplate.ParseFS(embFS, "data/paste.js")
	if err != nil {
		return nil, err
	}

	// paste_continue.tmpl
	data.PasteContinue, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/paste_continue.tmpl")
	if err != nil {
		return nil, err
	}

	// settings.tmpl
	data.Settings, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/settings.tmpl")
	if err != nil {
		return nil, err
	}

	// list.tmpl
	data.ListPage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/list.tmpl")
	if err != nil {
		return nil, err
	}

	// about.tmpl
	data.About, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/about.tmpl")
	if err != nil {
		return nil, err
	}

	// terms.tmpl
	data.TermsOfUse, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/terms.tmpl")
	if err != nil {
		return nil, err
	}

	// authors.tmpl
	data.Authors, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/authors.tmpl")
	if err != nil {
		return nil, err
	}

	// license.tmpl
	data.License, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/license.tmpl")
	if err != nil {
		return nil, err
	}

	// source_code.tmpl
	data.SourceCodePage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/source_code.tmpl")
	if err != nil {
		return nil, err
	}

	// security_policy.tmpl
	data.SecurityPolicy, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/security_policy.tmpl")
	if err != nil {
		return nil, err
	}

	// docs.tmpl
	data.Docs, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/docs.tmpl")
	if err != nil {
		return nil, err
	}

	// docs_apiv1.tmpl
	data.DocsApiV1, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/docs_apiv1.tmpl")
	if err != nil {
		return nil, err
	}

	// docs_libraries.tmpl
	data.DocsLibraries, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/docs_libraries.tmpl")
	if err != nil {
		return nil, err
	}

	// docs_customize.tmpl
	data.DocsCustomize, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/docs_customize.tmpl")
	if err != nil {
		return nil, err
	}

	// docs_cli.tmpl
	data.DocsCliExamples, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/docs_cli.tmpl")
	if err != nil {
		return nil, err
	}

	// error.tmpl
	data.ErrorPage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/error.tmpl")
	if err != nil {
		return nil, err
	}

	// emb.tmpl
	data.EmbeddedPage, err = template.ParseFS(embFS, "data/emb.tmpl")
	if err != nil {
		return nil, err
	}

	// emb_help.tmpl
	data.EmbeddedHelpPage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/emb_help.tmpl")
	if err != nil {
		return nil, err
	}

	// login.tmpl
	data.Login, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/login.tmpl")
	if err != nil {
		return nil, err
	}

	// healthz.tmpl
	data.HealthzPage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/healthz.tmpl")
	if err != nil {
		return nil, err
	}

	// stub.tmpl - shared template for placeholder/stub pages (auth, user, org)
	data.StubPage, err = template.ParseFS(embFS, "data/base.tmpl", "data/_header.tmpl", "data/_nav.tmpl", "data/_footer.tmpl", "data/stub.tmpl")
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (data *Data) Handler(rw http.ResponseWriter, req *http.Request) {
	// Process request
	var err error

	rw.Header().Set("Server", config.Software+"/"+data.Version)

	// Check authentication for protected routes (when server.public=false)
	if data.IsAuthRequired() && !IsPublicPath(req.URL.Path) {
		if !data.requireAuth(rw, req) {
			data.Log.HttpRequest(req, 302)
			return
		}
	}

	switch req.URL.Path {
	// Health checks per AI.md PART 13
	// /healthz and /server/healthz — content-negotiated (HTML/JSON/text)
	// /api/v1/server/healthz — JSON API (handled by apiv1 package)
	case "/healthz":
		err = data.handleHealthz(rw, req)
	case "/server/healthz":
		err = data.handleHealthz(rw, req)
	// Search engines
	case "/robots.txt":
		err = data.handleRobotsTxt(rw, req)
	case "/sitemap.xml":
		err = data.handleSitemap(rw, req)
	case "/favicon.ico":
		err = data.handleFavicon(rw, req)
	// Security
	case "/.well-known/security.txt":
		err = data.handleSecurityTxt(rw, req)
	case "/.well-known/change-password":
		// Per AI.md PART 34 - redirect to password change page
		if data.isAuthenticated(req) {
			http.Redirect(rw, req, "/users/security", http.StatusFound)
		} else {
			http.Redirect(rw, req, "/auth/password/forgot", http.StatusFound)
		}
		return
	// Resources
	case "/style.css":
		err = data.handleStyleCSS(rw, req)
	case "/main.js":
		err = data.handleMainJS(rw, req)
	case "/toast.js":
		err = data.handleToastJS(rw, req)
	case "/nav.js":
		err = data.handleNavJS(rw, req)
	case "/settings.js":
		err = data.handleSettingsJS(rw, req)
	case "/history.js":
		err = data.handleHistoryJS(rw, req)
	case "/code.js":
		err = data.handleCodeJS(rw, req)
	case "/paste.js":
		err = data.handlePasteJS(rw, req)
	// PWA Support
	case "/manifest.json":
		err = data.handleManifest(rw, req)
	case "/sw.js":
		err = data.handleServiceWorker(rw, req)
	// Server routes - Per AI.md PART 16
	case "/server/about":
		err = data.handleAbout(rw, req)
	case "/server/about/authors":
		err = data.handleAuthors(rw, req)
	case "/server/about/license":
		err = data.handleLicense(rw, req)
	case "/server/about/source_code":
		err = data.handleSourceCodePage(rw, req)
	case "/server/about/security":
		err = data.handleSecurityPolicy(rw, req)
	case "/server/help":
		// Help redirects to docs
		err = data.handleDocs(rw, req)
	// Legacy /about routes - 301 redirect to /server/about
	case "/about":
		http.Redirect(rw, req, "/server/about", http.StatusMovedPermanently)
	case "/about/authors":
		http.Redirect(rw, req, "/server/about/authors", http.StatusMovedPermanently)
	case "/about/license":
		http.Redirect(rw, req, "/server/about/license", http.StatusMovedPermanently)
	case "/about/source_code":
		http.Redirect(rw, req, "/server/about/source_code", http.StatusMovedPermanently)
	case "/about/security":
		http.Redirect(rw, req, "/server/about/security", http.StatusMovedPermanently)
	case "/docs":
		err = data.handleDocs(rw, req)
	case "/docs/apiv1":
		err = data.handleDocsAPIv1(rw, req)
	case "/docs/libraries":
		err = data.handleDocsLibraries(rw, req)
	// Redirect old URL
	case "/docs/api_libs":
		http.Redirect(rw, req, "/docs/libraries", http.StatusMovedPermanently)
	case "/docs/customize":
		err = data.handleDocsCustomize(rw, req)
	case "/docs/cli":
		err = data.handleDocsCliExamples(rw, req)
	// Canonical auth routes per AI.md PART 34
	case "/server/auth/login":
		if req.Method == "POST" {
			err = data.handleLoginSubmit(rw, req)
		} else {
			err = data.handleLoginPage(rw, req)
		}
	case "/server/auth/logout":
		err = data.handleLogout(rw, req)
	case "/server/auth/register":
		err = data.handleRegisterPage(rw, req)
	// Legacy auth redirects
	case "/login":
		http.Redirect(rw, req, "/server/auth/login", http.StatusMovedPermanently)
	case "/logout":
		http.Redirect(rw, req, "/server/auth/logout", http.StatusMovedPermanently)
	// User routes (PART 34)
	case "/users":
		err = data.handleUserDashboard(rw, req)
	case "/users/notifications":
		err = data.handleUserNotifications(rw, req)
	case "/users/settings":
		err = data.handleUserSettings(rw, req)
	case "/users/settings/privacy":
		err = data.handleUserSettingsPrivacy(rw, req)
	case "/users/settings/notifications":
		err = data.handleUserSettingsNotifications(rw, req)
	case "/users/settings/appearance":
		err = data.handleUserSettingsAppearance(rw, req)
	case "/users/security":
		err = data.handleUserSecurity(rw, req)
	case "/users/tokens":
		err = data.handleUserTokens(rw, req)
	case "/users/domains":
		err = data.handleUserDomains(rw, req)
	// Pages
	case "/":
		err = data.handleNewPaste(rw, req)
	case "/list":
		err = data.handleList(rw, req)
	case "/settings":
		err = data.handleSettings(rw, req)
	case "/terms":
		err = data.handleTermsOfUse(rw, req)
	// Else
	default:
		if strings.HasPrefix(req.URL.Path, "/dl/") {
			err = data.handleDownload(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/emb/") {
			err = data.handleEmbedded(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/emb_help/") {
			err = data.handleEmbeddedHelp(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/u/") {
			err = data.handleURLRedirect(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/qr/") {
			err = data.handleQRCode(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/edit/") {
			err = data.handleEditPaste(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/server/auth/") {
			// Canonical auth routes per AI.md PART 34 — rewrite to /auth/ prefix for routeAuth
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/server")
			err = data.routeAuth(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/auth/") {
			// Legacy /auth/* prefix — still supported
			err = data.routeAuth(rw, req)

		} else if strings.HasPrefix(req.URL.Path, "/orgs") {
			// Organization routes (PART 35)
			err = data.routeOrgs(rw, req)

		} else {
			err = data.handleGetPaste(rw, req)
		}
	}

	// Log
	if err == nil {
		data.Log.HttpRequest(req, 200)

	} else {
		// Log the original error before writing HTTP response
		data.Log.HttpError(req, err)

		code, writeErr := data.writeError(rw, req, err)
		if writeErr != nil {
			data.Log.HttpError(req, writeErr)
		}
		data.Log.HttpRequest(req, code)
	}
}
