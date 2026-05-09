package resolver

import (
	"strings"

	"github.com/spf13/cobra"
)

// FindParentNeedingPicker reports whether args resolves to a cobra command
// that has subcommands but no Run/RunE of its own, with no further args
// after the command name. In that case cobra would print a help page;
// callers should instead launch the interactive picker so the user can
// drill into a subcommand the same way `bctl` (no args) lets them pick a
// top-level one.
//
// Conservative on flags: any '-'-prefixed token in args disables the
// picker so flows like `bctl profiles --help` and `bctl --tenant foo
// profiles` fall through to cobra unchanged.
//
// args is the slice after the program name (typically os.Args[1:]).
func FindParentNeedingPicker(root *cobra.Command, args []string) (*cobra.Command, bool) {
	if len(args) == 0 {
		return nil, false
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return nil, false
		}
	}
	matched, remaining, err := root.Find(args)
	if err != nil || matched == nil || matched == root {
		return nil, false
	}
	if len(remaining) > 0 {
		return nil, false
	}
	if !matched.HasSubCommands() || matched.Runnable() {
		return nil, false
	}
	return matched, true
}

// SubcommandChoices builds the picker entries for the immediate subcommands
// of a parent command. Cobra's auto-generated 'help' and 'completion' entries
// are filtered out so users only see real workflow choices.
func SubcommandChoices(parent *cobra.Command) []CommandChoice {
	choices := make([]CommandChoice, 0, len(parent.Commands()))
	for _, c := range parent.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		choices = append(choices, CommandChoice{Name: c.Name(), Short: c.Short})
	}
	return choices
}
