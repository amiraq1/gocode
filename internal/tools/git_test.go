package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitTools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test Agent")

	sb, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry(func(string) bool { return true })
	for _, tool := range AdvancedGitTools(sb) {
		reg.Register(tool)
	}

	ctx := context.Background()

	if err := os.WriteFile(filepath.Join(root, "hello.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	statusOut := reg.Execute(ctx, "git_status", `{}`)
	if !strings.Contains(statusOut, "hello.go") {
		t.Fatalf("expected hello.go in git_status, got: %s", statusOut)
	}

	badAdd := reg.Execute(ctx, "git_add", `{"paths":["../outside.txt"]}`)
	if !strings.Contains(badAdd, "error:") && !strings.Contains(badAdd, "invalid") {
		t.Fatalf("expected error for ../ path in git_add, got: %s", badAdd)
	}

	addOut := reg.Execute(ctx, "git_add", `{"paths":["hello.go"]}`)
	if strings.Contains(addOut, "error:") {
		t.Fatalf("git_add failed: %s", addOut)
	}

	commitArgs, _ := json.Marshal(map[string]string{"message": "feat: initial file"})
	commitOut := reg.Execute(ctx, "git_commit", string(commitArgs))
	if strings.Contains(commitOut, "error:") {
		t.Fatalf("git_commit failed: %s", commitOut)
	}

	logOut := reg.Execute(ctx, "git_log", `{"count":5}`)
	if !strings.Contains(logOut, "feat: initial file") {
		t.Fatalf("expected commit message in git_log, got: %s", logOut)
	}
}
