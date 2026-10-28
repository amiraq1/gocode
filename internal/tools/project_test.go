package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateOutputUTF8(t *testing.T) {
	// نص عربي طويل للتحقق من عدم كسر محارف UTF-8 عند القص
	input := strings.Repeat("مرحبا بالعالم في بيئة تيرمكس ", 100)
	out := TruncateOutput(input, 200)
	if !utf8.ValidString(out) {
		t.Fatal("TruncateOutput produced invalid UTF-8 string")
	}
	if !strings.Contains(out, "[truncated") {
		t.Fatal("expected truncation marker in output")
	}
}

func TestInspectProject(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "node_modules", "pkg", "index.js"), []byte(""), 0o644)

	summary, err := InspectProject(root, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "Go (go.mod)") {
		t.Errorf("expected Go stack detection, got:\n%s", summary)
	}
	if strings.Contains(summary, "node_modules") {
		t.Errorf("node_modules should be ignored, got:\n%s", summary)
	}
}
