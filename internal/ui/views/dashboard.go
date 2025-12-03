package views

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/doganarif/k9sight/internal/k8s"
	"github.com/doganarif/k9sight/internal/ui/components"
	"github.com/doganarif/k9sight/internal/ui/keys"
	"github.com/doganarif/k9sight/internal/ui/styles"
)

type PanelFocus int

const (
	FocusLogs PanelFocus = iota
	FocusEvents
	FocusMetrics
	FocusManifest
)

type Dashboard struct {
	pod           *k8s.PodInfo
	logs          components.LogsPanel
	events        components.EventsPanel
	metrics       components.MetricsPanel
	manifest      components.ManifestPanel
	statusBar     components.StatusBar
	breadcrumb    components.Breadcrumb
	help          components.HelpPanel
	actionMenu    components.ActionMenu
	podActionMenu components.PodActionMenu
	confirmDialog components.ConfirmDialog
	resultViewer  components.ResultViewer
	focus         PanelFocus
	fullscreen    bool
	width         int
	height        int
	keys          keys.KeyMap
	statusMsg     string // Temporary status message (e.g., "Copied!")
	namespace     string // Current namespace for kubectl commands
	context       string // Current context for kubectl commands
	pendingAction *components.PodActionItem // Action waiting for confirmation
}

func NewDashboard() Dashboard {
	return Dashboard{
		logs:          components.NewLogsPanel(),
		events:        components.NewEventsPanel(),
		metrics:       components.NewMetricsPanel(),
		manifest:      components.NewManifestPanel(),
		statusBar:     components.NewStatusBar(),
		breadcrumb:    components.NewBreadcrumb(),
		help:          components.NewHelpPanel(),
		actionMenu:    components.NewActionMenu(),
		podActionMenu: components.NewPodActionMenu(),
		confirmDialog: components.NewConfirmDialog(),
		resultViewer:  components.NewResultViewer(),
		focus:         FocusLogs,
		keys:          keys.DefaultKeyMap(),
	}
}

func (d Dashboard) Init() tea.Cmd {
	return nil
}

// DeletePodRequest is sent to app.go to request pod deletion
type DeletePodRequest struct {
	Namespace string
	PodName   string
}

// ExecFinishedMsg is sent when an external command finishes
type ExecFinishedMsg struct {
	Err error
}

// DescribeOutputMsg contains the output of kubectl describe
type DescribeOutputMsg struct {
	Title   string
	Content string
	Err     error
}

func (d Dashboard) Update(msg tea.Msg) (Dashboard, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle ExecFinishedMsg (after external command returns)
	if result, ok := msg.(ExecFinishedMsg); ok {
		if result.Err != nil {
			d.statusMsg = "Command failed: " + result.Err.Error()
		} else {
			d.statusMsg = "Command completed"
		}
		return d, nil
	}

	// Handle DescribeOutputMsg (display describe output in result viewer)
	if result, ok := msg.(DescribeOutputMsg); ok {
		if result.Err != nil {
			d.statusMsg = "Describe failed: " + result.Err.Error()
		} else {
			d.resultViewer.Show(result.Title, result.Content, d.width-4, d.height-4)
		}
		return d, nil
	}

	// Handle ActionMenuResult (copy commands)
	if result, ok := msg.(components.ActionMenuResult); ok {
		if result.Copied && result.Err == nil {
			d.statusMsg = "Copied: " + result.Item.Label
		} else if result.Err != nil {
			d.statusMsg = "Copy failed: " + result.Err.Error()
		}
		return d, nil
	}

	// Handle PodActionMenuResult
	if result, ok := msg.(components.PodActionMenuResult); ok {
		switch result.Item.Action {
		case "delete":
			// Show confirmation dialog
			d.confirmDialog.Show(
				"Delete Pod",
				"Are you sure you want to delete pod '"+d.pod.Name+"'?",
				"delete",
				d.pod,
			)
			return d, nil
		case "exec":
			// Show confirmation before exec
			d.pendingAction = &result.Item
			d.confirmDialog.Show(
				"Exec into Pod",
				"Open shell in '"+d.pod.Name+"'?\nThis will suspend the UI until you exit the shell.",
				"exec",
				d.pod,
			)
			return d, nil
		case "port-forward":
			// Show confirmation before port-forward
			d.pendingAction = &result.Item
			d.confirmDialog.Show(
				"Port Forward",
				"Start port forwarding for '"+d.pod.Name+"'?\nPress Ctrl+C in terminal to stop and return.",
				"port-forward",
				d.pod,
			)
			return d, nil
		case "describe":
			// Run describe command and capture output
			d.statusMsg = "Loading describe..."
			cmdStr := result.Item.Command
			podName := d.pod.Name
			return d, func() tea.Msg {
				c := exec.Command("sh", "-c", cmdStr)
				output, err := c.CombinedOutput()
				if err != nil {
					return DescribeOutputMsg{Err: err}
				}
				return DescribeOutputMsg{
					Title:   "Pod: " + podName,
					Content: string(output),
				}
			}
		case "copy":
			// Copy the command to clipboard
			err := components.CopyToClipboard(result.Item.Command)
			if err == nil {
				d.statusMsg = "Copied: " + result.Item.Label
			} else {
				d.statusMsg = "Copy failed: " + err.Error()
			}
			return d, nil
		}
		return d, nil
	}

	// Handle ConfirmResult
	if result, ok := msg.(components.ConfirmResult); ok {
		if result.Confirmed {
			switch result.Action {
			case "delete":
				if pod, ok := result.Data.(*k8s.PodInfo); ok {
					d.statusMsg = "Deleting pod..."
					return d, func() tea.Msg {
						return DeletePodRequest{
							Namespace: pod.Namespace,
							PodName:   pod.Name,
						}
					}
				}
			case "exec", "port-forward":
				// Execute the pending action
				if d.pendingAction != nil {
					cmdStr := d.pendingAction.Command
					d.pendingAction = nil
					c := exec.Command("sh", "-c", cmdStr)
					return d, tea.ExecProcess(c, func(err error) tea.Msg {
						if err != nil {
							return ExecFinishedMsg{Err: err}
						}
						return ExecFinishedMsg{}
					})
				}
			}
		} else {
			// Cancelled - clear pending action
			d.pendingAction = nil
		}
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Confirm dialog takes highest priority
		if d.confirmDialog.IsVisible() {
			d.confirmDialog, cmd = d.confirmDialog.Update(msg)
			return d, cmd
		}

		// Result viewer takes priority (for describe output etc)
		if d.resultViewer.IsVisible() {
			d.resultViewer, cmd = d.resultViewer.Update(msg)
			return d, cmd
		}

		// Pod action menu takes priority
		if d.podActionMenu.IsVisible() {
			d.podActionMenu, cmd = d.podActionMenu.Update(msg)
			return d, cmd
		}

		// Action menu (copy commands) takes priority
		if d.actionMenu.IsVisible() {
			d.actionMenu, cmd = d.actionMenu.Update(msg)
			return d, cmd
		}

		if d.help.IsVisible() {
			if msg.String() == "?" || msg.String() == "esc" {
				d.help.Hide()
				return d, nil
			}
			return d, nil
		}

		// When logs panel is searching, pass all keys to it (except esc/enter handled above)
		if d.focus == FocusLogs && d.logs.IsSearching() {
			d.logs, cmd = d.logs.Update(msg)
			return d, cmd
		}

		// Clear status message on any key press
		d.statusMsg = ""

		switch {
		case key.Matches(msg, d.keys.PodActions):
			if d.pod != nil {
				var containers []string
				for _, c := range d.pod.Containers {
					containers = append(containers, c.Name)
				}
				items := components.PodActions(d.namespace, d.pod.Name, containers)
				d.podActionMenu.Show("Pod Actions", items)
			}
			return d, nil

		case key.Matches(msg, d.keys.CopyCommands):
			if d.pod != nil {
				var containers []string
				for _, c := range d.pod.Containers {
					containers = append(containers, c.Name)
				}
				selectedContainer := d.logs.SelectedContainer()
				items := components.KubectlCommands(d.namespace, d.pod.Name, selectedContainer, containers)
				d.actionMenu.Show("Copy kubectl command", items)
			}
			return d, nil

		case key.Matches(msg, d.keys.Help):
			d.help.Toggle()
			return d, nil

		case key.Matches(msg, d.keys.NextPanel):
			d.nextPanel()
			return d, nil

		case key.Matches(msg, d.keys.PrevPanel):
			d.prevPanel()
			return d, nil

		case key.Matches(msg, d.keys.Panel1):
			d.focus = FocusLogs
			return d, nil

		case key.Matches(msg, d.keys.Panel2):
			d.focus = FocusEvents
			return d, nil

		case key.Matches(msg, d.keys.Panel3):
			d.focus = FocusMetrics
			return d, nil

		case key.Matches(msg, d.keys.Panel4):
			d.focus = FocusManifest
			return d, nil

		case key.Matches(msg, d.keys.ToggleFullView):
			d.fullscreen = !d.fullscreen
			return d, nil
		}
	}

	switch d.focus {
	case FocusLogs:
		d.logs, cmd = d.logs.Update(msg)
		cmds = append(cmds, cmd)
	case FocusEvents:
		d.events, cmd = d.events.Update(msg)
		cmds = append(cmds, cmd)
	case FocusMetrics:
		d.metrics, cmd = d.metrics.Update(msg)
		cmds = append(cmds, cmd)
	case FocusManifest:
		d.manifest, cmd = d.manifest.Update(msg)
		cmds = append(cmds, cmd)
	}

	return d, tea.Batch(cmds...)
}

func (d *Dashboard) nextPanel() {
	d.focus = (d.focus + 1) % 4
}

func (d *Dashboard) prevPanel() {
	d.focus = (d.focus + 3) % 4
}

func (d Dashboard) View() string {
	if d.pod == nil {
		return styles.PanelStyle.Render("No pod selected")
	}

	var b strings.Builder

	// Show breadcrumb with optional status message
	breadcrumbView := d.breadcrumb.View()
	if d.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(styles.Success).
			Bold(true)
		breadcrumbView = breadcrumbView + "  " + statusStyle.Render(d.statusMsg)
	}
	b.WriteString(breadcrumbView)
	b.WriteString("\n")

	if d.fullscreen {
		// Render only the focused panel in fullscreen
		b.WriteString(d.renderFullscreenPanel())
	} else {
		// Normal 4-panel layout
		topRow := d.renderTopRow()
		bottomRow := d.renderBottomRow()

		b.WriteString(topRow)
		b.WriteString("\n")
		b.WriteString(bottomRow)
	}

	content := b.String()

	// Render confirm dialog as overlay (highest priority)
	if d.confirmDialog.IsVisible() {
		return d.renderFloatingDialog(d.confirmDialog.View())
	}

	// Render result viewer as overlay (for describe output etc)
	if d.resultViewer.IsVisible() {
		return d.renderFloatingDialog(d.resultViewer.View())
	}

	// Render pod action menu as overlay
	if d.podActionMenu.IsVisible() {
		return d.renderFloatingDialog(d.podActionMenu.View())
	}

	// Render action menu as overlay if visible
	if d.actionMenu.IsVisible() {
		return d.renderFloatingMenu(content)
	}

	if d.help.IsVisible() {
		return d.renderFloatingHelp(content)
	}

	return content
}

func (d Dashboard) renderFullscreenPanel() string {
	panelWidth := d.width - 4
	panelHeight := d.height - 8

	var content string
	switch d.focus {
	case FocusLogs:
		d.logs.SetSize(panelWidth, panelHeight)
		content = d.logs.View()
	case FocusEvents:
		d.events.SetSize(panelWidth, panelHeight)
		content = d.events.View()
	case FocusMetrics:
		d.metrics.SetSize(panelWidth, panelHeight)
		content = d.metrics.View()
	case FocusManifest:
		d.manifest.SetSize(panelWidth, panelHeight)
		content = d.manifest.View()
	}

	return d.wrapPanel(content, panelWidth, panelHeight, true)
}

func (d Dashboard) renderTopRow() string {
	halfWidth := d.width / 2
	panelHeight := (d.height - 6) / 2

	d.logs.SetSize(halfWidth-2, panelHeight)
	d.events.SetSize(halfWidth-2, panelHeight)

	logsView := d.wrapPanel(d.logs.View(), halfWidth-2, panelHeight, d.focus == FocusLogs)
	eventsView := d.wrapPanel(d.events.View(), halfWidth-2, panelHeight, d.focus == FocusEvents)

	return lipgloss.JoinHorizontal(lipgloss.Top, logsView, eventsView)
}

func (d Dashboard) renderBottomRow() string {
	halfWidth := d.width / 2
	panelHeight := (d.height - 6) / 2

	d.metrics.SetSize(halfWidth-2, panelHeight)
	d.manifest.SetSize(halfWidth-2, panelHeight)

	metricsView := d.wrapPanel(d.metrics.View(), halfWidth-2, panelHeight, d.focus == FocusMetrics)
	manifestView := d.wrapPanel(d.manifest.View(), halfWidth-2, panelHeight, d.focus == FocusManifest)

	return lipgloss.JoinHorizontal(lipgloss.Top, metricsView, manifestView)
}

func (d Dashboard) wrapPanel(content string, width, height int, active bool) string {
	style := styles.PanelStyle
	if active {
		style = styles.ActivePanelStyle
	}

	return style.
		Width(width).
		Height(height).
		Render(content)
}

func (d Dashboard) renderFloatingHelp(base string) string {
	// Get the help content
	helpContent := d.help.View()

	// Use lipgloss.Place to center the help modal over the content
	return lipgloss.Place(
		d.width,
		d.height-4,
		lipgloss.Center,
		lipgloss.Center,
		helpContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Background),
	)
}

func (d Dashboard) renderFloatingMenu(base string) string {
	menuContent := d.actionMenu.View()

	return lipgloss.Place(
		d.width,
		d.height-4,
		lipgloss.Center,
		lipgloss.Center,
		menuContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Background),
	)
}

func (d Dashboard) renderFloatingDialog(dialogContent string) string {
	return lipgloss.Place(
		d.width,
		d.height-4,
		lipgloss.Center,
		lipgloss.Center,
		dialogContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(styles.Background),
	)
}

func (d *Dashboard) SetPod(pod *k8s.PodInfo) {
	d.pod = pod
	d.manifest.SetPod(pod)
	d.metrics.SetPod(pod)

	// Extract container names for logs panel
	var containerNames []string
	for _, c := range pod.Containers {
		containerNames = append(containerNames, c.Name)
	}
	d.logs.SetContainers(containerNames)
}

func (d *Dashboard) SetLogs(logs []k8s.LogLine) {
	d.logs.SetLogs(logs)
}

func (d *Dashboard) SetEvents(events []k8s.EventInfo) {
	d.events.SetEvents(events)
}

func (d *Dashboard) SetMetrics(metrics *k8s.PodMetrics) {
	d.metrics.SetMetrics(metrics)
}

func (d *Dashboard) SetRelated(related *k8s.RelatedResources) {
	d.manifest.SetRelated(related)
}

func (d *Dashboard) SetHelpers(helpers []k8s.DebugHelper) {
	d.manifest.SetHelpers(helpers)
}

func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.statusBar.SetWidth(width)
	d.breadcrumb.SetWidth(width)
	d.help.SetSize(width, height)
}

func (d *Dashboard) SetBreadcrumb(items ...string) {
	d.breadcrumb.SetItems(items...)
}

func (d *Dashboard) SetContext(ctx string) {
	d.context = ctx
	d.statusBar.SetContext(ctx)
}

func (d *Dashboard) SetNamespace(ns string) {
	d.namespace = ns
	d.statusBar.SetNamespace(ns)
}

func (d *Dashboard) SetResource(res string) {
	d.statusBar.SetResource(res)
}

func (d Dashboard) Focus() PanelFocus {
	return d.focus
}

func (d Dashboard) HelpVisible() bool {
	return d.help.IsVisible()
}

func (d Dashboard) ShortHelp() string {
	return d.help.ShortHelp()
}

// Logs panel state getters for app to react to
func (d Dashboard) LogsSelectedContainer() string {
	return d.logs.SelectedContainer()
}

func (d Dashboard) LogsShowPrevious() bool {
	return d.logs.ShowPrevious()
}

func (d *Dashboard) GetPod() *k8s.PodInfo {
	return d.pod
}

func (d Dashboard) IsLogsSearching() bool {
	return d.logs.IsSearching()
}

func (d Dashboard) HasActiveOverlay() bool {
	return d.resultViewer.IsVisible() ||
		d.confirmDialog.IsVisible() ||
		d.podActionMenu.IsVisible() ||
		d.actionMenu.IsVisible() ||
		d.help.IsVisible()
}
