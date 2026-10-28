package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	root := t.TempDir()
	sb, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"a.go", "sub/b.go", "./x/../y", "."} {
		if _, err := sb.Resolve(p); err != nil {
			t.Errorf("%q should be allowed: %v", p, err)
		}
	}
	for _, p := range []string{"..", "../etc", "/etc/passwd", "sub/../../x"} {
		if _, err := sb.Resolve(p); err == nil {
			t.Errorf("%q should be rejected", p)
		}
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(sb.Root, "link")); err == nil {
		if _, err := sb.Resolve("link/secret.txt"); err == nil {
			t.Error("symlink escape was not blocked")
		}
	}

	// اختبار رابط رمزي معلّق يشير إلى ملف غير موجود خارج المشروع
	danglingTarget := filepath.Join(outside, "nonexistent", "file.txt")
	if err := os.Symlink(danglingTarget, filepath.Join(sb.Root, "dangling")); err == nil {
		if _, err := sb.Resolve("dangling"); err == nil {
			t.Error("dangling symlink escape was not blocked")
		}
	}
}
