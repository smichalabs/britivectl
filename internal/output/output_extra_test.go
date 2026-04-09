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

// captureStderr temporarily redirects os.Stderr and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestSuccess(t *testing.T) {
	got := captureStdout(t, func() {
		output.Success("hello %s", "world")
	})
	if !strings.Contains(got, "hello world") {
		t.Errorf("Success() output = %q, want it to contain %q", got, "hello world")
	}
}

func TestError(t *testing.T) {
	got := captureStderr(t, func() {
		output.Error("bad %s", "thing")
	})
	if !strings.Contains(got, "bad thing") {
		t.Errorf("Error() output = %q, want it to contain %q", got, "bad thing")
	}
}

func TestWarning(t *testing.T) {
	got := captureStdout(t, func() {
		output.Warning("warn %s", "msg")
	})
	if !strings.Contains(got, "warn msg") {
		t.Errorf("Warning() output = %q, want it to contain %q", got, "warn msg")
	}
}

func TestInfo(t *testing.T) {
	got := captureStdout(t, func() {
		output.Info("info %s", "msg")
	})
	if !strings.Contains(got, "info msg") {
		t.Errorf("Info() output = %q, want it to contain %q", got, "info msg")
	}
}

func TestPrintJSON_Error(t *testing.T) {
	err := output.PrintJSON(make(chan int))
	if err == nil {
		t.Error("PrintJSON(chan int) expected an error, got nil")
	}
}

func TestPrintAWSCredsProcess_NoSessionToken(t *testing.T) {
	got := captureStdout(t, func() {
		output.PrintAWSCredsProcess(map[string]string{
			"AccessKeyId":     "AKIA1234",
			"SecretAccessKey": "secret",
			// SessionToken intentionally omitted / empty
			"SessionToken": "",
		})
	})

	var v map[string]interface{}
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Fatalf("PrintAWSCredsProcess() produced invalid JSON: %v\nOutput: %s", err, got)
	}

	if _, ok := v["SessionToken"]; ok {
		t.Errorf("PrintAWSCredsProcess() output contains SessionToken key, but it should be absent when empty")
	}
}

func TestPrintAWSCredsProcess_NoExpiration(t *testing.T) {
	got := captureStdout(t, func() {
		output.PrintAWSCredsProcess(map[string]string{
			"AccessKeyId":     "AKIA1234",
			"SecretAccessKey": "secret",
			// Expiration intentionally omitted / empty
			"Expiration": "",
		})
	})

	var v map[string]interface{}
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Fatalf("PrintAWSCredsProcess() produced invalid JSON: %v\nOutput: %s", err, got)
	}

	if _, ok := v["Expiration"]; ok {
		t.Errorf("PrintAWSCredsProcess() output contains Expiration key, but it should be absent when empty")
	}
}
