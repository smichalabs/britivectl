// Package resolver matches user-provided profile shorthands against the
// local profile cache and falls back to an interactive picker when the
// shorthand is missing or ambiguous.
package resolver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/smichalabs/britivectl/internal/config"
)

// ErrNoMatch is returned when no profile matches the given query and the
// caller has disabled interactive selection.
var ErrNoMatch = errors.New("no matching profile")

// ErrCanceled is returned when the user cancels interactive selection.
var ErrCanceled = errors.New("selection canceled")

// Match is a single profile that matched a resolver query.
type Match struct {
	Alias   string
	Profile config.Profile
}

// Resolve returns the single profile that matches query. If the query is
// empty or ambiguous, the user is prompted to pick one interactively.
//
// Matching rules (in priority order):
//  1. Exact alias match -> return immediately
//  2. Substring match (case-insensitive) on alias or britive_path
//  3. Fuzzy match on alias (all query characters appear in order)
//
// If multiple profiles survive all three stages, the picker is shown.
func Resolve(ctx context.Context, profiles map[string]config.Profile, query string, in io.Reader, out io.Writer) (Match, error) {
	if len(profiles) == 0 {
		return Match{}, errors.New("no profiles available -- run 'bctl profiles sync'")
	}

	// Exact match first -- always wins.
	if query != "" {
		if p, ok := profiles[query]; ok {
			return Match{Alias: query, Profile: p}, nil
		}
	}

	// Flatten for ranking.
	all := make([]Match, 0, len(profiles))
	for alias, profile := range profiles {
		all = append(all, Match{Alias: alias, Profile: profile})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Alias < all[j].Alias })

	matches := filter(all, query)

	switch len(matches) {
	case 0:
		if query == "" {
			return pick(ctx, all, in, out)
		}
		return Match{}, fmt.Errorf("%w: %q", ErrNoMatch, query)
	case 1:
		return matches[0], nil
	default:
		return pick(ctx, matches, in, out)
	}
}

// filter applies the substring + fuzzy rules. An empty query returns nil
// (caller interprets as "show all").
func filter(all []Match, query string) []Match {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)

	// Substring match on alias or britive_path.
	var subs []Match
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.Alias), q) ||
			strings.Contains(strings.ToLower(m.Profile.BritivePath), q) {
			subs = append(subs, m)
		}
	}
	if len(subs) > 0 {
		return subs
	}

	// Fuzzy match (subsequence) on alias.
	var fuzzy []Match
	for _, m := range all {
		if isSubsequence(q, strings.ToLower(m.Alias)) {
			fuzzy = append(fuzzy, m)
		}
	}
	return fuzzy
}

// isSubsequence reports whether every rune of needle appears in haystack in
// the same order. Used for the fuzzy fallback.
func isSubsequence(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	nr := []rune(needle)
	hr := []rune(haystack)
	i := 0
	for j := 0; j < len(hr) && i < len(nr); j++ {
		if hr[j] == nr[i] {
			i++
		}
	}
	return i == len(nr)
}

// pick prompts the user to choose from a list of matches. When stdin is a
// real TTY, it launches a bubbletea list with live filtering. When stdin is
// piped (tests, CI, scripts), it falls back to a numbered prompt so the
// caller can pipe in a selection.
func pick(ctx context.Context, matches []Match, in io.Reader, out io.Writer) (Match, error) {
	if isTTY(in) {
		return interactivePick(ctx, matches)
	}
	return numberedPick(ctx, matches, in, out)
}

// numberedPick is the non-TTY fallback: a printed list with a number prompt.
// Used by tests and any script that pipes input into bctl.
func numberedPick(ctx context.Context, matches []Match, in io.Reader, out io.Writer) (Match, error) {
	fmt.Fprintln(out, "Multiple profiles matched. Pick one:")
	fmt.Fprintln(out)
	for i, m := range matches {
		path := m.Profile.BritivePath
		if path == "" {
			path = "(no path)"
		}
		cloud := m.Profile.Cloud
		if cloud == "" {
			cloud = "?"
		}
		fmt.Fprintf(out, "  [%d] %-30s  %-6s  %s\n", i+1, m.Alias, cloud, path)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Enter number (or 'q' to cancel):")

	// Respect context cancellation by reading in a goroutine.
	type result struct {
		choice string
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		reader := bufio.NewReader(in)
		line, err := reader.ReadString('\n')
		ch <- result{choice: strings.TrimSpace(line), err: err}
	}()

	select {
	case <-ctx.Done():
		return Match{}, ctx.Err()
	case r := <-ch:
		if errors.Is(r.err, io.EOF) && r.choice == "" {
			return Match{}, ErrCanceled
		}
		if r.err != nil && !errors.Is(r.err, io.EOF) {
			return Match{}, fmt.Errorf("reading selection: %w", r.err)
		}
		if r.choice == "" || r.choice == "q" || r.choice == "Q" {
			return Match{}, ErrCanceled
		}
		idx, err := strconv.Atoi(r.choice)
		if err != nil || idx < 1 || idx > len(matches) {
			return Match{}, fmt.Errorf("invalid selection %q", r.choice)
		}
		return matches[idx-1], nil
	}
}
