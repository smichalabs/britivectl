package resolver

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandChoice describes a top-level subcommand the user can pick from
// the interactive command picker. It is kept simple (just a name and a
// short description) so cmd/ can populate it without pulling in cobra
// types here.
type CommandChoice struct {
	Name  string // e.g. "checkout"
	Short string // e.g. "Check out a Britive profile"
}

// PickCommand shows an fzf-style TUI of the given commands and returns the
// name chosen by the user. The first entry is highlighted as the default so
// pressing enter without filtering runs it immediately.
//
// Returns ErrCanceled if the user aborts (esc or ctrl+c).
// Falls back to returning the first command on non-TTY stdin so scripts and
// tests that pipe input still get a deterministic choice.
func PickCommand(ctx context.Context, commands []CommandChoice) (string, error) {
	if len(commands) == 0 {
		return "", fmt.Errorf("no commands available")
	}
	if !isTTY(os.Stdin) {
		return commands[0].Name, nil
	}

	items := make([]list.Item, len(commands))
	for i, c := range commands {
		items[i] = commandItem{cmd: c}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#7C3AED")).
		BorderLeftForeground(lipgloss.Color("#7C3AED"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#A78BFA")).
		BorderLeftForeground(lipgloss.Color("#7C3AED"))

	l := list.New(items, delegate, 0, 0)
	l.Title = "bctl -- pick a command (type to filter, enter to run, esc to cancel)"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	m := commandModel{list: l}
	prog := tea.NewProgram(m, tea.WithContext(ctx), tea.WithAltScreen())
	res, err := prog.Run()
	if err != nil {
		return "", fmt.Errorf("running command picker: %w", err)
	}
	final := res.(commandModel)
	if final.canceled || final.chosen == "" {
		return "", ErrCanceled
	}
	return final.chosen, nil
}

// commandItem adapts a CommandChoice into bubbles/list.Item.
type commandItem struct{ cmd CommandChoice }

func (c commandItem) Title() string       { return c.cmd.Name }
func (c commandItem) Description() string { return c.cmd.Short }
func (c commandItem) FilterValue() string { return c.cmd.Name + " " + c.cmd.Short }

// commandModel is the bubbletea Model for the command picker. It is
// structurally identical to tuiModel but stores a string result because
// the caller only needs the chosen command name.
type commandModel struct {
	list     list.Model
	chosen   string
	canceled bool
}

func (m commandModel) Init() tea.Cmd {
	// Start in filter-input mode so the user can type immediately.
	return func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	}
}

func (m commandModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(commandItem); ok {
				m.chosen = item.cmd.Name
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m commandModel) View() string {
	return m.list.View()
}
