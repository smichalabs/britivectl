package resolver

import (
	"testing"

	"github.com/spf13/cobra"
)

// buildTestRoot wires a cobra tree that mirrors bctl's shape: a root with a
// runnable leaf, a parent with subcommands and no Run, and a hidden parent
// to exercise the filtering paths.
func buildTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "bctl"}

	// Runnable leaf.
	root.AddCommand(&cobra.Command{
		Use:   "checkout",
		Short: "checkout",
		RunE:  func(_ *cobra.Command, _ []string) error { return nil },
	})

	// Parent with two subcommands and no Run of its own. Mirrors `profiles`.
	parent := &cobra.Command{Use: "profiles", Short: "profiles"}
	parent.AddCommand(&cobra.Command{Use: "list", Short: "list", RunE: func(_ *cobra.Command, _ []string) error { return nil }})
	parent.AddCommand(&cobra.Command{Use: "sync", Short: "sync", RunE: func(_ *cobra.Command, _ []string) error { return nil }})
	root.AddCommand(parent)

	// Parent that includes hidden + auto-generated entries to exercise the
	// SubcommandChoices filter.
	noisy := &cobra.Command{Use: "noisy", Short: "noisy"}
	noisy.AddCommand(&cobra.Command{Use: "real", Short: "real", RunE: func(_ *cobra.Command, _ []string) error { return nil }})
	noisy.AddCommand(&cobra.Command{Use: "secret", Short: "secret", Hidden: true, RunE: func(_ *cobra.Command, _ []string) error { return nil }})
	noisy.AddCommand(&cobra.Command{Use: "help", Short: "help", RunE: func(_ *cobra.Command, _ []string) error { return nil }})
	noisy.AddCommand(&cobra.Command{Use: "completion", Short: "completion", RunE: func(_ *cobra.Command, _ []string) error { return nil }})
	root.AddCommand(noisy)

	return root
}

func TestFindParentNeedingPicker_Cases(t *testing.T) {
	root := buildTestRoot()

	cases := []struct {
		name    string
		args    []string
		want    bool
		wantCmd string
	}{
		{name: "empty args", args: []string{}, want: false},
		{name: "leaf command runs cobra", args: []string{"checkout"}, want: false},
		{name: "leaf with positional", args: []string{"checkout", "admin-prod"}, want: false},
		{name: "any flag disables picker", args: []string{"profiles", "--help"}, want: false},
		{name: "leading global flag disables picker", args: []string{"--tenant", "x", "profiles"}, want: false},
		{name: "valid sub disables picker", args: []string{"profiles", "list"}, want: false},
		{name: "unknown sub disables picker", args: []string{"profiles", "bogus"}, want: false},
		{name: "parent triggers picker", args: []string{"profiles"}, want: true, wantCmd: "profiles"},
		{name: "noisy parent triggers picker", args: []string{"noisy"}, want: true, wantCmd: "noisy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := FindParentNeedingPicker(root, tc.args)
			if ok != tc.want {
				t.Fatalf("FindParentNeedingPicker(%v) = (_, %v), want (_, %v)", tc.args, ok, tc.want)
			}
			if tc.want && got.Name() != tc.wantCmd {
				t.Errorf("matched parent = %q, want %q", got.Name(), tc.wantCmd)
			}
		})
	}
}

func TestSubcommandChoices_FiltersHiddenHelpCompletion(t *testing.T) {
	root := buildTestRoot()
	parent, ok := FindParentNeedingPicker(root, []string{"noisy"})
	if !ok {
		t.Fatal("expected noisy to trigger picker")
	}
	choices := SubcommandChoices(parent)
	if len(choices) != 1 || choices[0].Name != "real" {
		t.Errorf("expected only 'real' to surface, got %+v", choices)
	}
}

func TestSubcommandChoices_PreservesShortDescriptions(t *testing.T) {
	root := buildTestRoot()
	parent, _ := FindParentNeedingPicker(root, []string{"profiles"})
	choices := SubcommandChoices(parent)
	if len(choices) != 2 {
		t.Fatalf("expected 2 choices, got %d", len(choices))
	}
	for _, c := range choices {
		if c.Short == "" {
			t.Errorf("choice %q has empty Short", c.Name)
		}
	}
}
