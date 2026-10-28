package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gocode/internal/provider"
)

const maxMemoryBytes = 8 * 1024

type Session struct {
	ID        string             `json:"id"`
	Workspace string             `json:"workspace"`
	UpdatedAt time.Time          `json:"updated_at"`
	Messages  []provider.Message `json:"messages"`
}

type Summary struct {
	ID        string
	UpdatedAt time.Time
	TurnCount int
	Preview   string
}

type Store struct {
	baseDir string
	wsRoot  string
	wsDir   string
}

func NewStore(workspaceRoot string) (*Store, error) {
	base := os.Getenv("GOCODE_DATA_DIR")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		base = filepath.Join(home, ".config", "gocode")
	}
	h := sha256.Sum256([]byte(workspaceRoot))
	wsID := filepath.Base(workspaceRoot) + "-" + hex.EncodeToString(h[:4])
	wsDir := filepath.Join(base, "workspaces", wsID)

	if err := os.MkdirAll(filepath.Join(wsDir, "sessions"), 0o755); err != nil {
		return nil, err
	}
	return &Store{baseDir: base, wsRoot: workspaceRoot, wsDir: wsDir}, nil
}

func NewSessionID() string {
	return time.Now().Format("20060102-150405")
}

func (s *Store) Save(id string, msgs []provider.Message) error {
	if id == "" {
		return errors.New("empty session id")
	}
	sess := Session{
		ID:        id,
		Workspace: s.wsRoot,
		UpdatedAt: time.Now(),
		Messages:  msgs,
	}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(s.wsDir, "sessions", id+".json")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (s *Store) Load(id string) (*Session, error) {
	cleanID := filepath.Base(strings.TrimSuffix(id, ".json"))
	p := filepath.Join(s.wsDir, "sessions", cleanID+".json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) LoadLatest() (*Session, error) {
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, os.ErrNotExist
	}
	return s.Load(list[0].ID)
}

func (s *Store) List() ([]Summary, error) {
	dir := filepath.Join(s.wsDir, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		sess, err := s.Load(id)
		if err != nil {
			continue
		}
		preview := ""
		turns := 0
		for _, m := range sess.Messages {
			if m.Role == "user" {
				turns++
				if preview == "" {
					preview = firstLine(m.Content, 60)
				}
			}
		}
		out = append(out, Summary{
			ID:        sess.ID,
			UpdatedAt: sess.UpdatedAt,
			TurnCount: turns,
			Preview:   preview,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (s *Store) LoadMemory() string {
	var parts []string

	repoDoc := filepath.Join(s.wsRoot, "GOCODE.md")
	if data, err := os.ReadFile(repoDoc); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		parts = append(parts, "### Project GOCODE.md:\n"+strings.TrimSpace(string(data)))
	}

	memFile := filepath.Join(s.wsDir, "memory.md")
	if data, err := os.ReadFile(memFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		parts = append(parts, "### Saved Project Notes:\n"+strings.TrimSpace(string(data)))
	}

	return strings.Join(parts, "\n\n")
}

func (s *Store) AppendMemory(note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return errors.New("empty note")
	}
	memFile := filepath.Join(s.wsDir, "memory.md")
	existing, _ := os.ReadFile(memFile)

	entry := fmt.Sprintf("- %s\n", note)
	combined := string(existing) + entry
	if len(combined) > maxMemoryBytes {
		return fmt.Errorf("memory limit exceeded (%d bytes max)", maxMemoryBytes)
	}

	tmp := memFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(combined), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, memFile)
}

func firstLine(s string, maxRunes int) string {
	line := strings.SplitN(strings.TrimSpace(s), "\n", 2)[0]
	r := []rune(line)
	if len(r) <= maxRunes {
		return line
	}
	return string(r[:maxRunes]) + "…"
}
