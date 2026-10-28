package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultMaxDepth = 3
	maxTreeEntries  = 200
)

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, "target": true,
	".next": true, "__pycache__": true, ".gradle": true,
	".idea": true, ".vscode": true, ".cache": true,
}

var stackMarkers = []struct {
	File, Stack string
}{
	{"go.mod", "Go"},
	{"package.json", "Node.js / JavaScript"},
	{"tsconfig.json", "TypeScript"},
	{"Cargo.toml", "Rust"},
	{"pyproject.toml", "Python"},
	{"requirements.txt", "Python"},
	{"Makefile", "Make"},
	{"CMakeLists.txt", "C/C++ (CMake)"},
	{"build.gradle", "Android / Gradle"},
	{"build.gradle.kts", "Android / Kotlin (Gradle)"},
}

type summaryArgs struct {
	Path     string `json:"path"`
	MaxDepth int    `json:"max_depth"`
}

func ProjectTools(sb *Sandbox) []*Tool {
	return []*Tool{
		{
			Name:        "project_summary",
			Description: "Inspect the workspace: detect tech stack, count files by extension, and show a directory tree (ignoring .git, node_modules, vendor, etc.).",
			Params:      json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Subdirectory to inspect, default '.'"},"max_depth":{"type":"integer","description":"Max tree depth (1-6), default 3"}}}`),
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a summaryArgs
				_ = json.Unmarshal(raw, &a)
				if a.MaxDepth <= 0 {
					a.MaxDepth = defaultMaxDepth
				} else if a.MaxDepth > 6 {
					a.MaxDepth = 6
				}
				target, err := sb.Resolve(a.Path)
				if err != nil {
					return "", err
				}
				return InspectProject(target, a.MaxDepth)
			},
		},
	}
}

func InspectProject(root string, maxDepth int) (string, error) {
	var stacks []string
	for _, m := range stackMarkers {
		if _, err := os.Stat(filepath.Join(root, m.File)); err == nil {
			stacks = append(stacks, fmt.Sprintf("%s (%s)", m.Stack, m.File))
		}
	}

	extCounts := map[string]int{}
	var treeLines []string
	totalFiles, totalDirs := 0, 0
	truncatedTree := false

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // تجاوز الملفات التي لا نملك صلاحية قراءتها بهدوء
		}
		if p == root {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if ignoredDirs[name] || (strings.HasPrefix(name, ".") && name != ".github") {
				return filepath.SkipDir
			}
			totalDirs++
		} else {
			if strings.HasSuffix(name, ".gocode-tmp") {
				return nil
			}
			totalFiles++
			ext := strings.ToLower(filepath.Ext(name))
			if ext == "" {
				ext = "(no ext)"
			}
			extCounts[ext]++
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator)) + 1
		if depth <= maxDepth {
			if len(treeLines) < maxTreeEntries {
				indent := strings.Repeat("  ", depth-1)
				label := name
				if d.IsDir() {
					label += "/"
				}
				treeLines = append(treeLines, indent+label)
			} else {
				truncatedTree = true
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if len(stacks) > 0 {
		fmt.Fprintf(&b, "Detected stack: %s\n", strings.Join(stacks, ", "))
	} else {
		b.WriteString("Detected stack: unknown / generic\n")
	}
	fmt.Fprintf(&b, "Totals: %d files, %d directories\n", totalFiles, totalDirs)

	if len(extCounts) > 0 {
		type kv struct {
			ext   string
			count int
		}
		var list []kv
		for k, v := range extCounts {
			list = append(list, kv{k, v})
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].count != list[j].count {
				return list[i].count > list[j].count
			}
			return list[i].ext < list[j].ext
		})
		if len(list) > 8 {
			list = list[:8]
		}
		var parts []string
		for _, item := range list {
			parts = append(parts, fmt.Sprintf("%s: %d", item.ext, item.count))
		}
		fmt.Fprintf(&b, "Top file types: %s\n", strings.Join(parts, ", "))
	}

	b.WriteString("\nWorkspace tree:\n")
	if len(treeLines) == 0 {
		b.WriteString("  (empty)\n")
	} else {
		for _, line := range treeLines {
			b.WriteString("  " + line + "\n")
		}
		if truncatedTree {
			fmt.Fprintf(&b, "  ... (tree capped at %d entries)\n", maxTreeEntries)
		}
	}
	return b.String(), nil
}
