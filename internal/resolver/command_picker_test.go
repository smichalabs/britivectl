package resolver

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func sampleCommandChoices() []CommandChoice {
	return []CommandChoice{
		{Name: "checkout", Short: "Check out a Britive profile"},
		{Name: "status", Short: "Show active checkouts"},
		{Name: "login", Short: "Authenticate with Britive"},
	}
}

func TestPickCommand_NoCommands(t *testing.T) {
	_, err := PickCommand(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty commands slice, got nil")
	}
}

func TestCommandItem_Title(t *testing.T) {
	item := commandItem{cmd: CommandChoice{Name: "checkout"}}
	if got := item.Title(); got != "checkout" {
		t.Errorf("Title() = %q, want checkout", got)
	}
}

func TestCommandItem_Description(t *testing.T) {
	item := commandItem{cmd: CommandChoice{Short: "Check out a profile"}}
	if got := item.Description(); got != "Check out a profile" {
		t.Errorf("Description() = %q, want short text", got)
	}
}

func TestCommandItem_FilterValue(t *testing.T) {
	item := commandItem{cmd: CommandChoice{Name: "checkout", Short: "Check out a profile"}}
	got := item.FilterValue()
	if !strings.Contains(got, "checkout") || !strings.Contains(got, "profile") {
		t.Errorf("FilterValue() = %q, want both name and short", got)
	}
}

func TestCommandModel_Init_SendsSlashKey(t *testing.T) {
	m := commandModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		t.Fatalf("Init() command returned %T, want tea.KeyMsg", msg)
	}
	if len(key.Runes) == 0 || key.Runes[0] != '/' {
		t.Errorf("Init() keypress = %v, want '/'", key.Runes)
	}
}

func TestCommandModel_Update_WindowSize(t *testing.T) {
	m := commandModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if _, ok := updated.(commandModel); !ok {
		t.Errorf("Update() returned wrong model type: %T", updated)
	}
}

func TestCommandModel_Update_QuitKeys(t *testing.T) {
	m := commandModel{list: list.New(nil, list.NewDefaultDelegate(), 0, 0)}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	final := updated.(commandModel)
	if !final.canceled {
		t.Error("ctrl+c should set canceled = true")
	}
	if cmd == nil {
		t.Error("ctrl+c should return tea.Quit cmd")
	}
}

func TestCommandModel_Update_EnterSelects(t *testing.T) {
	items := []list.Item{commandItem{cmd: CommandChoice{Name: "checkout"}}}
	l := list.New(items, list.NewDefaultDelegate(), 100, 40)
	m := commandModel{list: l}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(commandModel)
	if final.chosen != "checkout" {
		t.Errorf("chosen = %q, want checkout", final.chosen)
	}
}

func TestCommandModel_View(t *testing.T) {
	m := commandModel{list: list.New(nil, list.NewDefaultDelegate(), 80, 20)}
	if v := m.View(); v == "" {
		t.Error("View() returned empty string")
	}
}
