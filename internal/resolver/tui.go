package resolver

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// interactivePick renders a bubbletea list with live filtering so the user
// can type to narrow down matches. Enter selects the highlighted row. Esc
// or q quits.
//
// Returns ErrCanceled if the user aborts.
func interactivePick(ctx context.Context, matches []Match) (Match, error) {
	items := make([]list.Item, len(matches))
	for i, m := range matches {
		items[i] = profileItem{match: m}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#7C3AED")).
		BorderLeftForeground(lipgloss.Color("#7C3AED"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#A78BFA")).
		BorderLeftForeground(lipgloss.Color("#7C3AED"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "Pick a profile (type to filter, enter to select, esc to cancel)"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	m := tuiModel{list: l}
	// tea.WithAltScreen gives us a dedicated screen buffer so the picker
	// does not interleave with earlier command output.
	prog := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
	res, err := prog.Run()
	if err != nil {
		return Match{}, fmt.Errorf("running picker: %w", err)
	}
	final := res.(tuiModel)
	if final.canceled {
		return Match{}, ErrCanceled
	}
	if final.chosen == nil {
		return Match{}, ErrCanceled
	}
	return *final.chosen, nil
}

// profileItem adapts a Match into bubbles/list.Item so the list can render
// and filter it.
type profileItem struct{ match Match }

func (p profileItem) Title() string { return p.match.Alias }

func (p profileItem) Description() string {
	path := p.match.Profile.BritivePath
	if path == "" {
		path = "(no path)"
	}
	cloud := p.match.Profile.Cloud
	if cloud == "" {
		cloud = "?"
	}
	return fmt.Sprintf("[%s] %s", cloud, path)
}

// FilterValue is what bubbles/list runs the fuzzy match against. We include
// both alias and path so typing a substring of the Britive path matches even
// when the alias itself doesn't contain that substring.
func (p profileItem) FilterValue() string {
	return p.match.Alias + " " + p.match.Profile.BritivePath
}

// tuiModel is the bubbletea Model for the picker.
type tuiModel struct {
	list     list.Model
	chosen   *Match
	canceled bool
}

// Init sends a synthetic "/" keypress so the list starts in filter input
// mode, letting the user type immediately to filter without pressing "/"
// first.
func (m tuiModel) Init() tea.Cmd {
	return func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		// When the user is composing the filter, forward every key to the
		// list unchanged. This lets "q", "enter", etc. do the right thing
		// inside the filter input.
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		// Not filtering (filter has been applied or never entered).
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(profileItem); ok {
				mm := item.match
				m.chosen = &mm
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	return m.list.View()
}

// isTTY reports whether the given reader is a terminal. We only launch the
// bubbletea TUI when stdin is a real TTY so piped input (tests, CI, scripts)
// falls back to the numbered picker.
func isTTY(r io.Reader) bool {
	type fder interface {
		Fd() uintptr
	}
	f, ok := r.(fder)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
