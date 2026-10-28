package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultTimeout = 90 * time.Second
	maxOutputBytes = 32 * 1024
)

type cmdArgs struct {
	Args       []string `json:"args"`
	TimeoutSec int      `json:"timeout_sec,omitempty"`
}

// ExecSafe يشغّل أمراً بدون shell، بمعاملات منفصلة، مع مهلة وحد للمخرجات.
func ExecSafe(ctx context.Context, dir string, args []string, timeout time.Duration) (string, int, error) {
	if len(args) == 0 {
		return "", 0, fmt.Errorf("empty command")
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	output := out.String()
	if len(output) > maxOutputBytes {
		output = TruncateOutput(output, maxOutputBytes)
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			return output + "\n[timeout]", -1, nil
		} else {
			return "", 0, err
		}
	}
	return strings.TrimSpace(output), exitCode, nil
}

func GoTestTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:          "go_test",
		Description:   "Run 'go test' in the workspace. Returns output and exit code for self-correction.",
		Params:        json.RawMessage(`{"type":"object","properties":{"packages":{"type":"string","description":"Packages to test, default './...'"},"timeout_sec":{"type":"integer"}}}`),
		NeedsApproval: true,
		Describe: func(raw json.RawMessage) string {
			var a struct{ Packages string `json:"packages"` }
			json.Unmarshal(raw, &a)
			if a.Packages == "" {
				a.Packages = "./..."
			}
			return fmt.Sprintf("⚡ go test %s", a.Packages)
		},
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			var a struct {
				Packages   string `json:"packages"`
				TimeoutSec int    `json:"timeout_sec"`
			}
			json.Unmarshal(raw, &a)
			if a.Packages == "" {
				a.Packages = "./..."
			}
			timeout := time.Duration(a.TimeoutSec) * time.Second
			out, code, err := ExecSafe(ctx, sb.Root, []string{"go", "test", a.Packages}, timeout)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Exit code: %d\nOutput:\n%s", code, out), nil
		},
	}
}

func GoBuildTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:          "go_build",
		Description:   "Run 'go build' to verify the project compiles.",
		Params:        json.RawMessage(`{"type":"object","properties":{"packages":{"type":"string","description":"Packages to build, default './...'"},"timeout_sec":{"type":"integer"}}}`),
		NeedsApproval: true,
		Describe: func(raw json.RawMessage) string {
			var a struct{ Packages string `json:"packages"` }
			json.Unmarshal(raw, &a)
			if a.Packages == "" {
				a.Packages = "./..."
			}
			return fmt.Sprintf("⚡ go build %s", a.Packages)
		},
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			var a struct {
				Packages   string `json:"packages"`
				TimeoutSec int    `json:"timeout_sec"`
			}
			json.Unmarshal(raw, &a)
			if a.Packages == "" {
				a.Packages = "./..."
			}
			timeout := time.Duration(a.TimeoutSec) * time.Second
			out, code, err := ExecSafe(ctx, sb.Root, []string{"go", "build", "-o", "/dev/null", a.Packages}, timeout)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Exit code: %d\nOutput:\n%s", code, out), nil
		},
	}
}

func GitDiffTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:        "git_diff",
		Description: "Show git diff of current changes (read-only).",
		Params:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Optional path to diff"}}}`),
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			var a struct{ Path string `json:"path"` }
			json.Unmarshal(raw, &a)
			args := []string{"git", "diff", "--no-color"}
			if a.Path != "" {
				args = append(args, "--", a.Path)
			}
			out, code, err := ExecSafe(ctx, sb.Root, args, 30*time.Second)
			if err != nil {
				return "", err
			}
			if code != 0 {
				return fmt.Sprintf("git diff failed (exit %d):\n%s", code, out), nil
			}
			if out == "" {
				return "no changes", nil
			}
			return out, nil
		},
	}
}

func DevExecTools(sb *Sandbox) []*Tool {
	return []*Tool{
		GoTestTool(sb),
		GoBuildTool(sb),
		GitDiffTool(sb),
	}
}
