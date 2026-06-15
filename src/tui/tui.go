// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package tui provides Terminal User Interface support per AI.md PART 33
// Uses charmbracelet/bubbletea for interactive terminal applications
package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	focusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	blurredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	noStyle = lipgloss.NewStyle()

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)
)

// Input represents a text input field
type Input struct {
	Value       string
	Placeholder string
	Focused     bool
	CursorPos   int
	Password    bool
}

// NewInput creates a new input field
func NewInput(placeholder string, password bool) *Input {
	return &Input{
		Placeholder: placeholder,
		Password:    password,
	}
}

// View renders the input field
func (i *Input) View() string {
	var value string
	if i.Password && i.Value != "" {
		value = strings.Repeat("*", len(i.Value))
	} else if i.Value != "" {
		value = i.Value
	} else {
		value = blurredStyle.Render(i.Placeholder)
	}

	style := blurredStyle
	if i.Focused {
		style = focusedStyle
		if i.Value == "" {
			value = cursorStyle.Render("_")
		} else {
			value = value + cursorStyle.Render("_")
		}
	}

	return style.Render("[ " + value + " ]")
}

// HandleKey handles keyboard input
func (i *Input) HandleKey(key string) {
	switch key {
	case "backspace":
		if len(i.Value) > 0 {
			i.Value = i.Value[:len(i.Value)-1]
		}
	default:
		if len(key) == 1 {
			i.Value += key
		}
	}
}

// SetupModel represents the setup wizard state
type SetupModel struct {
	serverURL  *Input
	apiToken   *Input
	focusIndex int
	testing    bool
	testResult string
	testError  string
	done       bool
	cancelled  bool
	width      int
	height     int
}

// NewSetupModel creates a new setup wizard model
func NewSetupModel() SetupModel {
	return SetupModel{
		serverURL:  NewInput("https://paste.example.com", false),
		apiToken:   NewInput("(optional) usr_...", true),
		focusIndex: 0,
	}
}

// Init initializes the setup model
func (m SetupModel) Init() tea.Cmd {
	m.serverURL.Focused = true
	return nil
}

// Update handles messages for the setup wizard
func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "tab", "down":
			m.focusIndex = (m.focusIndex + 1) % 3
			m.updateFocus()

		case "shift+tab", "up":
			m.focusIndex = (m.focusIndex - 1 + 3) % 3
			m.updateFocus()

		case "enter":
			if m.focusIndex == 2 {
				// Test connection button
				m.testing = true
				return m, m.testConnection()
			}

		default:
			// Pass key to focused input
			if m.focusIndex == 0 {
				m.serverURL.HandleKey(msg.String())
			} else if m.focusIndex == 1 {
				m.apiToken.HandleKey(msg.String())
			}
		}

	case testResultMsg:
		m.testing = false
		if msg.err != nil {
			m.testError = msg.err.Error()
			m.testResult = ""
		} else {
			m.testResult = msg.result
			m.testError = ""
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *SetupModel) updateFocus() {
	m.serverURL.Focused = m.focusIndex == 0
	m.apiToken.Focused = m.focusIndex == 1
}

type testResultMsg struct {
	result string
	err    error
}

func (m SetupModel) testConnection() tea.Cmd {
	return func() tea.Msg {
		if m.serverURL.Value == "" {
			return testResultMsg{err: fmt.Errorf("server URL is required")}
		}

		// Validate URL format
		if !strings.HasPrefix(m.serverURL.Value, "http://") && !strings.HasPrefix(m.serverURL.Value, "https://") {
			return testResultMsg{err: fmt.Errorf("URL must start with http:// or https://")}
		}

		// Test connection via health endpoint
		client := &http.Client{Timeout: 10 * time.Second}
		healthURL := strings.TrimRight(m.serverURL.Value, "/") + "/api/v1/healthz"
		resp, err := client.Get(healthURL)
		if err != nil {
			return testResultMsg{err: fmt.Errorf("connection failed: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return testResultMsg{err: fmt.Errorf("server returned status %d", resp.StatusCode)}
		}

		var healthResp struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
			return testResultMsg{result: "Connected (could not parse health response)"}
		}

		if healthResp.Version != "" {
			return testResultMsg{result: fmt.Sprintf("Connected to CasPb %s (status: %s)", healthResp.Version, healthResp.Status)}
		}
		return testResultMsg{result: fmt.Sprintf("Connected (status: %s)", healthResp.Status)}
	}
}

// View renders the setup wizard
func (m SetupModel) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("CASPB CLI SETUP")
	subtitle := subtitleStyle.Render("Configure your server connection")

	b.WriteString(title + "\n")
	b.WriteString(subtitle + "\n\n")

	// Server URL field
	label := "Server URL:"
	if m.focusIndex == 0 {
		label = focusedStyle.Render(label)
	}
	b.WriteString(label + "\n")
	b.WriteString(m.serverURL.View() + "\n\n")

	// API Token field
	label = "API Token (optional):"
	if m.focusIndex == 1 {
		label = focusedStyle.Render(label)
	}
	b.WriteString(label + "\n")
	b.WriteString(m.apiToken.View() + "\n\n")

	// Test button
	buttonStyle := blurredStyle
	if m.focusIndex == 2 {
		buttonStyle = focusedStyle
	}

	if m.testing {
		b.WriteString(buttonStyle.Render("[ Testing... ]") + "\n")
	} else {
		b.WriteString(buttonStyle.Render("[ Test Connection ]") + "\n")
	}

	// Status messages
	if m.testError != "" {
		b.WriteString("\n" + errorStyle.Render("Error: "+m.testError))
	}
	if m.testResult != "" {
		b.WriteString("\n" + successStyle.Render(m.testResult))
	}

	// Help
	b.WriteString("\n" + helpStyle.Render("Tab: next field • Enter: test • Esc: cancel"))

	return boxStyle.Render(b.String())
}

// GetServerURL returns the configured server URL
func (m SetupModel) GetServerURL() string {
	return m.serverURL.Value
}

// GetAPIToken returns the configured API token
func (m SetupModel) GetAPIToken() string {
	return m.apiToken.Value
}

// IsCancelled returns true if setup was cancelled
func (m SetupModel) IsCancelled() bool {
	return m.cancelled
}

// IsDone returns true if setup completed successfully
func (m SetupModel) IsDone() bool {
	return m.done
}

// RunSetupWizard launches the TUI setup wizard
func RunSetupWizard() (*SetupResult, error) {
	model := NewSetupModel()
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	m := finalModel.(SetupModel)
	if m.IsCancelled() {
		return nil, fmt.Errorf("setup cancelled")
	}

	return &SetupResult{
		ServerURL: m.GetServerURL(),
		APIToken:  m.GetAPIToken(),
	}, nil
}

// SetupResult holds the result of the setup wizard
type SetupResult struct {
	ServerURL string
	APIToken  string
}
