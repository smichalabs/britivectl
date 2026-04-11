package resolver

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/smichalabs/britivectl/internal/config"
)

func sampleItem() profileItem {
	return profileItem{
		match: Match{
			Alias: "dev",
			Profile: config.Profile{
				BritivePath: "AWS/Sandbox/Developer",
				Cloud:       "aws",
			},
		},
	}
}

func TestProfileItem_Title(t *testing.T) {
	if got := sampleItem().Title(); got != "dev" {
		t.Errorf("Title() = %q, want dev", got)
	}
}

func TestProfileItem_Description(t *testing.T) {
	got := sampleItem().Description()
	if !strings.Contains(got, "aws") {
		t.Errorf("Description() = %q, want to contain 'aws'", got)
	}
	if !strings.Contains(got, "AWS/Sandbox/Developer") {
		t.Errorf("Description() = %q, want to contain path", got)
	}
}

func TestProfileItem_Description_MissingFields(t *testing.T) {
	p := profileItem{match: Match{Alias: "x"}}
	got := p.Description()
	if !strings.Contains(got, "no path") {
		t.Errorf("Description() = %q, want to contain '(no path)'", got)
	}
	if !strings.Contains(got, "?") {
		t.Errorf("Description() = %q, want '?' for unknown cloud", got)
	}
}

func TestProfileItem_FilterValue(t *testing.T) {
	got := sampleItem().FilterValue()
	if !strings.Contains(got, "dev") || !strings.Contains(got, "Sandbox") {
		t.Errorf("FilterValue() = %q, want alias + path", got)
	}
}

func TestTuiModel_Init_NoCmd(t *testing.T) {
	m := tuiModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned a command, want nil")
	}
}

func TestTuiModel_Update_WindowSize(t *testing.T) {
	m := tuiModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if _, ok := updated.(tuiModel); !ok {
		t.Errorf("Update() returned wrong model type: %T", updated)
	}
}

func TestTuiModel_Update_QuitKey(t *testing.T) {
	m := tuiModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	final := updated.(tuiModel)
	if !final.canceled {
		t.Error("ctrl+c should set canceled = true")
	}
	if cmd == nil {
		t.Error("ctrl+c should return tea.Quit cmd")
	}
}

func TestTuiModel_Update_EscKey(t *testing.T) {
	m := tuiModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	final := updated.(tuiModel)
	if !final.canceled {
		t.Error("esc should set canceled = true")
	}
}

func TestTuiModel_Update_EnterSelectsItem(t *testing.T) {
	items := []list.Item{sampleItem()}
	l := list.New(items, list.NewDefaultDelegate(), 100, 40)
	m := tuiModel{list: l}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(tuiModel)
	if final.chosen == nil {
		t.Fatal("enter should set chosen")
	}
	if final.chosen.Alias != "dev" {
		t.Errorf("chosen = %q, want dev", final.chosen.Alias)
	}
}

func TestTuiModel_View(t *testing.T) {
	m := tuiModel{list: list.New(nil, list.NewDefaultDelegate(), 80, 20)}
	if v := m.View(); v == "" {
		t.Error("View() returned empty string")
	}
}

func TestIsTTY_NotATerminal(t *testing.T) {
	// A bytes.Reader is never a TTY.
	if isTTY(bytes.NewReader(nil)) {
		t.Error("isTTY(bytes.Reader) = true, want false")
	}
}
