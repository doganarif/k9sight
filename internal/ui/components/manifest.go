package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/doganarif/k9sight/internal/k8s"
	"github.com/doganarif/k9sight/internal/ui/styles"
)

type ManifestViewMode int

const (
	ManifestViewSummary ManifestViewMode = iota
	ManifestViewDetails
	ManifestViewResources
)

var manifestViewModeLabels = map[ManifestViewMode]string{
	ManifestViewSummary:   "Summary",
	ManifestViewDetails:   "Details",
	ManifestViewResources: "Resources",
}

type ManifestPanel struct {
	pod       *k8s.PodInfo
	related   *k8s.RelatedResources
	helpers   []k8s.DebugHelper
	viewport  viewport.Model
	ready     bool
	width     int
	height    int
	viewMode  ManifestViewMode
}

func NewManifestPanel() ManifestPanel {
	return ManifestPanel{}
}

func (m ManifestPanel) Init() tea.Cmd {
	return nil
}

func (m ManifestPanel) Update(msg tea.Msg) (ManifestPanel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "d":
			m.viewMode = (m.viewMode + 1) % 3
			m.updateContent()
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ManifestPanel) View() string {
	if !m.ready {
		return styles.PanelStyle.Render("Loading manifest...")
	}

	var header strings.Builder
	header.WriteString(styles.PanelTitleStyle.Render("Pod Details"))
	header.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf(" [%s]", manifestViewModeLabels[m.viewMode])))
	header.WriteString(styles.HelpDescStyle.Render(" (d:cycle)"))
	header.WriteString("\n")

	return header.String() + m.viewport.View()
}

func (m *ManifestPanel) SetPod(pod *k8s.PodInfo) {
	m.pod = pod
	m.updateContent()
}

func (m *ManifestPanel) SetRelated(related *k8s.RelatedResources) {
	m.related = related
	m.updateContent()
}

func (m *ManifestPanel) SetHelpers(helpers []k8s.DebugHelper) {
	m.helpers = helpers
	m.updateContent()
}

func (m *ManifestPanel) SetSize(width, height int) {
	m.width = width
	m.height = height - 2

	if !m.ready {
		m.viewport = viewport.New(width, m.height)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = m.height
	}

	m.updateContent()
}

func (m *ManifestPanel) updateContent() {
	if !m.ready || m.pod == nil {
		return
	}

	var content strings.Builder

	switch m.viewMode {
	case ManifestViewSummary:
		// Summary: Basic pod info and debug hints
		content.WriteString(m.renderPodInfo())
		if len(m.helpers) > 0 {
			content.WriteString("\n")
			content.WriteString(m.renderHelpers())
		}

	case ManifestViewDetails:
		// Details: Pod info, containers, labels, conditions
		content.WriteString(m.renderPodInfo())
		content.WriteString("\n")
		content.WriteString(m.renderContainers())
		content.WriteString("\n")
		content.WriteString(m.renderLabels())
		content.WriteString("\n")
		content.WriteString(m.renderConditions())

	case ManifestViewResources:
		// Resources: Container resources and related resources
		content.WriteString(m.renderContainerResources())
		if m.related != nil {
			content.WriteString("\n")
			content.WriteString(m.renderRelated())
		}
	}

	m.viewport.SetContent(content.String())
}

func (m ManifestPanel) renderPodInfo() string {
	var b strings.Builder

	b.WriteString(styles.SubtitleStyle.Render("Pod Info\n"))
	b.WriteString(fmt.Sprintf("  Name:      %s\n", m.pod.Name))
	b.WriteString(fmt.Sprintf("  Namespace: %s\n", m.pod.Namespace))
	b.WriteString(fmt.Sprintf("  Node:      %s\n", m.pod.Node))
	b.WriteString(fmt.Sprintf("  IP:        %s\n", m.pod.IP))

	statusStyle := styles.GetStatusStyle(m.pod.Status)
	b.WriteString(fmt.Sprintf("  Status:    %s\n", statusStyle.Render(m.pod.Status)))
	b.WriteString(fmt.Sprintf("  Ready:     %s\n", m.pod.Ready))
	b.WriteString(fmt.Sprintf("  Restarts:  %d\n", m.pod.Restarts))
	b.WriteString(fmt.Sprintf("  Age:       %s\n", m.pod.Age))

	if m.pod.OwnerRef != "" {
		b.WriteString(fmt.Sprintf("  Owner:     %s/%s\n", m.pod.OwnerKind, m.pod.OwnerRef))
	}

	return b.String()
}

func (m ManifestPanel) renderHelpers() string {
	var b strings.Builder

	b.WriteString(styles.EventWarning.Render("Debug Hints\n"))
	for _, helper := range m.helpers {
		severity := styles.StatusMuted
		switch helper.Severity {
		case "High":
			severity = styles.StatusError
		case "Warning":
			severity = styles.EventWarning
		}

		b.WriteString(severity.Render(fmt.Sprintf("  [%s] %s\n", helper.Severity, helper.Issue)))
		for _, suggestion := range helper.Suggestions {
			b.WriteString(styles.SubtitleStyle.Render(fmt.Sprintf("    • %s\n", suggestion)))
		}
	}

	return b.String()
}

func (m ManifestPanel) renderContainers() string {
	var b strings.Builder

	b.WriteString(styles.SubtitleStyle.Render("Containers\n"))
	for _, c := range m.pod.Containers {
		stateStyle := styles.GetStatusStyle(c.State)

		b.WriteString(styles.LogContainer.Render(fmt.Sprintf("  %s\n", c.Name)))
		b.WriteString(fmt.Sprintf("    Image:    %s\n", styles.Truncate(c.Image, m.width-14)))
		b.WriteString(fmt.Sprintf("    State:    %s", stateStyle.Render(c.State)))
		if c.Reason != "" {
			b.WriteString(fmt.Sprintf(" (%s)", c.Reason))
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    Ready:    %v\n", c.Ready))
		b.WriteString(fmt.Sprintf("    Restarts: %d\n", c.RestartCount))

		if len(c.Ports) > 0 {
			ports := make([]string, len(c.Ports))
			for i, p := range c.Ports {
				ports[i] = fmt.Sprintf("%d", p)
			}
			b.WriteString(fmt.Sprintf("    Ports:    %s\n", strings.Join(ports, ", ")))
		}
	}

	return b.String()
}

func (m ManifestPanel) renderRelated() string {
	var b strings.Builder

	b.WriteString(styles.SubtitleStyle.Render("Related Resources\n"))

	if len(m.related.Services) > 0 {
		b.WriteString("  Services:\n")
		for _, svc := range m.related.Services {
			b.WriteString(fmt.Sprintf("    • %s (%s) - %s [%d endpoints]\n",
				svc.Name, svc.Type, svc.Ports, svc.Endpoints))
		}
	}

	if len(m.related.Ingresses) > 0 {
		b.WriteString("  Ingresses:\n")
		for _, ing := range m.related.Ingresses {
			b.WriteString(fmt.Sprintf("    • %s - %s%s\n", ing.Name, ing.Hosts, ing.Paths))
		}
	}

	if len(m.related.ConfigMaps) > 0 {
		b.WriteString(fmt.Sprintf("  ConfigMaps: %s\n", strings.Join(m.related.ConfigMaps, ", ")))
	}

	if len(m.related.Secrets) > 0 {
		b.WriteString(fmt.Sprintf("  Secrets: %s\n", strings.Join(m.related.Secrets, ", ")))
	}

	return b.String()
}

func (m ManifestPanel) renderLabels() string {
	var b strings.Builder

	b.WriteString(styles.SubtitleStyle.Render("Labels\n"))
	if len(m.pod.Labels) == 0 {
		b.WriteString("  <none>\n")
	} else {
		for k, v := range m.pod.Labels {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	return b.String()
}

func (m ManifestPanel) renderConditions() string {
	var b strings.Builder

	b.WriteString(styles.SubtitleStyle.Render("Conditions\n"))
	for _, cond := range m.pod.Conditions {
		status := styles.StatusRunning
		if cond.Status != "True" {
			status = styles.StatusError
		}
		b.WriteString(fmt.Sprintf("  %s: %s\n",
			cond.Type,
			status.Render(string(cond.Status))))
	}

	return b.String()
}

func (m ManifestPanel) renderContainerResources() string {
	var b strings.Builder

	b.WriteString(styles.SubtitleStyle.Render("Container Resources\n"))
	for _, c := range m.pod.Containers {
		b.WriteString(styles.LogContainer.Render(fmt.Sprintf("  %s\n", c.Name)))

		// CPU
		b.WriteString("    CPU:\n")
		b.WriteString(fmt.Sprintf("      Request: %s\n", c.Resources.CPURequest))
		b.WriteString(fmt.Sprintf("      Limit:   %s\n", c.Resources.CPULimit))

		// Memory
		b.WriteString("    Memory:\n")
		b.WriteString(fmt.Sprintf("      Request: %s\n", c.Resources.MemoryRequest))
		b.WriteString(fmt.Sprintf("      Limit:   %s\n", c.Resources.MemoryLimit))

		// Ports
		if len(c.Ports) > 0 {
			ports := make([]string, len(c.Ports))
			for i, p := range c.Ports {
				ports[i] = fmt.Sprintf("%d", p)
			}
			b.WriteString(fmt.Sprintf("    Ports: %s\n", strings.Join(ports, ", ")))
		}
		b.WriteString("\n")
	}

	return b.String()
}
