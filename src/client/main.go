
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/casjay-forks/caspaste/src/completion"
	"github.com/casjay-forks/caspaste/src/display"
	"github.com/casjay-forks/caspaste/src/tui"
)

// Build info - set via -ldflags at build time
var (
	Version      = "unknown"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = ""
)

// Config represents the CLI configuration file per AI.md PART 33
type Config struct {
	Server    string `yaml:"server"`
	Token     string `yaml:"token,omitempty"`
	TokenFile string `yaml:"token_file,omitempty"`
	Username  string `yaml:"username,omitempty"`
	Password  string `yaml:"password,omitempty"`
}

// Runtime flags for token override per AI.md PART 33
var (
	flagToken     string
	flagTokenFile string
)

// APIResponse is the unified response wrapper per AI.md PART 16
type APIResponse struct {
	OK      bool            `json:"ok"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
}

// API response types (data payloads)
type NewPasteResponse struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
}

type GetPasteResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Syntax     string `json:"syntax"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
	OneUse     bool   `json:"oneUse"`
}

type ListPasteItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Syntax     string `json:"syntax"`
	CreateTime int64  `json:"createTime"`
	DeleteTime int64  `json:"deleteTime"`
}

type ListResponse struct {
	Pastes []ListPasteItem `json:"pastes"`
	Total  int             `json:"total"`
}

type ServerInfoResponse struct {
	Version           string   `json:"version"`
	TitleMaxLen       int      `json:"titleMaxlength"`
	BodyMaxLen        int      `json:"bodyMaxlength"`
	MaxLifeTime       int64    `json:"maxLifeTime"`
	ServerAbout       string   `json:"serverAbout"`
	ServerRules       string   `json:"serverRules"`
	ServerTermsOfUse  string   `json:"serverTermsOfUse"`
	AdminName         string   `json:"adminName"`
	AdminMail         string   `json:"adminMail"`
	Syntaxes          []string `json:"syntaxes"`
	UiDefaultLifeTime string   `json:"uiDefaultLifeTime"`
	AuthRequired      bool     `json:"authRequired"`
}

// parseAPIResponse parses the unified API response format
// Returns the data field if successful, or an error message if not
func parseAPIResponse(body []byte) (json.RawMessage, error) {
	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// Not a wrapped response, return raw body as data (for backwards compatibility)
		return body, nil
	}

	if !resp.OK {
		if resp.Message != "" {
			return nil, fmt.Errorf("%s: %s", resp.Error, resp.Message)
		}
		return nil, fmt.Errorf("%s", resp.Error)
	}

	// If no data field, return the whole body (for backwards compatibility)
	if resp.Data == nil || len(resp.Data) == 0 {
		return body, nil
	}

	return resp.Data, nil
}

func main() {
	// Handle --shell completions/init commands first (per AI.md PART 8/33)
	if len(os.Args) >= 2 && os.Args[1] == "--shell" {
		completion.Handle(os.Args[1:])
		return
	}

	// Parse global flags before command processing per AI.md PART 33
	// Global flags: --token, --token-file, --color
	args := parseGlobalFlags(os.Args[1:])

	// Detect display mode per AI.md PART 33
	mode := display.DetectForCLI()

	// No args - launch TUI if in TUI mode, otherwise show usage
	if len(args) < 1 {
		if mode == display.ModeTUI {
			launchTUI()
			return
		}
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	// Update os.Args for subcommand parsing (remove global flags)
	os.Args = append([]string{os.Args[0]}, args...)

	switch command {
	case "help", "--help", "-h":
		printUsage()
	case "version", "--version", "-v":
		fmt.Printf("%s v%s\n", filepath.Base(os.Args[0]), Version)
	case "config":
		handleConfig()
	case "new", "create", "paste":
		handleNew()
	case "get", "show", "view":
		handleGet()
	case "list", "ls":
		handleList()
	case "info", "server-info":
		handleServerInfo()
	case "health", "healthz":
		handleHealth()
	case "login":
		// If TUI mode available, use TUI setup wizard
		if mode == display.ModeTUI {
			launchSetupWizard()
			return
		}
		handleLogin()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

// launchTUI launches the main TUI application
func launchTUI() {
	cfg := loadConfig()

	// If no server configured, run setup wizard first
	if cfg.Server == "" {
		result, err := tui.RunSetupWizard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Setup cancelled: %v\n", err)
			os.Exit(1)
		}

		// Save the configured values
		cfg.Server = result.ServerURL
		if result.APIToken != "" {
			cfg.Password = result.APIToken
		}
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
		}
	}

	// Launch main TUI app
	if err := tui.RunApp(cfg.Server, cfg.Password); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// launchSetupWizard launches the TUI setup wizard
func launchSetupWizard() {
	result, err := tui.RunSetupWizard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup cancelled: %v\n", err)
		os.Exit(1)
	}

	// Save the configured values
	cfg := loadConfig()
	cfg.Server = result.ServerURL
	if result.APIToken != "" {
		cfg.Password = result.APIToken
	}

	if err := saveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	} else {
		fmt.Printf("Configuration saved to %s\n", getConfigPath())
	}
}

func printUsage() {
	binName := filepath.Base(os.Args[0])
	fmt.Printf(`CasPaste CLI v%s
A command-line client for CasPaste pastebin servers.

Usage: %s [global options] <command> [options]

Global Options:
  --token TOKEN       API token for authentication
  --token-file FILE   Read token from file
  --color MODE        Color output: always, never, auto (default: auto)

Commands:
  config              Show or edit configuration
  login               Configure server and credentials interactively
  new, create, paste  Create a new paste
  get, show, view     Get a paste by ID
  list, ls            List pastes
  info, server-info   Get server information
  health, healthz     Check server health
  help                Show this help message
  version             Show version

Shell Completions:
  --shell completions [SHELL]   Print shell completion script
  --shell init [SHELL]          Print shell init command for eval
  --shell --help                Show shell integration help

  Supported shells: bash, zsh, fish, sh, dash, ksh, powershell, pwsh
  If SHELL is omitted, it is auto-detected from $SHELL.

  Example: eval "$(%s --shell init)"

Examples:
  # Configure server and credentials
  %s login

  # Create paste from stdin
  echo "Hello World" | %s new

  # Create paste from file
  %s new -f script.py -s python

  # Get a paste
  %s get abc123

  # List recent pastes
  %s list -n 10

  # Use API token for authentication
  %s --token usr_abc123 new -f file.txt

Configuration:
  Config file: ~/.config/casjay-forks/caspaste/cli.yml

  Token priority (highest to lowest):
    1. --token flag
    2. --token-file flag
    3. CASPASTE_TOKEN environment variable
    4. Config file token field

  Or use environment variables:
    CASPASTE_SERVER=https://paste.example.com
    CASPASTE_TOKEN=usr_abc123
    CASPASTE_USERNAME=admin
    CASPASTE_PASSWORD=secret

`, Version, binName, binName, binName, binName, binName, binName, binName, binName)
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	// Check XDG_CONFIG_HOME first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "casjay-forks", "caspaste", "cli.yml")
	}
	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "casjay-forks", "caspaste", "cli.yml")
}

// loadConfig loads configuration from file and environment
func loadConfig() Config {
	var cfg Config

	// Load from file first
	configPath := getConfigPath()
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			yaml.Unmarshal(data, &cfg)
		}
	}

	// Environment variables override file config
	if server := os.Getenv("CASPASTE_SERVER"); server != "" {
		cfg.Server = server
	}
	if username := os.Getenv("CASPASTE_USERNAME"); username != "" {
		cfg.Username = username
	}
	if password := os.Getenv("CASPASTE_PASSWORD"); password != "" {
		cfg.Password = password
	}

	return cfg
}

// saveConfig saves configuration to file
func saveConfig(cfg Config) error {
	configPath := getConfigPath()
	if configPath == "" {
		return fmt.Errorf("could not determine config path")
	}

	// Create directory if needed
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (contains password)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// parseGlobalFlags extracts global flags before command processing per AI.md PART 33
// Global flags: --token, --token-file, --color
// Returns remaining args after extracting global flags
func parseGlobalFlags(args []string) []string {
	var remaining []string
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--token":
			if i+1 < len(args) {
				flagToken = args[i+1]
				i += 2
				continue
			}
			i++
		case "--token-file":
			if i+1 < len(args) {
				flagTokenFile = args[i+1]
				i += 2
				continue
			}
			i++
		case "--color":
			if i+1 < len(args) {
				display.SetColorMode(args[i+1])
				i += 2
				continue
			}
			i++
		default:
			// Check for --flag=VALUE format
			if strings.HasPrefix(args[i], "--token=") {
				flagToken = strings.TrimPrefix(args[i], "--token=")
				i++
				continue
			}
			if strings.HasPrefix(args[i], "--token-file=") {
				flagTokenFile = strings.TrimPrefix(args[i], "--token-file=")
				i++
				continue
			}
			if strings.HasPrefix(args[i], "--color=") {
				display.SetColorMode(strings.TrimPrefix(args[i], "--color="))
				i++
				continue
			}
			remaining = append(remaining, args[i])
			i++
		}
	}
	return remaining
}

// getToken returns the API token with proper priority per AI.md PART 33:
// 1. --token flag (explicit)
// 2. --token-file flag (file path)
// 3. Environment variable: CASPASTE_TOKEN
// 4. Config file: cli.yml -> token
func getToken(cfg Config) string {
	// 1. CLI flag --token takes highest priority
	if flagToken != "" {
		return flagToken
	}

	// 2. CLI flag --token-file
	if flagTokenFile != "" {
		data, err := os.ReadFile(flagTokenFile)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// 3. Environment variable CASPASTE_TOKEN
	if envToken := os.Getenv("CASPASTE_TOKEN"); envToken != "" {
		return envToken
	}

	// 4. Config file token field
	if cfg.Token != "" {
		return cfg.Token
	}

	// 5. Config file token_file field
	if cfg.TokenFile != "" {
		data, err := os.ReadFile(cfg.TokenFile)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}

// makeRequest makes an HTTP request with token or basic auth per AI.md PART 33
func makeRequest(method, endpoint string, body io.Reader, contentType string, cfg Config) (*http.Response, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("server not configured. Run 'caspaste-cli login' first")
	}

	url := strings.TrimSuffix(cfg.Server, "/") + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set User-Agent per AI.md requirement
	req.Header.Set("User-Agent", filepath.Base(os.Args[0])+"/"+Version)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Auth priority per AI.md PART 33:
	// 1. Token auth (if token available)
	// 2. Basic auth (if username/password configured)
	token := getToken(cfg)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if cfg.Username != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func handleConfig() {
	cfg := loadConfig()
	configPath := getConfigPath()

	fmt.Printf("Config file: %s\n\n", configPath)
	fmt.Printf("Server:   %s\n", cfg.Server)
	fmt.Printf("Username: %s\n", cfg.Username)
	if cfg.Password != "" {
		fmt.Printf("Password: ******* (set)\n")
	} else {
		fmt.Printf("Password: (not set)\n")
	}
}

func handleLogin() {
	cfg := loadConfig()
	reader := bufio.NewReader(os.Stdin)

	// Server URL
	fmt.Printf("Server URL [%s]: ", cfg.Server)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cfg.Server = input
	}

	// Username
	fmt.Printf("Username [%s]: ", cfg.Username)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cfg.Username = input
	}

	// Password
	fmt.Print("Password: ")
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		cfg.Password = input
	}

	// Test connection
	fmt.Print("\nTesting connection... ")
	resp, err := makeRequest("GET", "/api/v1/healthz", nil, "", cfg)
	if err != nil {
		fmt.Printf("FAILED\nError: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("FAILED\nServer returned: %s\n", resp.Status)
		os.Exit(1)
	}
	fmt.Println("OK")

	// Test auth if credentials provided
	if cfg.Username != "" && cfg.Password != "" {
		fmt.Print("Testing authentication... ")
		// We can't easily test auth without creating a paste
		// Just save and let the user know
		fmt.Println("(will be verified on first paste)")
	}

	// Save config
	if err := saveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
	} else {
		fmt.Printf("\nConfiguration saved to %s\n", getConfigPath())
	}
}

func handleNew() {
	cfg := loadConfig()

	// Parse flags
	var title, syntax, lifetime, filePath string
	var oneUse, private bool

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--title":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "-s", "--syntax":
			if i+1 < len(args) {
				syntax = args[i+1]
				i++
			}
		case "-l", "--lifetime":
			if i+1 < len(args) {
				lifetime = args[i+1]
				i++
			}
		case "-f", "--file":
			if i+1 < len(args) {
				filePath = args[i+1]
				i++
			}
		case "-1", "--one-use":
			oneUse = true
		case "-p", "--private":
			private = true
		case "-h", "--help":
			fmt.Println(`Create a new paste

Usage: caspaste-cli new [options]

Options:
  -f, --file FILE      Read content from file (default: stdin)
  -t, --title TITLE    Paste title
  -s, --syntax SYNTAX  Syntax highlighting (e.g., python, go, bash)
  -l, --lifetime TIME  Expiration time (e.g., 1h, 1d, 1w, never)
  -1, --one-use        Delete after first view
  -p, --private        Don't show in public listings

Examples:
  echo "Hello" | caspaste-cli new
  caspaste-cli new -f script.py -s python -t "My Script"
  cat log.txt | caspaste-cli new -l 1h -1`)
			return
		}
	}

	// Read content
	var content []byte
	var err error

	if filePath != "" {
		content, err = os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		// Auto-detect syntax from extension if not specified
		if syntax == "" {
			ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
			syntax = extToSyntax(ext)
		}
		// Use filename as title if not specified
		if title == "" {
			title = filepath.Base(filePath)
		}
	} else {
		// Read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Println("Reading from stdin... (press Ctrl+D when done)")
		}
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	}

	if len(content) == 0 {
		fmt.Fprintf(os.Stderr, "Error: empty content\n")
		os.Exit(1)
	}

	// Build form data
	form := url.Values{}
	form.Set("body", string(content))
	if title != "" {
		form.Set("title", title)
	}
	if syntax != "" {
		form.Set("syntax", syntax)
	}
	if lifetime != "" {
		form.Set("expireAfter", lifetime)
	}
	if oneUse {
		form.Set("oneUse", "true")
	}
	if private {
		form.Set("private", "true")
	}

	// Make request - POST to /api/v1/pastes per REST API spec
	resp, err := makeRequest("POST", "/api/v1/pastes", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		fmt.Fprintf(os.Stderr, "Error: Authentication required. Run 'caspaste-cli login' to configure credentials.\n")
		os.Exit(1)
	}

	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		fmt.Fprintf(os.Stderr, "Error: Too many failed attempts. Try again in %s seconds.\n", retryAfter)
		os.Exit(1)
	}

	if resp.StatusCode != 200 {
		// Parse unified error response per AI.md PART 16
		data, parseErr := parseAPIResponse(body)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n%s\n", resp.Status, string(data))
		}
		os.Exit(1)
	}

	// Parse unified success response per AI.md PART 16
	data, parseErr := parseAPIResponse(body)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		os.Exit(1)
	}

	var result NewPasteResponse
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Paste created!\n")
	fmt.Printf("ID:  %s\n", result.ID)
	fmt.Printf("URL: %s\n", result.URL)
	if result.DeleteTime > 0 {
		fmt.Printf("Expires: %s\n", time.Unix(result.DeleteTime, 0).Format(time.RFC3339))
	}
}

func handleGet() {
	cfg := loadConfig()

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: caspaste-cli get <paste-id>\n")
		os.Exit(1)
	}

	pasteID := os.Args[2]

	// Check for raw output flag
	raw := false
	for _, arg := range os.Args[3:] {
		if arg == "-r" || arg == "--raw" {
			raw = true
		}
	}

	// GET /api/v1/pastes?id= per REST API spec
	resp, err := makeRequest("GET", "/api/v1/pastes?id="+url.QueryEscape(pasteID), nil, "", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 404 {
		fmt.Fprintf(os.Stderr, "Error: Paste not found\n")
		os.Exit(1)
	}

	if resp.StatusCode != 200 {
		// Parse unified error response per AI.md PART 16
		_, parseErr := parseAPIResponse(body)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Status)
		}
		os.Exit(1)
	}

	// Parse unified success response per AI.md PART 16
	data, parseErr := parseAPIResponse(body)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		os.Exit(1)
	}

	var result GetPasteResponse
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if raw {
		fmt.Print(result.Body)
	} else {
		fmt.Printf("ID:      %s\n", result.ID)
		if result.Title != "" {
			fmt.Printf("Title:   %s\n", result.Title)
		}
		fmt.Printf("Syntax:  %s\n", result.Syntax)
		fmt.Printf("Created: %s\n", time.Unix(result.CreateTime, 0).Format(time.RFC3339))
		if result.DeleteTime > 0 {
			fmt.Printf("Expires: %s\n", time.Unix(result.DeleteTime, 0).Format(time.RFC3339))
		}
		if result.OneUse {
			fmt.Println("OneUse:  Yes (this paste is now deleted)")
		}
		fmt.Println("\n--- Content ---")
		fmt.Println(result.Body)
	}
}

func handleList() {
	cfg := loadConfig()

	// Parse flags
	limit := "20"
	offset := "0"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--limit":
			if i+1 < len(args) {
				limit = args[i+1]
				i++
			}
		case "-o", "--offset":
			if i+1 < len(args) {
				offset = args[i+1]
				i++
			}
		}
	}

	// GET /api/v1/pastes without id parameter returns list per REST API spec
	endpoint := fmt.Sprintf("/api/v1/pastes?limit=%s&offset=%s", limit, offset)
	resp, err := makeRequest("GET", endpoint, nil, "", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// Parse unified error response per AI.md PART 16
		_, parseErr := parseAPIResponse(body)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Status)
		}
		os.Exit(1)
	}

	// Parse unified success response per AI.md PART 16
	data, parseErr := parseAPIResponse(body)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		os.Exit(1)
	}

	var result ListResponse
	if err := json.Unmarshal(data, &result); err != nil {
		// Try parsing as array (older API format or direct array in data)
		var pastes []ListPasteItem
		if err2 := json.Unmarshal(data, &pastes); err2 == nil {
			result.Pastes = pastes
			result.Total = len(pastes)
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}
	}

	if len(result.Pastes) == 0 {
		fmt.Println("No pastes found")
		return
	}

	fmt.Printf("%-12s %-30s %-12s %s\n", "ID", "TITLE", "SYNTAX", "CREATED")
	fmt.Println(strings.Repeat("-", 70))

	for _, p := range result.Pastes {
		title := p.Title
		if title == "" {
			title = "(untitled)"
		}
		if len(title) > 28 {
			title = title[:25] + "..."
		}
		created := time.Unix(p.CreateTime, 0).Format("2006-01-02")
		fmt.Printf("%-12s %-30s %-12s %s\n", p.ID, title, p.Syntax, created)
	}
}

func handleServerInfo() {
	cfg := loadConfig()

	// GET /api/v1/server/info per REST API spec
	resp, err := makeRequest("GET", "/api/v1/server/info", nil, "", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// Parse unified error response per AI.md PART 16
		_, parseErr := parseAPIResponse(body)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Status)
		}
		os.Exit(1)
	}

	// Parse unified success response per AI.md PART 16
	data, parseErr := parseAPIResponse(body)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		os.Exit(1)
	}

	var result ServerInfoResponse
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Server: %s\n", cfg.Server)
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Title Max Length: %d\n", result.TitleMaxLen)
	fmt.Printf("Body Max Length: %d bytes (%.1f MB)\n", result.BodyMaxLen, float64(result.BodyMaxLen)/1024/1024)
	if result.MaxLifeTime > 0 {
		fmt.Printf("Max Lifetime: %d seconds\n", result.MaxLifeTime)
	} else {
		fmt.Printf("Max Lifetime: unlimited\n")
	}
	fmt.Printf("Admin: %s <%s>\n", result.AdminName, result.AdminMail)
	fmt.Printf("Auth Required: %v\n", result.AuthRequired)
	fmt.Printf("Supported Syntaxes: %d languages\n", len(result.Syntaxes))
}

func handleHealth() {
	cfg := loadConfig()

	resp, err := makeRequest("GET", "/api/v1/healthz", nil, "", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("Server %s is healthy\n", cfg.Server)
	} else {
		fmt.Printf("Server %s returned: %s\n", cfg.Server, resp.Status)
		os.Exit(1)
	}
}

// extToSyntax maps file extensions to syntax names
func extToSyntax(ext string) string {
	mapping := map[string]string{
		"py":    "python",
		"js":    "javascript",
		"ts":    "typescript",
		"go":    "go",
		"rs":    "rust",
		"rb":    "ruby",
		"php":   "php",
		"java":  "java",
		"c":     "c",
		"cpp":   "cpp",
		"h":     "c",
		"hpp":   "cpp",
		"cs":    "csharp",
		"sh":    "bash",
		"bash":  "bash",
		"zsh":   "bash",
		"ps1":   "powershell",
		"sql":   "sql",
		"json":  "json",
		"yaml":  "yaml",
		"yml":   "yaml",
		"xml":   "xml",
		"html":  "html",
		"css":   "css",
		"scss":  "scss",
		"md":    "markdown",
		"txt":   "plaintext",
		"log":   "plaintext",
		"conf":  "ini",
		"ini":   "ini",
		"toml":  "toml",
		"dockerfile": "docker",
		"makefile":   "makefile",
	}

	if syntax, ok := mapping[strings.ToLower(ext)]; ok {
		return syntax
	}
	return ""
}

// uploadFile handles file upload for binary files
func uploadFile(filePath string, cfg Config) (*NewPasteResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	writer.Close()

	// POST to /api/v1/pastes per REST API spec
	resp, err := makeRequest("POST", "/api/v1/pastes", &buf, writer.FormDataContentType(), cfg)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// Parse unified error response per AI.md PART 16
		_, parseErr := parseAPIResponse(body)
		if parseErr != nil {
			return nil, parseErr
		}
		return nil, fmt.Errorf("upload failed: %s", resp.Status)
	}

	// Parse unified success response per AI.md PART 16
	data, parseErr := parseAPIResponse(body)
	if parseErr != nil {
		return nil, parseErr
	}

	var result NewPasteResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
