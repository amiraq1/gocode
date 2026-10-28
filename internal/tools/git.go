package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func GitStatusTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:        "git_status",
		Description: "Show git status of the workspace (short format).",
		Params:      json.RawMessage(`{"type":"object","properties":{}}`),
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			out, code, err := ExecSafe(ctx, sb.Root, []string{"git", "status", "--short", "--branch"}, 15*time.Second)
			if err != nil {
				return "", err
			}
			if code != 0 {
				return fmt.Sprintf("git status failed (exit %d):\n%s", code, out), nil
			}
			if out == "" {
				return "clean working tree", nil
			}
			return out, nil
		},
	}
}

func GitLogTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:        "git_log",
		Description: "Show recent git commits (default 10).",
		Params:      json.RawMessage(`{"type":"object","properties":{"count":{"type":"integer","description":"Number of commits, default 10, max 30"}}}`),
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			var a struct{ Count int `json:"count"` }
			json.Unmarshal(raw, &a)
			if a.Count <= 0 || a.Count > 30 {
				a.Count = 10
			}
			args := []string{"git", "log", "--oneline", "-n", fmt.Sprintf("%d", a.Count)}
			out, code, err := ExecSafe(ctx, sb.Root, args, 15*time.Second)
			if err != nil {
				return "", err
			}
			if code != 0 {
				return fmt.Sprintf("git log failed (exit %d):\n%s", code, out), nil
			}
			if out == "" {
				return "no commits yet", nil
			}
			return out, nil
		},
	}
}

func GitAddTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:          "git_add",
		Description:   "Stage files for commit. Requires approval.",
		Params:        json.RawMessage(`{"type":"object","properties":{"paths":{"type":"array","items":{"type":"string"},"description":"Paths to stage, or ['.'] for all"}},"required":["paths"]}`),
		NeedsApproval: true,
		Describe: func(raw json.RawMessage) string {
			var a struct{ Paths []string `json:"paths"` }
			json.Unmarshal(raw, &a)
			return fmt.Sprintf("⚡ git add -- %s", strings.Join(a.Paths, " "))
		},
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			var a struct{ Paths []string `json:"paths"` }
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", fmt.Errorf("bad arguments: %w", err)
			}
			if len(a.Paths) == 0 {
				return "", fmt.Errorf("no paths provided")
			}
			safePaths := make([]string, 0, len(a.Paths))
			for _, p := range a.Paths {
				resolved, err := sb.Resolve(p)
				if err != nil {
					return "", fmt.Errorf("invalid path %q: %w", p, err)
				}
				rel, err := filepath.Rel(sb.Root, resolved)
				if err != nil || strings.HasPrefix(rel, "..") {
					return "", fmt.Errorf("path outside workspace: %s", p)
				}
				safePaths = append(safePaths, rel)
			}
			args := append([]string{"git", "add", "--"}, safePaths...)
			out, code, err := ExecSafe(ctx, sb.Root, args, 20*time.Second)
			if err != nil {
				return "", err
			}
			if code != 0 {
				return fmt.Sprintf("git add failed (exit %d):\n%s", code, out), nil
			}
			return "files staged successfully", nil
		},
	}
}

func GitCommitTool(sb *Sandbox) *Tool {
	return &Tool{
		Name:          "git_commit",
		Description:   "Create a git commit with a message. Requires approval. Does not push.",
		Params:        json.RawMessage(`{"type":"object","properties":{"message":{"type":"string","description":"Commit message"}},"required":["message"]}`),
		NeedsApproval: true,
		Describe: func(raw json.RawMessage) string {
			var a struct{ Message string `json:"message"` }
			json.Unmarshal(raw, &a)
			// عرض ما سيتم تضمينه في الـ commit
			status, _, _ := ExecSafe(context.Background(), sb.Root, []string{"git", "status", "--short"}, 10*time.Second)
			return fmt.Sprintf("⚡ git commit -m %q\n\nStaged/changed:\n%s", a.Message, status)
		},
		Run: func(ctx context.Context, raw json.RawMessage) (string, error) {
			var a struct{ Message string `json:"message"` }
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", fmt.Errorf("bad arguments: %w", err)
			}
			msg := strings.TrimSpace(a.Message)
			if msg == "" {
				return "", fmt.Errorf("commit message cannot be empty")
			}
			if len(msg) > 200 {
				return "", fmt.Errorf("commit message too long")
			}
			args := []string{"git", "commit", "-m", msg}
			out, code, err := ExecSafe(ctx, sb.Root, args, 30*time.Second)
			if err != nil {
				return "", err
			}
			if code != 0 {
				return fmt.Sprintf("git commit failed (exit %d):\n%s", code, out), nil
			}
			return out, nil
		},
	}
}

func AdvancedGitTools(sb *Sandbox) []*Tool {
	return []*Tool{
		GitStatusTool(sb),
		GitLogTool(sb),
		GitAddTool(sb),
		GitCommitTool(sb),
	}
}
