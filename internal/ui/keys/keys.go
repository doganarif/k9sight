package keys

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	// Navigation
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Home      key.Binding
	End       key.Binding
	PageUp    key.Binding
	PageDown  key.Binding

	// Actions
	Enter   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Search  key.Binding
	Clear   key.Binding

	// Panel navigation
	NextPanel key.Binding
	PrevPanel key.Binding
	Panel1    key.Binding
	Panel2    key.Binding
	Panel3    key.Binding
	Panel4    key.Binding

	// Mode switches
	Namespace    key.Binding
	ResourceType key.Binding

	// Log actions
	ToggleFollow key.Binding
	JumpToError  key.Binding
	ToggleWrap   key.Binding

	// Event actions
	ToggleAllEvents key.Binding

	// Manifest actions
	ToggleFullView key.Binding

	// Pod actions
	CopyCommands key.Binding
	PodActions   key.Binding

	// Workload actions
	Scale   key.Binding
	Restart key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Home: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "first"),
		),
		End: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "last"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn", "page down"),
		),

		// Actions
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Clear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear filter"),
		),

		// Panel navigation
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		PrevPanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-tab", "prev panel"),
		),
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "logs"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "events"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "metrics"),
		),
		Panel4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "manifest"),
		),

		// Mode switches
		Namespace: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "namespace"),
		),
		ResourceType: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "type"),
		),

		// Log actions
		ToggleFollow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow logs"),
		),
		JumpToError: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "next error"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "wrap lines"),
		),

		// Event actions
		ToggleAllEvents: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "all events"),
		),

		// Manifest actions
		ToggleFullView: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "full view"),
		),

		// Pod actions
		CopyCommands: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy kubectl"),
		),
		PodActions: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "pod actions"),
		),

		// Workload actions
		Scale: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scale"),
		),
		Restart: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "restart"),
		),
	}
}
