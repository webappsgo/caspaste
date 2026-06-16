// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Main TUI application per AI.md PART 33
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents different screens in the TUI
type View int

const (
	ViewDashboard View = iota
	ViewNewPaste
	ViewViewPaste
	ViewListPastes
	ViewSettings
	ViewHelp
)

// AppModel represents the main TUI application state
type AppModel struct {
	serverURL    string
	apiToken     string
	currentView  View
	width        int
	height       int
	menuItems    []MenuItem
	selectedMenu int

	// View-specific data
	pasteList    []PasteItem
	pasteContent string
	pasteInput   *Input
	titleInput   *Input
}

// MenuItem represents a menu item
type MenuItem struct {
	Label string
	View  View
	Key   string
}

// PasteItem represents a paste in the list
type PasteItem struct {
	ID      string
	Title   string
	Syntax  string
	Created string
}

// NewAppModel creates a new TUI application model
func NewAppModel(serverURL, apiToken string) AppModel {
	return AppModel{
		serverURL:   serverURL,
		apiToken:    apiToken,
		currentView: ViewDashboard,
		menuItems: []MenuItem{
			{Label: "Dashboard", View: ViewDashboard, Key: "d"},
			{Label: "New Paste", View: ViewNewPaste, Key: "n"},
			{Label: "List Pastes", View: ViewListPastes, Key: "l"},
			{Label: "Settings", View: ViewSettings, Key: "s"},
			{Label: "Help", View: ViewHelp, Key: "?"},
		},
		pasteInput: NewInput("Enter paste content...", false),
		titleInput: NewInput("Title (optional)", false),
	}
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "d":
			m.currentView = ViewDashboard
		case "n":
			m.currentView = ViewNewPaste
			m.pasteInput.Focused = true
		case "l":
			m.currentView = ViewListPastes
		case "s":
			m.currentView = ViewSettings
		case "?":
			m.currentView = ViewHelp

		case "up", "k":
			if m.selectedMenu > 0 {
				m.selectedMenu--
			}
		case "down", "j":
			if m.selectedMenu < len(m.menuItems)-1 {
				m.selectedMenu++
			}
		case "enter":
			if m.currentView == ViewDashboard {
				m.currentView = m.menuItems[m.selectedMenu].View
			}
		}
	}

	return m, nil
}

// View renders the application
func (m AppModel) View() string {
	// Build the main layout
	var content string

	switch m.currentView {
	case ViewDashboard:
		content = m.dashboardView()
	case ViewNewPaste:
		content = m.newPasteView()
	case ViewListPastes:
		content = m.listPastesView()
	case ViewSettings:
		content = m.settingsView()
	case ViewHelp:
		content = m.helpView()
	default:
		content = m.dashboardView()
	}

	// Header
	header := m.headerView()

	// Footer with help
	footer := m.footerView()

	return header + "\n" + content + "\n" + footer
}

func (m AppModel) headerView() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("CASPB")

	server := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(" • " + m.serverURL)

	return boxStyle.Width(m.width - 4).Render(title + server)
}

func (m AppModel) footerView() string {
	help := "q: quit • d: dashboard • n: new • l: list • s: settings • ?: help"
	return helpStyle.Render(help)
}

func (m AppModel) dashboardView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Dashboard") + "\n\n")

	// Menu
	for i, item := range m.menuItems {
		cursor := "  "
		style := blurredStyle
		if i == m.selectedMenu {
			cursor = "> "
			style = focusedStyle
		}
		b.WriteString(cursor + style.Render(fmt.Sprintf("[%s] %s", item.Key, item.Label)) + "\n")
	}

	b.WriteString("\n" + subtitleStyle.Render("Press enter to select, or use keyboard shortcuts"))

	return b.String()
}

func (m AppModel) newPasteView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("New Paste") + "\n\n")

	b.WriteString("Title:\n")
	b.WriteString(m.titleInput.View() + "\n\n")

	b.WriteString("Content:\n")
	b.WriteString(m.pasteInput.View() + "\n\n")

	b.WriteString(blurredStyle.Render("[ Create Paste ]"))

	return b.String()
}

func (m AppModel) listPastesView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Recent Pastes") + "\n\n")

	if len(m.pasteList) == 0 {
		b.WriteString(subtitleStyle.Render("No pastes found. Create one with 'n'"))
	} else {
		// Header
		b.WriteString(fmt.Sprintf("%-12s %-30s %-12s %s\n", "ID", "TITLE", "SYNTAX", "CREATED"))
		b.WriteString(strings.Repeat("-", 70) + "\n")

		for _, p := range m.pasteList {
			title := p.Title
			if title == "" {
				title = "(untitled)"
			}
			if len(title) > 28 {
				title = title[:25] + "..."
			}
			b.WriteString(fmt.Sprintf("%-12s %-30s %-12s %s\n", p.ID, title, p.Syntax, p.Created))
		}
	}

	return b.String()
}

func (m AppModel) settingsView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Settings") + "\n\n")

	b.WriteString(fmt.Sprintf("Server: %s\n", m.serverURL))
	if m.apiToken != "" {
		b.WriteString("API Token: ******** (configured)\n")
	} else {
		b.WriteString("API Token: (not configured)\n")
	}

	b.WriteString("\n" + subtitleStyle.Render("Edit ~/.config/casapps/caspb/cli.yml to change settings"))

	return b.String()
}

func (m AppModel) helpView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Help") + "\n\n")

	help := []struct {
		key  string
		desc string
	}{
		{"d", "Go to dashboard"},
		{"n", "Create new paste"},
		{"l", "List pastes"},
		{"s", "Settings"},
		{"?", "Show this help"},
		{"q", "Quit"},
		{"", ""},
		{"up/k", "Move up"},
		{"down/j", "Move down"},
		{"enter", "Select"},
		{"tab", "Next field"},
		{"esc", "Cancel/back"},
	}

	for _, h := range help {
		if h.key == "" {
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("  %-10s %s\n", h.key, h.desc))
		}
	}

	return b.String()
}

// RunApp launches the main TUI application
func RunApp(serverURL, apiToken string) error {
	model := NewAppModel(serverURL, apiToken)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
