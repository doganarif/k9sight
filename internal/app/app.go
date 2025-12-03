package app

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/doganarif/k9sight/internal/config"
	"github.com/doganarif/k9sight/internal/k8s"
	"github.com/doganarif/k9sight/internal/ui/components"
	"github.com/doganarif/k9sight/internal/ui/keys"
	"github.com/doganarif/k9sight/internal/ui/styles"
	"github.com/doganarif/k9sight/internal/ui/views"
)

type ViewState int

const (
	ViewNavigator ViewState = iota
	ViewDashboard
)

type Model struct {
	k8sClient          *k8s.Client
	config             *config.Config
	navigator          components.Navigator
	dashboard          views.Dashboard
	statusBar          components.StatusBar
	help               components.HelpPanel
	spinner            spinner.Model
	workloadActionMenu components.WorkloadActionMenu
	confirmDialog      components.ConfirmDialog
	view               ViewState
	width              int
	height             int
	loading            bool
	err                error
	keys               keys.KeyMap
	workload           *k8s.WorkloadInfo
	pod                *k8s.PodInfo
	statusMsg          string // Status message for navigator view

	// State tracking for reactive log fetching
	lastShowPrevious bool
	lastLogContainer string
}

type loadedMsg struct {
	workloads  []k8s.WorkloadInfo
	namespaces []string
	err        error
}

type podsLoadedMsg struct {
	pods []k8s.PodInfo
	err  error
}

type dashboardDataMsg struct {
	logs    []k8s.LogLine
	events  []k8s.EventInfo
	metrics *k8s.PodMetrics
	related *k8s.RelatedResources
	helpers []k8s.DebugHelper
}

type logsUpdatedMsg struct {
	logs []k8s.LogLine
}

type podDeletedMsg struct {
	namespace string
	podName   string
	err       error
}

type workloadActionMsg struct {
	action       string
	workloadName string
	namespace    string
	resourceType k8s.ResourceType
	replicas     int32
	err          error
}

type tickMsg time.Time

func New() (*Model, error) {
	client, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	client.SetNamespace(cfg.LastNamespace)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return &Model{
		k8sClient:          client,
		config:             cfg,
		navigator:          components.NewNavigator(),
		dashboard:          views.NewDashboard(),
		statusBar:          components.NewStatusBar(),
		help:               components.NewHelpPanel(),
		spinner:            s,
		workloadActionMenu: components.NewWorkloadActionMenu(),
		confirmDialog:      components.NewConfirmDialog(),
		view:               ViewNavigator,
		loading:            true,
		keys:      keys.DefaultKeyMap(),
	}, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadInitialData(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.navigator.SetSize(msg.Width, msg.Height-2)
		m.dashboard.SetSize(msg.Width, msg.Height-2)
		m.statusBar.SetWidth(msg.Width)
		m.help.SetSize(msg.Width, msg.Height)
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.navigator.SetWorkloads(msg.workloads)
		m.navigator.SetNamespaces(msg.namespaces)
		return m, nil

	case podsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.navigator.SetPods(msg.pods)
		m.navigator.SetMode(components.ModePods)
		return m, nil

	case dashboardDataMsg:
		m.loading = false
		m.dashboard.SetLogs(msg.logs)
		m.dashboard.SetEvents(msg.events)
		m.dashboard.SetMetrics(msg.metrics)
		m.dashboard.SetRelated(msg.related)
		m.dashboard.SetHelpers(msg.helpers)
		return m, nil

	case logsUpdatedMsg:
		m.dashboard.SetLogs(msg.logs)
		return m, nil

	case views.DeletePodRequest:
		return m, m.deletePod(msg.Namespace, msg.PodName)

	case podDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Go back to navigator after deletion
			m.view = ViewNavigator
			m.pod = nil
			if m.workload != nil {
				return m, m.loadPods(m.workload)
			}
			return m, m.loadWorkloads()
		}
		return m, nil

	case components.WorkloadActionMenuResult:
		workload := m.navigator.SelectedWorkload()
		if workload == nil {
			return m, nil
		}
		switch msg.Item.Action {
		case "scale":
			m.loading = true
			return m, m.scaleWorkload(workload, msg.Item.Replicas)
		case "copy":
			err := components.CopyToClipboard(msg.Item.Command)
			if err == nil {
				m.statusMsg = "Copied: " + msg.Item.Label
			} else {
				m.statusMsg = "Copy failed: " + err.Error()
			}
		}
		return m, nil

	case components.ConfirmResult:
		// Handle workload restart at app level
		if msg.Confirmed && msg.Action == "restart" {
			if workload, ok := msg.Data.(*k8s.WorkloadInfo); ok {
				m.loading = true
				m.statusMsg = "Restarting..."
				return m, m.restartWorkload(workload)
			}
		}
		// Forward other confirm results (exec, port-forward, delete) to dashboard
		if m.view == ViewDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case views.ExecFinishedMsg:
		// Forward exec finished to dashboard
		if m.view == ViewDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case views.DescribeOutputMsg:
		// Forward describe output to dashboard
		if m.view == ViewDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case workloadActionMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = "Error: " + msg.err.Error()
		} else {
			switch msg.action {
			case "scale":
				m.statusMsg = fmt.Sprintf("Scaled %s to %d replicas", msg.workloadName, msg.replicas)
			case "restart":
				m.statusMsg = fmt.Sprintf("Restart initiated for %s", msg.workloadName)
			}
			// Refresh workloads list
			return m, m.loadWorkloads()
		}
		return m, nil

	case tickMsg:
		if m.view == ViewDashboard && m.pod != nil {
			return m, tea.Batch(
				m.loadDashboardData(m.pod),
				m.tickCmd(),
			)
		}
		return m, m.tickCmd()

	case tea.KeyMsg:
		// Confirm dialog takes highest priority
		if m.confirmDialog.IsVisible() {
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		// Workload action menu takes priority
		if m.workloadActionMenu.IsVisible() {
			m.workloadActionMenu, cmd = m.workloadActionMenu.Update(msg)
			return m, cmd
		}

		// Help overlay takes priority
		if m.help.IsVisible() {
			if msg.String() == "?" || msg.String() == "esc" {
				m.help.Hide()
				return m, nil
			}
			return m, nil
		}

		// Clear status message on key press in navigator
		if m.view == ViewNavigator {
			m.statusMsg = ""
		}

		// When navigator is searching, only handle esc/enter at app level
		// All other keys go to the search input
		if m.view == ViewNavigator && m.navigator.IsSearching() {
			switch msg.String() {
			case "esc":
				m.navigator.CloseSearch()
				return m, nil
			case "enter":
				m.navigator.CloseSearch()
				return m, nil
			case "ctrl+c":
				m.saveConfig()
				return m, tea.Quit
			default:
				// Pass all other keys to navigator for search input
				m.navigator, cmd = m.navigator.Update(msg)
				return m, cmd
			}
		}

		// Normal key handling when not searching
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.saveConfig()
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.help.Toggle()
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m, m.refresh()

		case key.Matches(msg, m.keys.Namespace):
			if m.view == ViewNavigator {
				m.navigator.SetMode(components.ModeNamespace)
				return m, nil
			}

		case key.Matches(msg, m.keys.Back):
			// Don't handle back if dashboard has active overlay or is searching - let dashboard handle esc
			if m.view == ViewDashboard && (m.dashboard.IsLogsSearching() || m.dashboard.HasActiveOverlay()) {
				break // Fall through to dashboard update
			}
			return m.handleBack()

		case key.Matches(msg, m.keys.Enter):
			// Don't handle enter if dashboard has active overlay - let dashboard handle it
			if m.view == ViewDashboard && m.dashboard.HasActiveOverlay() {
				break // Fall through to dashboard update
			}
			return m.handleEnter()
		}
	}

	switch m.view {
	case ViewNavigator:
		if !m.navigator.IsSearching() {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				if key.Matches(msg, m.keys.ResourceType) {
					m.navigator.SetMode(components.ModeResourceType)
					return m, nil
				}
				// Scale action (only for scalable resource types)
				if key.Matches(msg, m.keys.Scale) && m.navigator.Mode() == components.ModeWorkloads {
					workload := m.navigator.SelectedWorkload()
					if workload != nil {
						rt := m.navigator.ResourceType()
						if rt == k8s.ResourceDeployments || rt == k8s.ResourceStatefulSets {
							items := components.ScaleActions(
								m.k8sClient.Namespace(),
								workload.Name,
								string(rt),
								workload.Replicas,
							)
							m.workloadActionMenu.Show("Scale "+workload.Name, items)
							return m, nil
						}
					}
				}
				// Restart action
				if key.Matches(msg, m.keys.Restart) && m.navigator.Mode() == components.ModeWorkloads {
					workload := m.navigator.SelectedWorkload()
					if workload != nil {
						rt := m.navigator.ResourceType()
						if rt == k8s.ResourceDeployments || rt == k8s.ResourceStatefulSets || rt == k8s.ResourceDaemonSets {
							m.confirmDialog.Show(
								"Restart "+string(rt),
								"Are you sure you want to restart '"+workload.Name+"'?",
								"restart",
								workload,
							)
							return m, nil
						}
					}
				}
			}
		}
		m.navigator, cmd = m.navigator.Update(msg)
		cmds = append(cmds, cmd)

	case ViewDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
		cmds = append(cmds, cmd)

		// Check if log state changed and needs refresh
		if m.pod != nil {
			currentShowPrevious := m.dashboard.LogsShowPrevious()
			currentContainer := m.dashboard.LogsSelectedContainer()

			if currentShowPrevious != m.lastShowPrevious || currentContainer != m.lastLogContainer {
				m.lastShowPrevious = currentShowPrevious
				m.lastLogContainer = currentContainer
				cmds = append(cmds, m.loadLogsForState(m.pod, currentContainer, currentShowPrevious))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.err != nil {
		return styles.StatusError.Render("Error: " + m.err.Error())
	}

	if m.loading {
		// Center loading spinner
		loadingMsg := m.spinner.View() + " Loading..."
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loadingMsg)
	}

	// Build footer with optional status message
	m.statusBar.SetContext(m.k8sClient.Context())
	m.statusBar.SetNamespace(m.k8sClient.Namespace())
	m.statusBar.SetResource(string(m.navigator.ResourceType()))
	footerLine := m.statusBar.View()
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
		footerLine = footerLine + "  " + statusStyle.Render(m.statusMsg)
	}
	footer := footerLine + "\n" + m.help.ShortHelp() + "  " + styles.Credit()
	footerHeight := 2

	// Calculate content height (full height minus footer)
	contentHeight := m.height - footerHeight - 1

	var content string
	switch m.view {
	case ViewNavigator:
		content = m.navigator.View()
	case ViewDashboard:
		content = m.dashboard.View()
	}

	// Render confirm dialog as overlay (highest priority)
	if m.confirmDialog.IsVisible() {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.confirmDialog.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(styles.Background),
		)
	}

	// Render workload action menu as overlay
	if m.workloadActionMenu.IsVisible() {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			m.workloadActionMenu.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(styles.Background),
		)
	}

	if m.help.IsVisible() {
		// Render floating help modal centered on screen
		helpModal := m.help.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			helpModal,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(styles.Background),
		)
	}

	// Create full height layout with content at top and footer at bottom
	contentStyle := lipgloss.NewStyle().Height(contentHeight)
	mainContent := contentStyle.Render(content)

	return mainContent + "\n" + footer
}

func (m *Model) handleBack() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewDashboard:
		m.view = ViewNavigator
		m.pod = nil
		if m.workload != nil {
			m.navigator.SetMode(components.ModePods)
		} else {
			m.navigator.SetMode(components.ModeWorkloads)
		}
		return m, nil

	case ViewNavigator:
		switch m.navigator.Mode() {
		case components.ModePods:
			m.navigator.SetMode(components.ModeWorkloads)
			m.workload = nil
			return m, m.loadWorkloads()
		case components.ModeNamespace:
			m.navigator.SetMode(components.ModeWorkloads)
			return m, nil
		case components.ModeResourceType:
			m.navigator.SetMode(components.ModeWorkloads)
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewNavigator:
		switch m.navigator.Mode() {
		case components.ModeWorkloads:
			workload := m.navigator.SelectedWorkload()
			if workload != nil {
				m.workload = workload
				m.loading = true
				return m, m.loadPods(workload)
			}

		case components.ModePods:
			pod := m.navigator.SelectedPod()
			if pod != nil {
				m.pod = pod
				m.view = ViewDashboard
				m.dashboard.SetPod(pod)
				m.dashboard.SetBreadcrumb(
					m.k8sClient.Namespace(),
					string(m.navigator.ResourceType()),
					m.workload.Name,
					pod.Name,
				)
				m.dashboard.SetContext(m.k8sClient.Context())
				m.dashboard.SetNamespace(m.k8sClient.Namespace())
				m.loading = true
				return m, tea.Batch(
					m.loadDashboardData(pod),
					m.tickCmd(),
				)
			}

		case components.ModeNamespace:
			ns := m.navigator.SelectedNamespace()
			if ns != "" {
				m.k8sClient.SetNamespace(ns)
				m.config.SetLastNamespace(ns)
				m.navigator.SetMode(components.ModeWorkloads)
				m.loading = true
				return m, m.loadWorkloads()
			}

		case components.ModeResourceType:
			rt := m.navigator.SelectedResourceType()
			m.navigator.SetResourceType(rt)
			m.config.SetLastResourceType(string(rt))
			m.navigator.SetMode(components.ModeWorkloads)
			m.loading = true
			return m, m.loadWorkloads()
		}
	}
	return m, nil
}

func (m *Model) refresh() tea.Cmd {
	switch m.view {
	case ViewNavigator:
		m.loading = true
		return m.loadWorkloads()
	case ViewDashboard:
		if m.pod != nil {
			m.loading = true
			return m.loadDashboardData(m.pod)
		}
	}
	return nil
}

func (m *Model) loadInitialData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		namespaces, err := m.k8sClient.ListNamespaces(ctx)
		if err != nil {
			return loadedMsg{err: err}
		}

		rt := k8s.ResourceType(m.config.LastResourceType)
		if rt == "" {
			rt = k8s.ResourceDeployments
		}
		m.navigator.SetResourceType(rt)

		workloads, err := k8s.ListWorkloads(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace(), rt)
		if err != nil {
			return loadedMsg{err: err}
		}

		return loadedMsg{
			workloads:  workloads,
			namespaces: namespaces,
		}
	}
}

func (m *Model) loadWorkloads() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		workloads, err := k8s.ListWorkloads(ctx, m.k8sClient.Clientset(), m.k8sClient.Namespace(), m.navigator.ResourceType())
		if err != nil {
			return loadedMsg{err: err}
		}

		namespaces, _ := m.k8sClient.ListNamespaces(ctx)

		return loadedMsg{
			workloads:  workloads,
			namespaces: namespaces,
		}
	}
}

func (m *Model) loadPods(workload *k8s.WorkloadInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		pods, err := k8s.GetWorkloadPods(ctx, m.k8sClient.Clientset(), *workload)
		if err != nil {
			return podsLoadedMsg{err: err}
		}
		return podsLoadedMsg{pods: pods}
	}
}

func (m *Model) loadDashboardData(pod *k8s.PodInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		logs, _ := k8s.GetAllContainerLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, 200)
		events, _ := k8s.GetPodEvents(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name)
		metrics, _ := k8s.GetPodMetrics(ctx, m.k8sClient.MetricsClient(), pod.Namespace, pod.Name)
		related, _ := k8s.GetRelatedResources(ctx, m.k8sClient.Clientset(), *pod)

		helpers := k8s.AnalyzePodIssues(pod, events)

		return dashboardDataMsg{
			logs:    logs,
			events:  events,
			metrics: metrics,
			related: related,
			helpers: helpers,
		}
	}
}

func (m *Model) loadLogsForState(pod *k8s.PodInfo, container string, previous bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var logs []k8s.LogLine
		var err error

		if previous {
			// Get previous logs for specific container or first container
			targetContainer := container
			if targetContainer == "" && len(pod.Containers) > 0 {
				targetContainer = pod.Containers[0].Name
			}
			if targetContainer != "" {
				logs, err = k8s.GetPreviousLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, targetContainer, 200)
			}
		} else if container != "" {
			// Get logs for specific container
			opts := k8s.LogOptions{
				Container:  container,
				TailLines:  200,
				Timestamps: true,
			}
			logs, err = k8s.GetPodLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, opts)
		} else {
			// Get all container logs
			logs, err = k8s.GetAllContainerLogs(ctx, m.k8sClient.Clientset(), pod.Namespace, pod.Name, 200)
		}

		if err != nil {
			return logsUpdatedMsg{logs: []k8s.LogLine{{Content: "Error fetching logs: " + err.Error(), IsError: true}}}
		}

		return logsUpdatedMsg{logs: logs}
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(m.config.RefreshInterval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) saveConfig() {
	_ = m.config.Save()
}

func (m *Model) deletePod(namespace, podName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.DeletePod(ctx, namespace, podName)
		return podDeletedMsg{
			namespace: namespace,
			podName:   podName,
			err:       err,
		}
	}
}

func (m *Model) scaleWorkload(workload *k8s.WorkloadInfo, replicas int32) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.ScaleWorkload(ctx, workload.Namespace, workload.Name, workload.Type, replicas)
		return workloadActionMsg{
			action:       "scale",
			workloadName: workload.Name,
			namespace:    workload.Namespace,
			resourceType: workload.Type,
			replicas:     replicas,
			err:          err,
		}
	}
}

func (m *Model) restartWorkload(workload *k8s.WorkloadInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.RestartWorkload(ctx, workload.Namespace, workload.Name, workload.Type)
		return workloadActionMsg{
			action:       "restart",
			workloadName: workload.Name,
			namespace:    workload.Namespace,
			resourceType: workload.Type,
			err:          err,
		}
	}
}
