package text

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrEmptyEdit — после правки текст пуст.
var ErrEmptyEdit = errors.New("empty text, cancelled")

// EditSession — правка текста во внешнем редакторе через временный файл.
type EditSession struct {
	Path  string // временный файл *.md
	hints []string
}

// NewEditSession создаёт файл с initial и строками-подсказками «# …» в конце.
func NewEditSession(initial, hint string) (*EditSession, error) {
	f, err := os.CreateTemp("", "platui-*.md")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()
	s := &EditSession{Path: f.Name()}
	body := initial
	if hint != "" {
		for _, l := range strings.Split(strings.TrimRight(hint, "\n"), "\n") {
			s.hints = append(s.hints, strings.TrimRight("# "+l, " "))
		}
		body = strings.TrimRight(initial, "\n") + "\n\n" + strings.Join(s.hints, "\n") + "\n"
	}
	if _, err := f.WriteString(body); err != nil {
		os.Remove(f.Name())
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	return s, nil
}

// Command собирает команду запуска редактора (editor — config.EditorCmd()).
func (s *EditSession) Command(editor string) *exec.Cmd {
	return exec.Command("sh", "-c", editor+` "$1"`, "platui-edit", s.Path)
}

// Result читает файл, отбрасывает строки-подсказки и обрезает края.
func (s *EditSession) Result() (string, error) {
	b, err := os.ReadFile(s.Path)
	if err != nil {
		return "", fmt.Errorf("read edited file: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	isHint := map[string]bool{}
	for _, h := range s.hints {
		isHint[h] = true
	}
	for len(lines) > 0 {
		last := strings.TrimRight(lines[len(lines)-1], " \t")
		if last != "" && !isHint[last] {
			break
		}
		lines = lines[:len(lines)-1]
	}
	out := strings.TrimSpace(strings.Join(lines, "\n"))
	if out == "" {
		return "", ErrEmptyEdit
	}
	return out, nil
}

// Close удаляет временный файл.
func (s *EditSession) Close() error {
	if err := os.Remove(s.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove temp file: %w", err)
	}
	return nil
}
