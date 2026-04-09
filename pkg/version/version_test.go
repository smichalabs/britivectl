package version_test

import (
	"strings"
	"testing"

	"github.com/smichalabs/britivectl/pkg/version"
)

func TestString(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
		wantParts []string
	}{
		{
			name:      "default values",
			version:   "0.0.1-alpha",
			commit:    "dev",
			buildDate: "unknown",
			wantParts: []string{"bctl", "0.0.1-alpha", "commit:", "dev", "built:", "unknown"},
		},
		{
			name:      "release values",
			version:   "1.2.3",
			commit:    "abc1234",
			buildDate: "2026-04-09T00:00:00Z",
			wantParts: []string{"bctl", "1.2.3", "commit:", "abc1234", "built:", "2026-04-09"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override package vars
			orig := version.Version
			origC := version.Commit
			origB := version.BuildDate
			defer func() {
				version.Version = orig
				version.Commit = origC
				version.BuildDate = origB
			}()
			version.Version = tt.version
			version.Commit = tt.commit
			version.BuildDate = tt.buildDate

			got := version.String()
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("String() = %q, want it to contain %q", got, part)
				}
			}
		})
	}
}
