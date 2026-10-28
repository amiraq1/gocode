package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type editArgs struct {
	Path       string `json:"path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
}

func EditTools(sb *Sandbox) []*Tool {
	return []*Tool{
		{
			Name:          "edit_file",
			Description:   "Surgically edit a file by replacing an exact old_string with new_string. The old_string must match uniquely unless replace_all is true. Shows a preview and requires approval.",
			Params:        json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"old_string":{"type":"string","description":"Exact text to find and replace"},"new_string":{"type":"string","description":"Replacement text"},"replace_all":{"type":"boolean","description":"Replace all occurrences, default false"}},"required":["path","old_string","new_string"]}`),
			NeedsApproval: true,
			Describe: func(raw json.RawMessage) string {
				var a editArgs
				if json.Unmarshal(raw, &a) != nil {
					return "edit_file (invalid arguments)"
				}
				return formatEditPreview(a.Path, a.OldString, a.NewString, a.ReplaceAll)
			},
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a editArgs
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", fmt.Errorf("bad arguments: %w", err)
				}
				if a.OldString == "" {
					return "", fmt.Errorf("old_string cannot be empty")
				}
				p, err := sb.Resolve(a.Path)
				if err != nil {
					return "", err
				}

				data, err := os.ReadFile(p)
				if err != nil {
					return "", err
				}
				content := string(data)
				count := strings.Count(content, a.OldString)
				if count == 0 {
					return "", fmt.Errorf("old_string not found in %s", a.Path)
				}
				if count > 1 && !a.ReplaceAll {
					return "", fmt.Errorf("old_string matches %d times in %s; set replace_all=true or make it unique", count, a.Path)
				}

				newContent := strings.ReplaceAll(content, a.OldString, a.NewString)
				if newContent == content {
					return "no changes made", nil
				}

				tmp := p + ".gocode-tmp"
				if err := os.WriteFile(tmp, []byte(newContent), 0o644); err != nil {
					return "", err
				}
				if err := os.Rename(tmp, p); err != nil {
					os.Remove(tmp)
					return "", err
				}
				return fmt.Sprintf("edited %s (%d replacement(s))", a.Path, count), nil
			},
		},
	}
}

func formatEditPreview(path, oldStr, newStr string, replaceAll bool) string {
	var b strings.Builder
	mode := "single match"
	if replaceAll {
		mode = "replace_all"
	}
	fmt.Fprintf(&b, "✎ edit_file %s (%s)\n", path, mode)
	for _, line := range strings.Split(oldStr, "\n") {
		fmt.Fprintf(&b, "\033[31m- %s\033[0m\n", line)
	}
	for _, line := range strings.Split(newStr, "\n") {
		fmt.Fprintf(&b, "\033[32m+ %s\033[0m\n", line)
	}
	return strings.TrimRight(b.String(), "\n")
}
