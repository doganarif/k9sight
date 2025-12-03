package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/doganarif/k9sight/internal/ui/styles"
)

// MenuItem represents a single menu item
type MenuItem struct {
	Label    string
	Value    string // The command or value to copy/execute
	Shortcut string // Single key shortcut (1-9)
}

// ActionMenu is a popup menu for actions
type ActionMenu struct {
	title    string
	items    []MenuItem
	selected int
	visible  bool
	width    int
	height   int
}

// ActionMenuResult is returned when an action is selected
type ActionMenuResult struct {
	Item    MenuItem
	Copied  bool
	Err     error
}

func NewActionMenu() ActionMenu {
	return ActionMenu{
		selected: 0,
	}
}

func (m ActionMenu) Init() tea.Cmd {
	return nil
}

func (m ActionMenu) Update(msg tea.Msg) (ActionMenu, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc" || msg.String() == "q":
			m.visible = false
			return m, nil

		case msg.String() == "up" || msg.String() == "k":
			if m.selected > 0 {
				m.selected--
			}

		case msg.String() == "down" || msg.String() == "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}

		case msg.String() == "enter":
			if m.selected >= 0 && m.selected < len(m.items) {
				item := m.items[m.selected]
				err := CopyToClipboard(item.Value)
				m.visible = false
				return m, func() tea.Msg {
					return ActionMenuResult{Item: item, Copied: true, Err: err}
				}
			}

		default:
			// Check for number shortcuts (1-9)
			if len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9' {
				idx := int(msg.String()[0] - '1')
				if idx < len(m.items) {
					item := m.items[idx]
					err := CopyToClipboard(item.Value)
					m.visible = false
					return m, func() tea.Msg {
						return ActionMenuResult{Item: item, Copied: true, Err: err}
					}
				}
			}
		}
	}

	return m, nil
}

func (m ActionMenu) View() string {
	if !m.visible || len(m.items) == 0 {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		MarginBottom(1)
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Items
	for i, item := range m.items {
		shortcut := fmt.Sprintf("[%d] ", i+1)
		shortcutStyle := lipgloss.NewStyle().Foreground(styles.Secondary)

		if i == m.selected {
			// Selected item
			selectedStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(styles.Background).
				Background(styles.Primary)
			b.WriteString(shortcutStyle.Render(shortcut))
			b.WriteString(selectedStyle.Render(item.Label))
		} else {
			// Normal item
			normalStyle := lipgloss.NewStyle().Foreground(styles.Text)
			b.WriteString(shortcutStyle.Render(shortcut))
			b.WriteString(normalStyle.Render(item.Label))
		}
		b.WriteString("\n")
	}

	// Footer hint
	hintStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		MarginTop(1)
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press number or Enter to copy • Esc to close"))

	// Wrap in a box
	content := b.String()
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Background(styles.Background)

	return boxStyle.Render(content)
}

func (m *ActionMenu) Show(title string, items []MenuItem) {
	m.title = title
	m.items = items
	m.selected = 0
	m.visible = true
}

func (m *ActionMenu) Hide() {
	m.visible = false
}

func (m ActionMenu) IsVisible() bool {
	return m.visible
}

func (m *ActionMenu) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// KubectlCommands generates common kubectl commands for a pod
func KubectlCommands(namespace, podName, containerName string, containers []string) []MenuItem {
	items := []MenuItem{
		{
			Label: "Get pod logs",
			Value: fmt.Sprintf("kubectl logs -n %s %s", namespace, podName),
		},
		{
			Label: "Get pod logs (follow)",
			Value: fmt.Sprintf("kubectl logs -n %s %s -f", namespace, podName),
		},
		{
			Label: "Describe pod",
			Value: fmt.Sprintf("kubectl describe pod -n %s %s", namespace, podName),
		},
		{
			Label: "Get pod YAML",
			Value: fmt.Sprintf("kubectl get pod -n %s %s -o yaml", namespace, podName),
		},
		{
			Label: "Exec into pod (sh)",
			Value: fmt.Sprintf("kubectl exec -it -n %s %s -- sh", namespace, podName),
		},
		{
			Label: "Exec into pod (bash)",
			Value: fmt.Sprintf("kubectl exec -it -n %s %s -- bash", namespace, podName),
		},
		{
			Label: "Delete pod",
			Value: fmt.Sprintf("kubectl delete pod -n %s %s", namespace, podName),
		},
	}

	// Add container-specific commands if multiple containers
	if len(containers) > 1 && containerName != "" {
		items = append([]MenuItem{
			{
				Label: fmt.Sprintf("Logs for container '%s'", containerName),
				Value: fmt.Sprintf("kubectl logs -n %s %s -c %s", namespace, podName, containerName),
			},
			{
				Label: fmt.Sprintf("Exec into '%s' (sh)", containerName),
				Value: fmt.Sprintf("kubectl exec -it -n %s %s -c %s -- sh", namespace, podName, containerName),
			},
		}, items...)
	}

	// Add previous logs option
	if containerName != "" {
		items = append(items, MenuItem{
			Label: "Get previous container logs",
			Value: fmt.Sprintf("kubectl logs -n %s %s -c %s --previous", namespace, podName, containerName),
		})
	} else if len(containers) > 0 {
		items = append(items, MenuItem{
			Label: "Get previous container logs",
			Value: fmt.Sprintf("kubectl logs -n %s %s -c %s --previous", namespace, podName, containers[0]),
		})
	}

	return items
}

// Verify ActionMenu satisfies common patterns
func (m ActionMenu) Matches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}

// PodActionItem represents an action that can be taken on a pod
type PodActionItem struct {
	Label       string
	Description string
	Action      string // "delete", "exec", "port-forward", "copy"
	Command     string // kubectl command if applicable
}

// PodActionMenuResult is returned when a pod action is selected
type PodActionMenuResult struct {
	Item PodActionItem
}

// PodActionMenu is similar to ActionMenu but for pod actions
type PodActionMenu struct {
	title    string
	items    []PodActionItem
	selected int
	visible  bool
}

func NewPodActionMenu() PodActionMenu {
	return PodActionMenu{
		selected: 0,
	}
}

func (m PodActionMenu) Init() tea.Cmd {
	return nil
}

func (m PodActionMenu) Update(msg tea.Msg) (PodActionMenu, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc" || msg.String() == "q":
			m.visible = false
			return m, nil

		case msg.String() == "up" || msg.String() == "k":
			if m.selected > 0 {
				m.selected--
			}

		case msg.String() == "down" || msg.String() == "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}

		case msg.String() == "enter":
			if m.selected >= 0 && m.selected < len(m.items) {
				item := m.items[m.selected]
				m.visible = false
				return m, func() tea.Msg {
					return PodActionMenuResult{Item: item}
				}
			}

		default:
			// Check for number shortcuts (1-9)
			if len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9' {
				idx := int(msg.String()[0] - '1')
				if idx < len(m.items) {
					item := m.items[idx]
					m.visible = false
					return m, func() tea.Msg {
						return PodActionMenuResult{Item: item}
					}
				}
			}
		}
	}

	return m, nil
}

func (m PodActionMenu) View() string {
	if !m.visible || len(m.items) == 0 {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary).
		MarginBottom(1)
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Items
	for i, item := range m.items {
		shortcut := fmt.Sprintf("[%d] ", i+1)
		shortcutStyle := lipgloss.NewStyle().Foreground(styles.Secondary)

		if i == m.selected {
			// Selected item
			selectedStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(styles.Background).
				Background(styles.Primary)
			descStyle := lipgloss.NewStyle().
				Foreground(styles.TextMuted).
				Italic(true)
			b.WriteString(shortcutStyle.Render(shortcut))
			b.WriteString(selectedStyle.Render(item.Label))
			if item.Description != "" {
				b.WriteString(" ")
				b.WriteString(descStyle.Render(item.Description))
			}
		} else {
			// Normal item
			normalStyle := lipgloss.NewStyle().Foreground(styles.Text)
			descStyle := lipgloss.NewStyle().Foreground(styles.Muted)
			b.WriteString(shortcutStyle.Render(shortcut))
			b.WriteString(normalStyle.Render(item.Label))
			if item.Description != "" {
				b.WriteString(" ")
				b.WriteString(descStyle.Render(item.Description))
			}
		}
		b.WriteString("\n")
	}

	// Footer hint
	hintStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		MarginTop(1)
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press number or Enter to select • Esc to close"))

	// Wrap in a box
	content := b.String()
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Background(styles.Background)

	return boxStyle.Render(content)
}

func (m *PodActionMenu) Show(title string, items []PodActionItem) {
	m.title = title
	m.items = items
	m.selected = 0
	m.visible = true
}

func (m *PodActionMenu) Hide() {
	m.visible = false
}

func (m PodActionMenu) IsVisible() bool {
	return m.visible
}

// WorkloadActionItem represents an action for workloads
type WorkloadActionItem struct {
	Label       string
	Description string
	Action      string // "scale", "restart", "copy"
	Replicas    int32  // For scale actions
	Command     string // kubectl command
}

// WorkloadActionMenuResult is returned when a workload action is selected
type WorkloadActionMenuResult struct {
	Item WorkloadActionItem
}

// WorkloadActionMenu for workload actions
type WorkloadActionMenu struct {
	title    string
	items    []WorkloadActionItem
	selected int
	visible  bool
}

func NewWorkloadActionMenu() WorkloadActionMenu {
	return WorkloadActionMenu{selected: 0}
}

func (m WorkloadActionMenu) Init() tea.Cmd { return nil }

func (m WorkloadActionMenu) Update(msg tea.Msg) (WorkloadActionMenu, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "esc" || msg.String() == "q":
			m.visible = false
			return m, nil
		case msg.String() == "up" || msg.String() == "k":
			if m.selected > 0 {
				m.selected--
			}
		case msg.String() == "down" || msg.String() == "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case msg.String() == "enter":
			if m.selected >= 0 && m.selected < len(m.items) {
				item := m.items[m.selected]
				m.visible = false
				return m, func() tea.Msg {
					return WorkloadActionMenuResult{Item: item}
				}
			}
		default:
			if len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9' {
				idx := int(msg.String()[0] - '1')
				if idx < len(m.items) {
					item := m.items[idx]
					m.visible = false
					return m, func() tea.Msg {
						return WorkloadActionMenuResult{Item: item}
					}
				}
			}
		}
	}
	return m, nil
}

func (m WorkloadActionMenu) View() string {
	if !m.visible || len(m.items) == 0 {
		return ""
	}

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).MarginBottom(1)
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	for i, item := range m.items {
		shortcut := fmt.Sprintf("[%d] ", i+1)
		shortcutStyle := lipgloss.NewStyle().Foreground(styles.Secondary)

		if i == m.selected {
			selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Background).Background(styles.Primary)
			descStyle := lipgloss.NewStyle().Foreground(styles.TextMuted).Italic(true)
			b.WriteString(shortcutStyle.Render(shortcut))
			b.WriteString(selectedStyle.Render(item.Label))
			if item.Description != "" {
				b.WriteString(" ")
				b.WriteString(descStyle.Render(item.Description))
			}
		} else {
			normalStyle := lipgloss.NewStyle().Foreground(styles.Text)
			descStyle := lipgloss.NewStyle().Foreground(styles.Muted)
			b.WriteString(shortcutStyle.Render(shortcut))
			b.WriteString(normalStyle.Render(item.Label))
			if item.Description != "" {
				b.WriteString(" ")
				b.WriteString(descStyle.Render(item.Description))
			}
		}
		b.WriteString("\n")
	}

	hintStyle := lipgloss.NewStyle().Foreground(styles.Muted).MarginTop(1)
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press number or Enter to select • Esc to close"))

	content := b.String()
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2).
		Background(styles.Background)
	return boxStyle.Render(content)
}

func (m *WorkloadActionMenu) Show(title string, items []WorkloadActionItem) {
	m.title = title
	m.items = items
	m.selected = 0
	m.visible = true
}

func (m *WorkloadActionMenu) Hide() { m.visible = false }
func (m WorkloadActionMenu) IsVisible() bool { return m.visible }

// ScaleActions returns scale options for a workload
func ScaleActions(namespace, name, resourceType string, currentReplicas int32) []WorkloadActionItem {
	items := []WorkloadActionItem{
		{Label: "Scale to 0", Action: "scale", Replicas: 0},
		{Label: "Scale to 1", Action: "scale", Replicas: 1},
		{Label: "Scale to 2", Action: "scale", Replicas: 2},
		{Label: "Scale to 3", Action: "scale", Replicas: 3},
		{Label: "Scale to 5", Action: "scale", Replicas: 5},
	}

	// Add current+1 and current-1 if not in list
	if currentReplicas > 0 {
		items = append([]WorkloadActionItem{
			{Label: fmt.Sprintf("Scale to %d (current-1)", currentReplicas-1), Action: "scale", Replicas: currentReplicas - 1},
		}, items...)
	}
	if currentReplicas < 10 {
		items = append(items, WorkloadActionItem{
			Label: fmt.Sprintf("Scale to %d (current+1)", currentReplicas+1), Action: "scale", Replicas: currentReplicas + 1,
		})
	}

	// Add copy command option
	items = append(items, WorkloadActionItem{
		Label:   "Copy scale command",
		Action:  "copy",
		Command: fmt.Sprintf("kubectl scale %s/%s -n %s --replicas=", resourceType, name, namespace),
	})

	return items
}

// PodActions returns the available actions for a pod
func PodActions(namespace, podName string, containers []string) []PodActionItem {
	items := []PodActionItem{
		{
			Label:       "Delete Pod",
			Description: "(requires confirmation)",
			Action:      "delete",
			Command:     fmt.Sprintf("kubectl delete pod -n %s %s", namespace, podName),
		},
	}

	// Add exec options
	if len(containers) == 1 {
		items = append(items, PodActionItem{
			Label:       "Exec (sh)",
			Description: "opens shell in terminal",
			Action:      "exec",
			Command:     fmt.Sprintf("kubectl exec -it -n %s %s -- sh", namespace, podName),
		})
		items = append(items, PodActionItem{
			Label:       "Exec (bash)",
			Description: "opens shell in terminal",
			Action:      "exec",
			Command:     fmt.Sprintf("kubectl exec -it -n %s %s -- bash", namespace, podName),
		})
	} else if len(containers) > 1 {
		// Multi-container pod - exec into first container by default
		for _, container := range containers {
			items = append(items, PodActionItem{
				Label:       fmt.Sprintf("Exec into '%s' (sh)", container),
				Description: "opens shell in terminal",
				Action:      "exec",
				Command:     fmt.Sprintf("kubectl exec -it -n %s %s -c %s -- sh", namespace, podName, container),
			})
		}
	}

	// Add port-forward option - runs in foreground (Ctrl+C to return)
	items = append(items, PodActionItem{
		Label:       "Port Forward :8080",
		Description: "runs in terminal, Ctrl+C to stop",
		Action:      "port-forward",
		Command:     fmt.Sprintf("kubectl port-forward -n %s %s 8080:8080", namespace, podName),
	})

	// Add describe - runs and shows output
	items = append(items, PodActionItem{
		Label:       "Describe Pod",
		Description: "shows pod details",
		Action:      "describe",
		Command:     fmt.Sprintf("kubectl describe pod -n %s %s", namespace, podName),
	})

	// Copy commands section
	items = append(items, PodActionItem{
		Label:       "Copy logs command",
		Description: "to clipboard",
		Action:      "copy",
		Command:     fmt.Sprintf("kubectl logs -n %s %s -f", namespace, podName),
	})

	return items
}
