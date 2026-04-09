package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
)

func init() {
	// Respect BCTL_NO_COLOR environment variable
	if os.Getenv("BCTL_NO_COLOR") != "" || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}
}

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgCyan)
)

// Success prints a green success message to stdout.
func Success(format string, args ...interface{}) {
	_, _ = successColor.Fprintf(os.Stdout, "✓ "+format+"\n", args...)
}

// Error prints a red error message to stderr.
func Error(format string, args ...interface{}) {
	_, _ = errorColor.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

// Warning prints a yellow warning message to stdout.
func Warning(format string, args ...interface{}) {
	_, _ = warningColor.Fprintf(os.Stdout, "⚠ "+format+"\n", args...)
}

// Info prints a cyan info message to stdout.
func Info(format string, args ...interface{}) {
	_, _ = infoColor.Fprintf(os.Stdout, "  "+format+"\n", args...)
}

// PrintJSON marshals v to indented JSON and writes to stdout.
func PrintJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// PrintEnv prints key=value pairs suitable for shell eval.
func PrintEnv(kv map[string]string) {
	for k, v := range kv {
		fmt.Printf("export %s=%q\n", k, v)
	}
}

// PrintAWSCredsProcess prints AWS credential_process JSON output.
func PrintAWSCredsProcess(creds map[string]string) {
	out := map[string]interface{}{
		"Version":         1,
		"AccessKeyId":     creds["AccessKeyId"],
		"SecretAccessKey": creds["SecretAccessKey"],
	}
	if v, ok := creds["SessionToken"]; ok && v != "" {
		out["SessionToken"] = v
	}
	if v, ok := creds["Expiration"]; ok && v != "" {
		out["Expiration"] = v
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}
