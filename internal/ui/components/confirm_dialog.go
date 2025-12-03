package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/doganarif/k9sight/internal/ui/styles"
)

// ConfirmDialog is a modal confirmation dialog
type ConfirmDialog struct {
	title    string
	message  string
	visible  bool
	selected bool // true = confirm (yes), false = cancel (no)
	action   string
	data     interface{}
}

// ConfirmResult is returned when a confirmation is made
type ConfirmResult struct {
	Confirmed bool
	Action    string
	Data      interface{}
}

func NewConfirmDialog() ConfirmDialog {
	return ConfirmDialog{
		selected: false, // Default to "No" for safety
	}
}

func (c ConfirmDialog) Init() tea.Cmd {
	return nil
}

func (c ConfirmDialog) Update(msg tea.Msg) (ConfirmDialog, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "n", "N":
			c.visible = false
			return c, func() tea.Msg {
				return ConfirmResult{Confirmed: false, Action: c.action, Data: c.data}
			}

		case "enter":
			c.visible = false
			return c, func() tea.Msg {
				return ConfirmResult{Confirmed: c.selected, Action: c.action, Data: c.data}
			}

		case "y", "Y":
			c.visible = false
			return c, func() tea.Msg {
				return ConfirmResult{Confirmed: true, Action: c.action, Data: c.data}
			}

		case "left", "h":
			c.selected = true // Yes is on left

		case "right", "l":
			c.selected = false // No is on right

		case "tab":
			c.selected = !c.selected
		}
	}

	return c, nil
}

func (c ConfirmDialog) View() string {
	if !c.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Warning).
		MarginBottom(1)
	b.WriteString(titleStyle.Render(c.title))
	b.WriteString("\n\n")

	// Message
	msgStyle := lipgloss.NewStyle().Foreground(styles.Text)
	b.WriteString(msgStyle.Render(c.message))
	b.WriteString("\n\n")

	// Buttons
	yesStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder())
	noStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder())

	if c.selected {
		yesStyle = yesStyle.
			BorderForeground(styles.Warning).
			Foreground(styles.Warning).
			Bold(true)
		noStyle = noStyle.
			BorderForeground(styles.Muted).
			Foreground(styles.Muted)
	} else {
		yesStyle = yesStyle.
			BorderForeground(styles.Muted).
			Foreground(styles.Muted)
		noStyle = noStyle.
			BorderForeground(styles.Primary).
			Foreground(styles.Primary).
			Bold(true)
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		yesStyle.Render("Yes"),
		"  ",
		noStyle.Render("No"),
	)
	b.WriteString(buttons)

	// Hint
	hintStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		MarginTop(1)
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("y/n • ←/→ to select • Enter to confirm"))

	// Wrap in a box
	content := b.String()
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Warning).
		Padding(1, 2).
		Background(styles.Background)

	return boxStyle.Render(content)
}

func (c *ConfirmDialog) Show(title, message, action string, data interface{}) {
	c.title = title
	c.message = message
	c.action = action
	c.data = data
	c.selected = false // Default to No for safety
	c.visible = true
}

func (c *ConfirmDialog) Hide() {
	c.visible = false
}

func (c ConfirmDialog) IsVisible() bool {
	return c.visible
}
