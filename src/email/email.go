// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package email provides SMTP email support per AI.md PART 18
// All emails require a valid and working SMTP server
package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds SMTP configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      string
	FromName string
	FromEmail string
}

// Client handles email sending
type Client struct {
	config   *Config
	enabled  bool
	mu       sync.RWMutex
}

// NewClient creates a new email client
func NewClient(cfg *Config) *Client {
	c := &Client{
		config:  cfg,
		enabled: false,
	}

	// Apply environment variable overrides
	c.applyEnvOverrides()

	return c
}

// applyEnvOverrides applies SMTP_* environment variables
func (c *Client) applyEnvOverrides() {
	if host := os.Getenv("SMTP_HOST"); host != "" {
		c.config.Host = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.config.Port = p
		}
	}
	if username := os.Getenv("SMTP_USERNAME"); username != "" {
		c.config.Username = username
	}
	if password := os.Getenv("SMTP_PASSWORD"); password != "" {
		c.config.Password = password
	}
	if tlsMode := os.Getenv("SMTP_TLS"); tlsMode != "" {
		c.config.TLS = tlsMode
	}
	if fromName := os.Getenv("SMTP_FROM_NAME"); fromName != "" {
		c.config.FromName = fromName
	}
	if fromEmail := os.Getenv("SMTP_FROM_EMAIL"); fromEmail != "" {
		c.config.FromEmail = fromEmail
	}
}

// IsEnabled returns true if email is enabled (SMTP configured and working)
func (c *Client) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// TestConnection tests the SMTP connection
func (c *Client) TestConnection() error {
	if c.config.Host == "" {
		return fmt.Errorf("SMTP host not configured")
	}

	addr := net.JoinHostPort(c.config.Host, fmt.Sprintf("%d", c.config.Port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Simple SMTP handshake
	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Try EHLO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP EHLO failed: %w", err)
	}

	c.mu.Lock()
	c.enabled = true
	c.mu.Unlock()

	return nil
}

// AutoDetect attempts to find a working SMTP server
func (c *Client) AutoDetect(fqdn string) error {
	// Priority order per AI.md PART 18
	hosts := []string{
		"127.0.0.1",
		"172.17.0.1",
	}

	// Add gateway IP if available
	if gw := getGatewayIP(); gw != "" {
		hosts = append(hosts, gw)
	}

	// Add FQDN variants
	if fqdn != "" && fqdn != "localhost" {
		hosts = append(hosts, fqdn)
		hosts = append(hosts, "mail."+fqdn)
		hosts = append(hosts, "smtp."+fqdn)
	}

	// Ports to try
	ports := []int{587, 465, 25}

	for _, host := range hosts {
		for _, port := range ports {
			c.config.Host = host
			c.config.Port = port

			if err := c.TestConnection(); err == nil {
				return nil
			}
		}
	}

	// Reset on failure
	c.config.Host = ""
	c.config.Port = 587
	c.mu.Lock()
	c.enabled = false
	c.mu.Unlock()

	return fmt.Errorf("no SMTP server found")
}

// getGatewayIP attempts to find the default gateway IP
func getGatewayIP() string {
	// Connect to a public IP to find local interface
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	// Assume gateway is .1 on the same subnet
	ip := localAddr.IP.To4()
	if ip == nil {
		return ""
	}
	ip[3] = 1
	return ip.String()
}

// Send sends an email
func (c *Client) Send(to, subject, body string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("email not enabled: SMTP not configured")
	}

	from := c.config.FromEmail
	if from == "" {
		return fmt.Errorf("from email not configured")
	}

	// Build message
	msg := buildMessage(from, c.config.FromName, to, subject, body)

	// Connect and send
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	var auth smtp.Auth
	if c.config.Username != "" {
		auth = smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)
	}

	// Handle TLS mode
	switch strings.ToLower(c.config.TLS) {
	case "tls", "ssl":
		return c.sendWithTLS(addr, auth, from, to, msg)
	case "starttls":
		return c.sendWithStartTLS(addr, auth, from, to, msg)
	case "none":
		return smtp.SendMail(addr, auth, from, []string{to}, msg)
	default:
		// Auto mode: try StartTLS first, fall back to plain
		if err := c.sendWithStartTLS(addr, auth, from, to, msg); err == nil {
			return nil
		}
		return smtp.SendMail(addr, auth, from, []string{to}, msg)
	}
}

// sendWithTLS sends email over implicit TLS (port 465)
func (c *Client) sendWithTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

// sendWithStartTLS sends email using STARTTLS
func (c *Client) sendWithStartTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	// Try STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: c.config.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}

// buildMessage builds an RFC 2822 compliant email message
func buildMessage(from, fromName, to, subject, body string) []byte {
	var msg strings.Builder

	// From header
	if fromName != "" {
		msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	} else {
		msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	}

	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return []byte(msg.String())
}

// GetConfig returns the current SMTP configuration (for display)
func (c *Client) GetConfig() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"host":      c.config.Host,
		"port":      c.config.Port,
		"username":  c.config.Username,
		"tls":       c.config.TLS,
		"from_name": c.config.FromName,
		"from_email": c.config.FromEmail,
		"enabled":   c.enabled,
	}
}
