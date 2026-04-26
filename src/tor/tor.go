// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package tor provides Tor hidden service support per AI.md PART 32
// Uses external Tor binary via github.com/cretz/bine for CGO_ENABLED=0 compatibility
// Hidden service is auto-enabled when Tor binary is found
package tor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil/ed25519"
)

// Config holds Tor configuration per AI.md PART 32
type Config struct {
	// Binary path (empty = auto-detect)
	Binary string `yaml:"binary" json:"binary"`

	// Outbound network settings
	UseNetwork          bool `yaml:"use_network" json:"use_network"`
	AllowUserPreference bool `yaml:"allow_user_preference" json:"allow_user_preference"`

	// Performance settings
	MaxCircuits      int           `yaml:"max_circuits" json:"max_circuits"`
	CircuitTimeout   time.Duration `yaml:"circuit_timeout" json:"circuit_timeout"`
	BootstrapTimeout time.Duration `yaml:"bootstrap_timeout" json:"bootstrap_timeout"`

	// Security settings
	SafeLogging               bool `yaml:"safe_logging" json:"safe_logging"`
	MaxStreamsPerCircuit      int  `yaml:"max_streams_per_circuit" json:"max_streams_per_circuit"`
	CloseCircuitOnStreamLimit bool `yaml:"close_circuit_on_stream_limit" json:"close_circuit_on_stream_limit"`

	// Bandwidth settings
	BandwidthRate       string `yaml:"bandwidth_rate" json:"bandwidth_rate"`
	BandwidthBurst      string `yaml:"bandwidth_burst" json:"bandwidth_burst"`
	MaxMonthlyBandwidth string `yaml:"max_monthly_bandwidth" json:"max_monthly_bandwidth"`

	// Hidden service settings
	NumIntroPoints int `yaml:"num_intro_points" json:"num_intro_points"`
	VirtualPort    int `yaml:"virtual_port" json:"virtual_port"`
}

// DefaultConfig returns the default Tor configuration per AI.md PART 32
func DefaultConfig() *Config {
	return &Config{
		Binary:                    "",
		UseNetwork:                false,
		AllowUserPreference:       true,
		MaxCircuits:               32,
		CircuitTimeout:            60 * time.Second,
		BootstrapTimeout:          3 * time.Minute,
		SafeLogging:               true,
		MaxStreamsPerCircuit:      100,
		CloseCircuitOnStreamLimit: true,
		BandwidthRate:             "1 MB",
		BandwidthBurst:            "2 MB",
		MaxMonthlyBandwidth:       "100 GB",
		NumIntroPoints:            3,
		VirtualPort:               80,
	}
}

// Status represents the current Tor status
type Status struct {
	Enabled    bool   `json:"enabled"`
	Running    bool   `json:"running"`
	StatusText string `json:"status"`
	Hostname   string `json:"hostname"`
	Error      string `json:"error,omitempty"`
}

// Service manages the Tor hidden service using bine per AI.md PART 32
type Service struct {
	config     *Config
	configDir  string
	dataDir    string
	logDir     string
	serverPort int

	// bine Tor instance
	torInstance *tor.Tor
	serviceID   string
	dialer      *tor.Dialer

	// State
	enabled    bool
	running    bool
	hostname   string
	binaryPath string
	lastError  string
	mu         sync.RWMutex
}

// NewService creates a new Tor service
func NewService(cfg *Config, configDir, dataDir, logDir string, serverPort int) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Service{
		config:     cfg,
		configDir:  configDir,
		dataDir:    dataDir,
		logDir:     logDir,
		serverPort: serverPort,
	}
}

// Start starts the Tor hidden service using bine per AI.md PART 32
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find Tor binary
	binaryPath, err := s.findTorBinary()
	if err != nil {
		s.enabled = false
		s.lastError = "Tor binary not found"
		log.Printf("Tor binary not found, hidden service disabled")
		return nil
	}
	s.binaryPath = binaryPath
	s.enabled = true
	log.Printf("Found Tor binary at %s", binaryPath)

	// Create directories with proper permissions
	if err := s.ensureDirs(); err != nil {
		s.lastError = err.Error()
		return fmt.Errorf("failed to create Tor directories: %w", err)
	}

	// Build bine StartConf
	torDataDir := filepath.Join(s.dataDir, "tor")
	conf := &tor.StartConf{
		DataDir:         torDataDir,
		NoAutoSocksPort: !s.config.UseNetwork && !s.config.AllowUserPreference,
		ExePath:         binaryPath,
		// ControlPort 0 = Tor picks an available port
		ControlPort: 0,
	}

	// Add extra args for security settings per AI.md PART 32
	conf.ExtraArgs = s.buildExtraArgs()

	// Start dedicated Tor process
	log.Printf("Starting Tor hidden service...")
	t, err := tor.Start(ctx, conf)
	if err != nil {
		s.lastError = fmt.Sprintf("failed to start Tor: %v", err)
		log.Printf("Tor: failed to start: %v", err)
		return nil
	}
	s.torInstance = t

	// Wait for Tor to bootstrap (connect to network)
	bootstrapCtx, cancel := context.WithTimeout(ctx, s.config.BootstrapTimeout)
	defer cancel()

	if err := t.EnableNetwork(bootstrapCtx, true); err != nil {
		t.Close()
		s.torInstance = nil
		s.lastError = fmt.Sprintf("bootstrap timeout: %v", err)
		log.Printf("Tor: bootstrap failed: %v", err)
		return nil
	}

	// Load or generate ED25519 key for persistent .onion address
	keyPath := filepath.Join(s.dataDir, "tor", "site", "hs_ed25519_secret_key")
	var onionKey control.Key
	if keyBlob, err := os.ReadFile(keyPath); err == nil && len(keyBlob) > 0 {
		// Load existing key for persistent address
		loadedKey, err := control.ED25519KeyFromBlob(string(keyBlob))
		if err != nil {
			log.Printf("Warning: failed to parse existing key: %v", err)
			// Fall through to generate new key
		} else {
			onionKey = loadedKey
		}
	}

	// If no key loaded, generate new ED25519-V3 key (v3 onion address)
	if onionKey == nil {
		onionKey = control.GenKey(control.KeyAlgoED25519V3)
	}

	// Create hidden service via ADD_ONION control command per AI.md PART 32
	addOnionReq := &control.AddOnionRequest{
		Key: onionKey,
		Ports: []*control.KeyVal{
			control.NewKeyVal(fmt.Sprintf("%d", s.config.VirtualPort), fmt.Sprintf("127.0.0.1:%d", s.serverPort)),
		},
	}

	// Call ADD_ONION via control connection
	resp, err := t.Control.AddOnion(addOnionReq)
	if err != nil {
		t.Close()
		s.torInstance = nil
		s.lastError = fmt.Sprintf("failed to create onion service: %v", err)
		log.Printf("Tor: failed to create onion service: %v", err)
		return nil
	}

	// Save key for persistent address (if newly generated or returned)
	if resp.Key != nil {
		if err := s.saveOnionKey(keyPath, resp.Key.Blob()); err != nil {
			log.Printf("Warning: failed to save onion key: %v", err)
		}
	}

	s.serviceID = resp.ServiceID
	s.hostname = resp.ServiceID + ".onion"

	// Initialize outbound dialer if enabled (server-wide or user preference allowed)
	if s.config.UseNetwork || s.config.AllowUserPreference {
		dialer, err := t.Dialer(ctx, nil)
		if err != nil {
			log.Printf("Warning: failed to create Tor dialer: %v", err)
		} else {
			s.dialer = dialer
			log.Printf("Tor outbound connections enabled")
		}
	}

	s.running = true
	s.lastError = ""
	log.Printf("Tor: %s", s.hostname)
	return nil
}

// buildExtraArgs builds extra arguments for Tor process per AI.md PART 32
func (s *Service) buildExtraArgs() []string {
	args := []string{}

	// SafeLogging
	if s.config.SafeLogging {
		args = append(args, "--SafeLogging", "1")
	}

	// Bandwidth settings
	if s.config.BandwidthRate != "" {
		args = append(args, "--BandwidthRate", s.config.BandwidthRate)
	}
	if s.config.BandwidthBurst != "" {
		args = append(args, "--BandwidthBurst", s.config.BandwidthBurst)
	}

	// Monthly bandwidth limit
	if s.config.MaxMonthlyBandwidth != "" && s.config.MaxMonthlyBandwidth != "unlimited" {
		args = append(args, "--AccountingStart", "month 1 00:00")
		args = append(args, "--AccountingMax", s.config.MaxMonthlyBandwidth)
	}

	// Disable relay features - we are not a relay or exit
	args = append(args,
		"--ExitRelay", "0",
		"--ExitPolicy", "reject *:*",
		"--ORPort", "0",
		"--DirPort", "0",
	)

	// Log file
	logFile := filepath.Join(s.logDir, "tor.log")
	args = append(args, "--Log", fmt.Sprintf("notice file %s", logFile))

	return args
}

// Stop stops the Tor hidden service
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.torInstance != nil {
		if err := s.torInstance.Close(); err != nil {
			return fmt.Errorf("failed to stop Tor: %w", err)
		}
		s.torInstance = nil
	}

	s.dialer = nil
	s.running = false
	return nil
}

// IsEnabled returns true if Tor is enabled (binary found)
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// IsRunning returns true if Tor is running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetHostname returns the .onion hostname
func (s *Service) GetHostname() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hostname
}

// GetStatus returns the current Tor status
func (s *Service) GetStatus() *Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &Status{
		Enabled:  s.enabled,
		Running:  s.running,
		Hostname: s.hostname,
		Error:    s.lastError,
	}

	if !s.enabled {
		status.StatusText = "disabled"
	} else if s.running {
		status.StatusText = "healthy"
	} else if s.lastError != "" {
		status.StatusText = "error"
	} else {
		status.StatusText = "stopped"
	}

	return status
}

// GetHTTPClient returns an HTTP client that uses Tor for outbound connections per AI.md PART 32
func (s *Service) GetHTTPClient(useTor bool) *http.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !useTor || s.dialer == nil {
		// Direct connection
		return &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Route through Tor network
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DialContext: s.dialer.DialContext,
		},
	}
}

// ShouldUseTor determines if Tor should be used based on config and user preference per AI.md PART 32
func (s *Service) ShouldUseTor(userPref *bool) bool {
	if !s.config.AllowUserPreference {
		return s.config.UseNetwork
	}

	if userPref == nil {
		return s.config.UseNetwork
	}

	return *userPref
}

// OnionAddress returns the full .onion address
func (s *Service) OnionAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hostname
}

// OutboundEnabled returns true if Tor outbound connections are available
func (s *Service) OutboundEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dialer != nil
}

// findTorBinary searches for the Tor binary
func (s *Service) findTorBinary() (string, error) {
	// Check config path first
	if s.config.Binary != "" {
		if _, err := os.Stat(s.config.Binary); err == nil {
			return s.config.Binary, nil
		}
	}

	// Check PATH
	if path, err := exec.LookPath("tor"); err == nil {
		return path, nil
	}

	// Platform-specific common locations
	var locations []string
	switch runtime.GOOS {
	case "linux":
		locations = []string{"/usr/bin/tor", "/usr/local/bin/tor"}
	case "darwin":
		locations = []string{"/usr/local/bin/tor", "/opt/homebrew/bin/tor"}
	case "windows":
		locations = []string{
			`C:\Program Files\Tor\tor.exe`,
			`C:\Program Files (x86)\Tor\tor.exe`,
		}
	case "freebsd":
		locations = []string{"/usr/local/bin/tor"}
	default:
		locations = []string{"/usr/local/bin/tor"}
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", fmt.Errorf("tor binary not found")
}

// ensureDirs creates required Tor directories with proper permissions per AI.md PART 32
func (s *Service) ensureDirs() error {
	dirs := []string{
		filepath.Join(s.configDir, "tor"),
		filepath.Join(s.dataDir, "tor"),
		filepath.Join(s.dataDir, "tor", "site"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	return nil
}

// saveOnionKey saves the ED25519 private key blob for persistent .onion address
func (s *Service) saveOnionKey(path string, keyBlob string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	// Write key with restricted permissions
	return os.WriteFile(path, []byte(keyBlob), 0600)
}

// GetConfig returns the current Tor configuration (for display)
func (s *Service) GetConfig() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"enabled":               s.enabled,
		"running":               s.running,
		"hostname":              s.hostname,
		"binary":                s.binaryPath,
		"use_network":           s.config.UseNetwork,
		"allow_user_preference": s.config.AllowUserPreference,
		"virtual_port":          s.config.VirtualPort,
		"num_intro_points":      s.config.NumIntroPoints,
	}
}

// Manager handles all Tor lifecycle operations per AI.md PART 32
type Manager struct {
	mu         sync.Mutex
	service    *Service
	config     *Config
	configDir  string
	dataDir    string
	logDir     string
	serverPort int
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewManager creates a new Tor manager with the given configuration
func NewManager(ctx context.Context, cfg *Config, configDir, dataDir, logDir string, serverPort int) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	managerCtx, cancel := context.WithCancel(ctx)

	return &Manager{
		config:     cfg,
		configDir:  configDir,
		dataDir:    dataDir,
		logDir:     logDir,
		serverPort: serverPort,
		ctx:        managerCtx,
		cancel:     cancel,
	}
}

// Start initializes Tor if binary is found
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.service = NewService(m.config, m.configDir, m.dataDir, m.logDir, m.serverPort)
	return m.service.Start(m.ctx)
}

// Stop stops the Tor service
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	if m.service != nil {
		return m.service.Stop()
	}
	return nil
}

// Restart stops and starts Tor (used for config changes, recovery)
func (m *Manager) Restart() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing
	if m.service != nil {
		m.service.Stop()
		m.service = nil
	}

	// Create new context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Start fresh
	m.service = NewService(m.config, m.configDir, m.dataDir, m.logDir, m.serverPort)
	return m.service.Start(m.ctx)
}

// UpdateConfig updates the configuration and restarts Tor
func (m *Manager) UpdateConfig(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config

	// Stop existing Tor
	if m.service != nil {
		m.service.Stop()
		m.service = nil
	}

	// Create new context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Start Tor with new config
	m.service = NewService(m.config, m.configDir, m.dataDir, m.logDir, m.serverPort)
	return m.service.Start(m.ctx)
}

// RegenerateAddress creates a new random .onion address
func (m *Manager) RegenerateAddress() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop Tor
	if m.service != nil {
		m.service.Stop()
		m.service = nil
	}

	// Delete existing keys to generate new address
	keyPath := filepath.Join(m.dataDir, "tor", "site", "hs_ed25519_secret_key")
	os.Remove(keyPath)

	// Create new context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Start Tor (will generate new keys and address)
	m.service = NewService(m.config, m.configDir, m.dataDir, m.logDir, m.serverPort)
	if err := m.service.Start(m.ctx); err != nil {
		return "", err
	}

	return m.service.GetHostname(), nil
}

// GetService returns the underlying Tor service
func (m *Manager) GetService() *Service {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.service
}

// GetStatus returns the current Tor status
func (m *Manager) GetStatus() *Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.service == nil {
		return &Status{
			Enabled:    false,
			Running:    false,
			StatusText: "disabled",
		}
	}

	return m.service.GetStatus()
}

// GetHTTPClient returns an HTTP client, optionally routed through Tor
func (m *Manager) GetHTTPClient(useTor bool) *http.Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.service == nil {
		return &http.Client{Timeout: 30 * time.Second}
	}

	return m.service.GetHTTPClient(useTor)
}

// Unused imports guard - remove if not needed
var _ = ed25519.GenerateKey
