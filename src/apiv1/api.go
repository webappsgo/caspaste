
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"net/http"
	"strings"
	"time"

	chromaLexers "github.com/alecthomas/chroma/v2/lexers"

	"github.com/casjay-forks/caspaste/src/caspasswd"
	"github.com/casjay-forks/caspaste/src/config"
	"github.com/casjay-forks/caspaste/src/httputil"
	"github.com/casjay-forks/caspaste/src/logger"
	"github.com/casjay-forks/caspaste/src/netshare"
	"github.com/casjay-forks/caspaste/src/storage"
)

type Data struct {
	Log logger.Logger
	DB  storage.DB

	RateLimitNew *netshare.RateLimitSystem
	RateLimitGet *netshare.RateLimitSystem

	Lexers []string

	Version     string
	BuildCommit string
	BuildDate   string
	Mode        string

	// DataDir is the application data directory path; used for disk health check
	DataDir string

	// SchedulerStatus returns the current scheduler health string ("ok" or "error: …")
	SchedulerStatus func() string

	ServerTagline     string
	ServerDescription string

	TitleMaxLen int
	BodyMaxLen  int
	MaxLifeTime int64

	ServerAbout      string
	ServerRules      string
	ServerTermsOfUse string

	ServerTitle string
	AdminName   string
	AdminMail   string

	// true = open/public, false = auth required
	Public        bool
	CasPasswdFile string
	BruteForce    *caspasswd.BruteForceProtection

	UiDefaultLifeTime string
}

func Load(db storage.DB, cfg config.Config) *Data {
	lexers := chromaLexers.Names(false)

	// Initialize brute force protection if authentication is required (server.public=false)
	var bruteForce *caspasswd.BruteForceProtection
	if !cfg.Public && cfg.CasPasswdFile != "" {
		// 5 failed attempts = 15 minute lockout
		bruteForce = caspasswd.NewBruteForceProtection(5, 15*time.Minute)
	}

	return &Data{
		DB:                db,
		Log:               cfg.Log,
		RateLimitNew:      cfg.RateLimitNew,
		RateLimitGet:      cfg.RateLimitGet,
		Lexers:            lexers,
		Version:           cfg.Version,
		BuildCommit:       cfg.BuildCommit,
		BuildDate:         cfg.BuildDate,
		Mode:              cfg.Mode,
		DataDir:           cfg.DataDir,
		ServerTagline:     cfg.ServerTagline,
		ServerDescription: cfg.ServerDescription,
		TitleMaxLen:       cfg.TitleMaxLen,
		BodyMaxLen:        cfg.BodyMaxLen,
		MaxLifeTime:       cfg.MaxLifeTime,
		ServerAbout:       cfg.ServerAbout,
		ServerRules:       cfg.ServerRules,
		ServerTermsOfUse:  cfg.ServerTermsOfUse,
		ServerTitle:       cfg.ServerTitle,
		AdminName:         cfg.AdminName,
		AdminMail:         cfg.AdminMail,
		Public:            cfg.Public,
		CasPasswdFile:     cfg.CasPasswdFile,
		BruteForce:        bruteForce,
		UiDefaultLifeTime: cfg.UiDefaultLifetime,
	}
}

func (data *Data) Hand(rw http.ResponseWriter, req *http.Request) {
	// Process request
	var err error

	rw.Header().Set("Server", config.Software+"/"+data.Version)

	// Build API paths dynamically per AI.md PART 14
	apiBase := config.APIBasePath()
	path := req.URL.Path

	// Strip .txt extension for routing per AI.md PART 14 content negotiation
	// The format is determined by httputil.GetAPIResponseFormat() in handlers
	routePath := httputil.StripTxtExtension(path)

	// Route API requests
	switch routePath {
	// Health check per AI.md PART 13
	case apiBase + "/server/healthz":
		err = data.handleHealthz(rw, req)
	// API v1 endpoints per AI.md PART 14 (noun-based REST routes)
	case apiBase + "/pastes":
		// Route by method: POST=create, GET=list or get single
		err = data.handlePastes(rw, req)
	case apiBase + "/server/info":
		err = data.handleServerInfo(rw, req)

	// External API Compatibility endpoints per AI.md "External API Compatibility"
	// pastebin.com compatibility
	case "/api/api_post.php":
		err = data.handleCompat(rw, req)
	// stikked/stiqued compatibility
	case "/api/create":
		err = data.handleCompat(rw, req)
	// lenpaste compatibility (also handled by apiBase + "/pastes" but adding for clarity)
	case "/api/v1/new":
		err = data.handleCompat(rw, req)
	// sprunge, ix, termbin, etc - root-level compat routes
	case "/sprunge", "/sprunge/":
		err = data.handleCompat(rw, req)
	case "/ix", "/ix/":
		err = data.handleCompat(rw, req)
	case "/termbin", "/nc":
		err = data.handleCompat(rw, req)
	case "/upload", "/p":
		err = data.handleCompat(rw, req)
	case "/compat", "/paste":
		err = data.handleCompat(rw, req)
	// hastebin compatibility
	case "/documents":
		err = data.handleCompat(rw, req)

	default:
		switch {
		// Support path-based paste ID: GET /api/v1/pastes/{id}
		case strings.HasPrefix(routePath, apiBase+"/pastes/"):
			err = data.handlePastes(rw, req)
		// hastebin: GET /documents/{key}
		case strings.HasPrefix(routePath, "/documents/"):
			err = data.handleCompat(rw, req)
		default:
			err = netshare.ErrNotFound
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
