package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gocode/internal/provider"
)

func TestStoreSaveLoadAndMemory(t *testing.T) {
	t.Setenv("GOCODE_DATA_DIR", t.TempDir())
	ws := t.TempDir()

	store, err := NewStore(ws)
	if err != nil {
		t.Fatal(err)
	}

	msgs1 := []provider.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "السؤال الأول"},
		{Role: "assistant", Content: "الجواب الأول"},
	}
	if err := store.Save("s1", msgs1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	msgs2 := append(msgs1, provider.Message{Role: "user", Content: "السؤال الثاني"})
	if err := store.Save("s2", msgs2); err != nil {
		t.Fatal(err)
	}

	latest, err := store.LoadLatest()
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != "s2" || len(latest.Messages) != 4 {
		t.Fatalf("expected latest session s2 with 4 messages, got %+v", latest)
	}

	_ = os.WriteFile(filepath.Join(ws, "GOCODE.md"), []byte("Use Go 1.22 conventions."), 0o644)
	if err := store.AppendMemory("Always run go test before commit"); err != nil {
		t.Fatal(err)
	}

	mem := store.LoadMemory()
	if !strings.Contains(mem, "Use Go 1.22 conventions.") || !strings.Contains(mem, "Always run go test") {
		t.Fatalf("unexpected memory content:\n%s", mem)
	}
}
