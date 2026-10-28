package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxSearchResults = 50
	maxSearchFileSize = 1 << 20 // 1MB
)

type searchArgs struct {
	Query         string `json:"query"`
	Path          string `json:"path"`
	CaseSensitive bool   `json:"case_sensitive"`
	MaxResults    int    `json:"max_results"`
}

func SearchTools(sb *Sandbox) []*Tool {
	return []*Tool{
		{
			Name:        "search",
			Description: "Search for a text string inside files in the workspace. Returns matching lines with file path and line number. Skips binary files and ignored directories.",
			Params:      json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Text to search for"},"path":{"type":"string","description":"Directory or file to search in, default '.'"},"case_sensitive":{"type":"boolean","description":"Case sensitive search, default false"},"max_results":{"type":"integer","description":"Maximum number of matches to return, default 50"}},"required":["query"]}`),
			Run: func(_ context.Context, raw json.RawMessage) (string, error) {
				var a searchArgs
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", fmt.Errorf("bad arguments: %w", err)
				}
				if a.Query == "" {
					return "", fmt.Errorf("query cannot be empty")
				}
				if a.MaxResults <= 0 || a.MaxResults > maxSearchResults {
					a.MaxResults = maxSearchResults
				}
				target, err := sb.Resolve(a.Path)
				if err != nil {
					return "", err
				}
				return searchInPath(target, a.Query, a.CaseSensitive, a.MaxResults)
			},
		},
	}
}

func searchInPath(root, query string, caseSensitive bool, maxResults int) (string, error) {
	var results []string
	queryLower := query
	if !caseSensitive {
		queryLower = strings.ToLower(query)
	}

	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || len(results) >= maxResults {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if ignoredDirs[name] || (strings.HasPrefix(name, ".") && name != ".github") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".gocode-tmp") {
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > maxSearchFileSize {
			return nil
		}

		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() && len(results) < maxResults {
			lineNum++
			line := scanner.Text()
			if !utf8.ValidString(line) {
				continue
			}
			searchIn := line
			if !caseSensitive {
				searchIn = strings.ToLower(line)
			}
			if strings.Contains(searchIn, queryLower) {
				rel, _ := filepath.Rel(root, p)
				results = append(results, fmt.Sprintf("%s:%d:%s", rel, lineNum, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "no matches found", nil
	}
	return strings.Join(results, "\n"), nil
}
