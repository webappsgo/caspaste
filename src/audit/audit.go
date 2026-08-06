// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package audit provides security audit logging per AI.md PART 11.
// All audit logs are JSON format, one entry per line (JSON Lines).
package audit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Event types per AI.md PART 11
const (
	// Authentication events
	EventAdminLogin       = "admin.login"
	EventAdminLogout      = "admin.logout"
	EventAdminLoginFailed = "admin.login_failed"
	EventUserLogin        = "user.login"
	EventUserLogout       = "user.logout"
	EventUserLoginFailed  = "user.login_failed"

	// Security events
	EventRateLimitExceeded = "security.rate_limit_exceeded"
	EventCSRFFailure       = "security.csrf_failure"
	EventInvalidToken      = "security.invalid_token"
	EventBruteForceDetect  = "security.brute_force_detected"
	EventIPBlocked         = "security.ip_blocked"

	// Server events
	EventServerStarted    = "server.started"
	EventServerStopped    = "server.stopped"
	EventMaintenanceEnter = "server.maintenance_entered"
	EventMaintenanceExit  = "server.maintenance_exited"

	// Backup events
	EventBackupCreated  = "backup.created"
	EventBackupRestored = "backup.restored"
	EventBackupFailed   = "backup.failed"

	// Config events
	EventConfigUpdated = "config.updated"
)

// Entry represents a single audit log entry per AI.md PART 11
type Entry struct {
	// Unique event ID (audit_XXXXXXXXXXXX)
	ID string `json:"id"`
	// Timestamp in UTC with milliseconds
	Time string `json:"time"`
	// Event type (e.g., admin.login, security.csrf_failure)
	Event string `json:"event"`
	// Result: success or failure
	Result string `json:"result"`
	// Actor who performed the action
	Actor *Actor `json:"actor,omitempty"`
	// Target of the action (optional)
	Target *Target `json:"target,omitempty"`
	// Client information
	Client *Client `json:"client,omitempty"`
	// Additional event-specific details
	Details map[string]interface{} `json:"details,omitempty"`
}

// Actor represents who performed the action
type Actor struct {
	// Type: admin, user, system, anonymous
	Type string `json:"type"`
	// ID of the actor (admin username, user ID, or "system")
	ID string `json:"id,omitempty"`
}

// Target represents the target of the action
type Target struct {
	// Type: user, org, config, backup, etc.
	Type string `json:"type"`
	// ID of the target
	ID string `json:"id,omitempty"`
}

// Client represents client connection information
type Client struct {
	// IP address
	IP string `json:"ip"`
	// User agent (optional)
	UserAgent string `json:"user_agent,omitempty"`
	// Request ID for tracing
	RequestID string `json:"request_id,omitempty"`
}

// Config represents audit log configuration
type Config struct {
	// Enabled controls whether audit logging is active
	Enabled bool
	// Directory for audit log file
	Directory string
	// Filename for audit log (default: audit.log)
	Filename string
	// MaskEmails masks email addresses in logs
	MaskEmails bool
	// IncludeUserAgent includes user agent in client info
	IncludeUserAgent bool
}

// Logger provides audit logging functionality
type Logger struct {
	config Config
	file   *os.File
	mu     sync.Mutex
}

// Global audit logger instance (set via Init)
var globalLogger *Logger
var globalMu sync.RWMutex

// Init initializes the global audit logger
func Init(cfg Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	globalMu.Lock()
	globalLogger = logger
	globalMu.Unlock()
	return nil
}

// GetLogger returns the global audit logger (nil if not initialized)
func GetLogger() *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// CloseGlobal closes the global audit logger
func CloseGlobal() error {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalLogger != nil {
		err := globalLogger.Close()
		globalLogger = nil
		return err
	}
	return nil
}

// emailRegex for masking email addresses
var emailRegex = regexp.MustCompile(`([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)

// New creates a new audit logger
func New(cfg Config) (*Logger, error) {
	if !cfg.Enabled {
		return &Logger{config: cfg}, nil
	}

	// Set defaults
	if cfg.Filename == "" {
		cfg.Filename = "audit.log"
	}

	// Create directory if needed
	if cfg.Directory != "" {
		if err := os.MkdirAll(cfg.Directory, 0750); err != nil {
			return nil, fmt.Errorf("failed to create audit log directory: %w", err)
		}
	}

	// Open audit log file
	logPath := filepath.Join(cfg.Directory, cfg.Filename)
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &Logger{
		config: cfg,
		file:   file,
	}, nil
}

// Close closes the audit log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// generateID generates a unique audit event ID
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "audit_" + hex.EncodeToString(b)
}

// maskEmail masks an email address (j***n@e***.com)
func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	local := parts[0]
	domain := parts[1]

	// Mask local part
	maskedLocal := string(local[0])
	if len(local) > 1 {
		maskedLocal += "***" + string(local[len(local)-1])
	}

	// Mask domain
	domainParts := strings.Split(domain, ".")
	if len(domainParts) >= 2 {
		maskedDomain := string(domainParts[0][0]) + "***." + domainParts[len(domainParts)-1]
		return maskedLocal + "@" + maskedDomain
	}

	return maskedLocal + "@" + domain
}

// maskEmails masks all email addresses in a string
func (l *Logger) maskEmails(s string) string {
	if !l.config.MaskEmails {
		return s
	}
	return emailRegex.ReplaceAllStringFunc(s, maskEmail)
}

// Log writes an audit entry
func (l *Logger) Log(entry Entry) error {
	if !l.config.Enabled || l.file == nil {
		return nil
	}

	// Set defaults
	if entry.ID == "" {
		entry.ID = generateID()
	}
	if entry.Time == "" {
		entry.Time = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	}
	if entry.Result == "" {
		entry.Result = "success"
	}

	// Mask emails in details
	if l.config.MaskEmails && entry.Details != nil {
		for k, v := range entry.Details {
			if s, ok := v.(string); ok {
				entry.Details[k] = l.maskEmails(s)
			}
		}
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Write to file (with newline for JSON Lines format)
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err = l.file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	return nil
}

// LogSuccess logs a successful event
func (l *Logger) LogSuccess(event string, actor *Actor, client *Client, details map[string]interface{}) error {
	return l.Log(Entry{
		Event:   event,
		Result:  "success",
		Actor:   actor,
		Client:  client,
		Details: details,
	})
}

// LogFailure logs a failed event
func (l *Logger) LogFailure(event string, actor *Actor, client *Client, reason string, details map[string]interface{}) error {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["reason"] = reason

	return l.Log(Entry{
		Event:   event,
		Result:  "failure",
		Actor:   actor,
		Client:  client,
		Details: details,
	})
}

// Helper methods for common events

// LogAdminLogin logs an admin login event
func (l *Logger) LogAdminLogin(adminUsername string, ip string, userAgent string, requestID string, mfaUsed bool) error {
	details := map[string]interface{}{
		"mfa_used": mfaUsed,
	}

	client := &Client{IP: ip, RequestID: requestID}
	if l.config.IncludeUserAgent {
		client.UserAgent = userAgent
	}

	return l.LogSuccess(EventAdminLogin, &Actor{Type: "admin", ID: adminUsername}, client, details)
}

// LogAdminLoginFailed logs a failed admin login attempt
func (l *Logger) LogAdminLoginFailed(attemptedUsername string, ip string, userAgent string, requestID string, reason string) error {
	details := map[string]interface{}{
		"attempted_username": attemptedUsername,
	}

	client := &Client{IP: ip, RequestID: requestID}
	if l.config.IncludeUserAgent {
		client.UserAgent = userAgent
	}

	return l.LogFailure(EventAdminLoginFailed, &Actor{Type: "anonymous"}, client, reason, details)
}

// LogRateLimitExceeded logs a rate limit exceeded event
func (l *Logger) LogRateLimitExceeded(ip string, endpoint string, limit int, requestID string) error {
	return l.LogFailure(EventRateLimitExceeded, &Actor{Type: "anonymous"},
		&Client{IP: ip, RequestID: requestID},
		"rate limit exceeded",
		map[string]interface{}{
			"endpoint": endpoint,
			"limit":    limit,
		})
}

// LogCSRFFailure logs a CSRF validation failure
func (l *Logger) LogCSRFFailure(ip string, endpoint string, requestID string) error {
	return l.LogFailure(EventCSRFFailure, &Actor{Type: "anonymous"},
		&Client{IP: ip, RequestID: requestID},
		"CSRF token validation failed",
		map[string]interface{}{
			"endpoint": endpoint,
		})
}

// LogServerStarted logs server startup
func (l *Logger) LogServerStarted(version string, mode string) error {
	return l.LogSuccess(EventServerStarted, &Actor{Type: "system", ID: "server"}, nil,
		map[string]interface{}{
			"version": version,
			"mode":    mode,
		})
}

// LogServerStopped logs server shutdown
func (l *Logger) LogServerStopped(reason string, uptime time.Duration) error {
	return l.LogSuccess(EventServerStopped, &Actor{Type: "system", ID: "server"}, nil,
		map[string]interface{}{
			"reason":    reason,
			"uptime_ms": uptime.Milliseconds(),
		})
}

// LogBackupCreated logs backup creation
func (l *Logger) LogBackupCreated(filename string, size int64, createdBy string) error {
	return l.LogSuccess(EventBackupCreated, &Actor{Type: "admin", ID: createdBy}, nil,
		map[string]interface{}{
			"filename": filename,
			"size":     size,
		})
}

// LogBackupRestored logs backup restoration
func (l *Logger) LogBackupRestored(filename string, restoredBy string) error {
	return l.LogSuccess(EventBackupRestored, &Actor{Type: "admin", ID: restoredBy}, nil,
		map[string]interface{}{
			"filename": filename,
		})
}

// LogBackupFailed logs backup failure
func (l *Logger) LogBackupFailed(operation string, errorMsg string) error {
	return l.LogFailure(EventBackupFailed, &Actor{Type: "system", ID: "server"}, nil,
		errorMsg,
		map[string]interface{}{
			"operation": operation,
		})
}

// LogBruteForceDetected logs brute force detection
func (l *Logger) LogBruteForceDetected(ip string, attemptCount int, requestID string) error {
	return l.LogFailure(EventBruteForceDetect, &Actor{Type: "anonymous"},
		&Client{IP: ip, RequestID: requestID},
		"brute force attempt detected",
		map[string]interface{}{
			"attempt_count": attemptCount,
		})
}

// Global convenience functions (use globalLogger)

// AdminLogin logs an admin login event using the global logger
func AdminLogin(adminUsername, ip, userAgent, requestID string, mfaUsed bool) {
	if l := GetLogger(); l != nil {
		l.LogAdminLogin(adminUsername, ip, userAgent, requestID, mfaUsed)
	}
}

// AdminLoginFailed logs a failed admin login using the global logger
func AdminLoginFailed(attemptedUsername, ip, userAgent, requestID, reason string) {
	if l := GetLogger(); l != nil {
		l.LogAdminLoginFailed(attemptedUsername, ip, userAgent, requestID, reason)
	}
}

// RateLimitExceeded logs a rate limit event using the global logger
func RateLimitExceeded(ip, endpoint string, limit int, requestID string) {
	if l := GetLogger(); l != nil {
		l.LogRateLimitExceeded(ip, endpoint, limit, requestID)
	}
}

// CSRFFailure logs a CSRF validation failure using the global logger
func CSRFFailure(ip, endpoint, requestID string) {
	if l := GetLogger(); l != nil {
		l.LogCSRFFailure(ip, endpoint, requestID)
	}
}

// ServerStarted logs server startup using the global logger
func ServerStarted(version, mode string) {
	if l := GetLogger(); l != nil {
		l.LogServerStarted(version, mode)
	}
}

// ServerStopped logs server shutdown using the global logger
func ServerStopped(reason string, uptime time.Duration) {
	if l := GetLogger(); l != nil {
		l.LogServerStopped(reason, uptime)
	}
}

// BruteForceDetected logs brute force detection using the global logger
func BruteForceDetected(ip string, attemptCount int, requestID string) {
	if l := GetLogger(); l != nil {
		l.LogBruteForceDetected(ip, attemptCount, requestID)
	}
}
