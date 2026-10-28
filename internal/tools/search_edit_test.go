package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc hello() {}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "util.go"), []byte("func helloWorld() {}\n"), 0o644)

	out, err := searchInPath(root, "hello", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "main.go:2") || !strings.Contains(out, "util.go:1") {
		t.Errorf("unexpected search results: %s", out)
	}
}

func TestEditFile(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "test.go")
	_ = os.WriteFile(path, []byte("func foo() {\n  return 1\n}\n"), 0o644)

	tools := EditTools(sb)
	editTool := tools[0]

	raw := []byte(`{"path":"test.go","old_string":"return 1","new_string":"return 42"}`)
	out, err := editTool.Run(nil, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "edited") {
		t.Errorf("expected success, got: %s", out)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "return 42") {
		t.Error("file content not updated")
	}
}
