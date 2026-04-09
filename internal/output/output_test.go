package output_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/smichalabs/britivectl/internal/output"
)

// captureStdout temporarily redirects os.Stdout and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name    string
		input   interface{}
		wantKey string
	}{
		{
			name:    "struct",
			input:   payload{Name: "alice", Age: 30},
			wantKey: `"name"`,
		},
		{
			name:    "map",
			input:   map[string]string{"foo": "bar"},
			wantKey: `"foo"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStdout(t, func() {
				if err := output.PrintJSON(tt.input); err != nil {
					t.Fatalf("PrintJSON() error: %v", err)
				}
			})
			if !strings.Contains(got, tt.wantKey) {
				t.Errorf("PrintJSON() output = %q, want it to contain %q", got, tt.wantKey)
			}
			// Must be valid JSON
			var v interface{}
			if err := json.Unmarshal([]byte(got), &v); err != nil {
				t.Errorf("PrintJSON() produced invalid JSON: %v\nOutput: %s", err, got)
			}
		})
	}
}

func TestPrintEnv(t *testing.T) {
	got := captureStdout(t, func() {
		output.PrintEnv(map[string]string{
			"AWS_ACCESS_KEY_ID": "AKIA1234",
		})
	})
	if !strings.Contains(got, "AWS_ACCESS_KEY_ID") {
		t.Errorf("PrintEnv() output = %q, want AWS_ACCESS_KEY_ID", got)
	}
	if !strings.Contains(got, "export") {
		t.Errorf("PrintEnv() output = %q, want 'export' keyword", got)
	}
}

func TestPrintAWSCredsProcess(t *testing.T) {
	got := captureStdout(t, func() {
		output.PrintAWSCredsProcess(map[string]string{
			"AccessKeyId":     "AKIA1234",
			"SecretAccessKey": "secret",
			"SessionToken":    "token",
			"Expiration":      "2026-04-09T00:00:00Z",
		})
	})
	var v map[string]interface{}
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Fatalf("PrintAWSCredsProcess() invalid JSON: %v\nOutput: %s", err, got)
	}
	if v["Version"] != float64(1) {
		t.Errorf("Version = %v, want 1", v["Version"])
	}
	if v["AccessKeyId"] != "AKIA1234" {
		t.Errorf("AccessKeyId = %v, want AKIA1234", v["AccessKeyId"])
	}
}

func TestPrintTable(t *testing.T) {
	got := captureStdout(t, func() {
		output.PrintTable(
			[]string{"NAME", "STATUS"},
			[][]string{
				{"profile-a", "active"},
				{"profile-b", "expired"},
			},
		)
	})
	if !strings.Contains(got, "profile-a") {
		t.Errorf("PrintTable() output = %q, want 'profile-a'", got)
	}
	if !strings.Contains(got, "NAME") {
		t.Errorf("PrintTable() output = %q, want 'NAME' header", got)
	}
}
