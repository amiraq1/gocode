package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Sandbox struct{ Root string }

func NewSandbox(root string) (*Sandbox, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	return &Sandbox{Root: real}, nil
}

// Resolve يحوّل المسار إلى مسار حقيقي مطلق ويرفض أي خروج من الجذر،
// بما في ذلك الخروج عبر الروابط الرمزية.
func (s *Sandbox) Resolve(p string) (string, error) {
	if p == "" {
		p = "."
	}
	var full string
	if filepath.IsAbs(p) {
		full = filepath.Clean(p)
	} else {
		full = filepath.Join(s.Root, p)
	}
	real, err := evalExisting(full)
	if err != nil {
		return "", err
	}
	if !within(s.Root, real) {
		return "", fmt.Errorf("path %q is outside the workspace", p)
	}
	return real, nil
}

// evalExisting يحل الروابط لأطول جزء موجود من المسار (لدعم الملفات الجديدة).
func evalExisting(p string) (string, error) {
	cur, rest := p, ""
	for {
		r, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return filepath.Join(r, rest), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		// إذا كان المسار نفسه رابطاً رمزياً ولكن هدفه غير موجود (dangling symlink) نرفضه فوراً
		if fi, lerr := os.Lstat(cur); lerr == nil && fi.Mode()&fs.ModeSymlink != 0 {
			return "", fmt.Errorf("dangling symlink not allowed: %q", cur)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p, nil
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
