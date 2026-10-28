package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxReadBytes = 100 * 1024
	maxDirItems  = 500
)

type pathArgs struct {
	Path string `json:"path"`
}

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func FSTools(sb *Sandbox) []*Tool {
	return []*Tool{
		{
			Name:        "read_file",
			Description: "Read a UTF-8 text file from the workspace (max 100KB).",
			Params:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Path relative to workspace root"}},"required":["path"]}`),
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a pathArgs
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", fmt.Errorf("bad arguments: %w", err)
				}
				p, err := sb.Resolve(a.Path)
				if err != nil {
					return "", err
				}
				info, err := os.Stat(p)
				if err != nil {
					return "", err
				}
				if info.IsDir() {
					return "", errors.New("path is a directory; use list_dir")
				}
				if info.Size() > maxReadBytes {
					return "", fmt.Errorf("file too large (%d bytes, limit %d)", info.Size(), maxReadBytes)
				}
				data, err := os.ReadFile(p)
				if err != nil {
					return "", err
				}
				if bytes.IndexByte(data, 0) >= 0 {
					return "", errors.New("binary file, refusing to read")
				}
				return string(data), nil
			},
		},
		{
			Name:        "list_dir",
			Description: "List entries of a directory in the workspace. Directories end with '/'.",
			Params:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Directory path, default '.'"}}}`),
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a pathArgs
				_ = json.Unmarshal(raw, &a)
				p, err := sb.Resolve(a.Path)
				if err != nil {
					return "", err
				}
				entries, err := os.ReadDir(p)
				if err != nil {
					return "", err
				}
				var b strings.Builder
				for i, e := range entries {
					if i >= maxDirItems {
						fmt.Fprintf(&b, "... (%d more)\n", len(entries)-maxDirItems)
						break
					}
					name := e.Name()
					if e.IsDir() {
						name += "/"
					}
					b.WriteString(name + "\n")
				}
				if b.Len() == 0 {
					return "(empty directory)", nil
				}
				return b.String(), nil
			},
		},
		{
			Name:          "write_file",
			Description:   "Create or overwrite a file with the full given content. Parent directories are created.",
			Params:        json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
			NeedsApproval: true,
			Describe: func(raw json.RawMessage) string {
				var a writeArgs
				if json.Unmarshal(raw, &a) != nil {
					return "write_file (invalid arguments)"
				}
				lines := strings.Split(a.Content, "\n")
				preview := lines
				if len(preview) > 15 {
					preview = append(preview[:15:15], "…")
				}
				return fmt.Sprintf("✎ write_file %s (%d bytes, %d lines)\n%s",
					a.Path, len(a.Content), len(lines), strings.Join(preview, "\n"))
			},
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a writeArgs
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", fmt.Errorf("bad arguments: %w", err)
				}
				p, err := sb.Resolve(a.Path)
				if err != nil {
					return "", err
				}
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					return "", errors.New("path is a directory")
				}
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return "", err
				}
				tmp := p + ".gocode-tmp" // كتابة ذرية: لا ملف نصف مكتوب عند الانقطاع
				if err := os.WriteFile(tmp, []byte(a.Content), 0o644); err != nil {
					return "", err
				}
				if err := os.Rename(tmp, p); err != nil {
					os.Remove(tmp)
					return "", err
				}
				return fmt.Sprintf("wrote %d bytes to %s", len(a.Content), a.Path), nil
			},
		},
	}
}
