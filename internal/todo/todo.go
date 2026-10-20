// Package todo is a personal todo list stored as a Markdown checkbox file.
// Lines that are not checkbox items are kept untouched.
package todo

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	itemRe = regexp.MustCompile(`^(\s*)([-*+]) \[( |x|X)\] (.*)$`)
	keyRe  = regexp.MustCompile(`(?:^|\W)([A-Z][A-Z0-9]*-[0-9]+)\b`)
)

// ErrChanged means the file was modified on disk since Load.
var ErrChanged = errors.New("todo file changed on disk")

// ErrNoItem means the item number is out of range.
var ErrNoItem = errors.New("no such todo item")

// Item is a checkbox item; N counts from 1 in file order.
type Item struct {
	N    int
	Text string
	Done bool
}

// List holds the file lines, item indexes and the content fingerprint at Load.
type List struct {
	path     string
	lines    []string
	idx      []int // line index of each item
	trailing bool  // file ends with a newline
	sum      [sha256.Size]byte
	exists   bool
}

// Keys returns task references (PROJ-42) in order, without duplicates.
func Keys(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range keyRe.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// DefaultPath returns $XDG_DATA_HOME/platui/todo.md or ~/.local/share/platui/todo.md.
func DefaultPath() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "platui", "todo.md"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("todo: cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "platui", "todo.md"), nil
}

func read(path string) ([]byte, bool, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return b, err == nil, err
}

// Load reads the file; a missing file gives an empty list.
func Load(path string) (*List, error) {
	b, exists, err := read(path)
	if err != nil {
		return nil, fmt.Errorf("todo: read %s: %w", path, err)
	}
	l := &List{path: path, exists: exists, sum: sha256.Sum256(b), trailing: true}
	s := string(b)
	if s != "" {
		l.trailing = strings.HasSuffix(s, "\n")
		l.lines = strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	}
	l.reindex()
	return l, nil
}

func (l *List) reindex() {
	l.idx = l.idx[:0]
	for i, ln := range l.lines {
		if itemRe.MatchString(ln) {
			l.idx = append(l.idx, i)
		}
	}
}

func parse(line string, n int) Item {
	m := itemRe.FindStringSubmatch(line)
	return Item{N: n, Text: m[4], Done: m[3] != " "}
}

// Items returns the items in file order.
func (l *List) Items() []Item {
	out := make([]Item, len(l.idx))
	for i, li := range l.idx {
		out[i] = parse(l.lines[li], i+1)
	}
	return out
}

func (l *List) line(n int) (int, error) {
	if n < 1 || n > len(l.idx) {
		return 0, fmt.Errorf("%w: %d", ErrNoItem, n)
	}
	return l.idx[n-1], nil
}

// Add appends "- [ ] text" after the last item (or at the end of the file).
func (l *List) Add(text string) Item {
	at := len(l.lines)
	if k := len(l.idx); k > 0 {
		at = l.idx[k-1] + 1
	}
	l.lines = append(l.lines, "")
	copy(l.lines[at+1:], l.lines[at:])
	l.lines[at] = "- [ ] " + oneLine(text)
	l.reindex()
	return Item{N: len(l.idx), Text: oneLine(text)}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ").Replace(s)), " ")
}

func (l *List) rewrite(n int, mark, text *string) error {
	i, err := l.line(n)
	if err != nil {
		return err
	}
	m := itemRe.FindStringSubmatch(l.lines[i])
	if mark != nil {
		m[3] = *mark
	}
	if text != nil {
		m[4] = *text
	}
	l.lines[i] = m[1] + m[2] + " [" + m[3] + "] " + m[4]
	return nil
}

// SetDone marks the item done or not, leaving the rest of the line intact.
func (l *List) SetDone(n int, done bool) error {
	mark := " "
	if done {
		mark = "x"
	}
	return l.rewrite(n, &mark, nil)
}

// Edit replaces the item text.
func (l *List) Edit(n int, text string) error {
	t := oneLine(text)
	return l.rewrite(n, nil, &t)
}

// Remove deletes the item; the caller asks for confirmation.
func (l *List) Remove(n int) error {
	i, err := l.line(n)
	if err != nil {
		return err
	}
	l.lines = append(l.lines[:i], l.lines[i+1:]...)
	l.reindex()
	return nil
}

// Move puts item n at the place of item to; other lines stay where they are.
func (l *List) Move(n, to int) error {
	if _, err := l.line(n); err != nil {
		return err
	}
	if _, err := l.line(to); err != nil {
		return err
	}
	vals := make([]string, len(l.idx))
	for i, li := range l.idx {
		vals[i] = l.lines[li]
	}
	v := vals[n-1]
	vals = append(vals[:n-1], vals[n:]...)
	vals = append(vals[:to-1], append([]string{v}, vals[to-1:]...)...)
	for i, li := range l.idx {
		l.lines[li] = vals[i]
	}
	return nil
}

// Save writes the file atomically unless it changed since Load.
func (l *List) Save() error {
	b, exists, err := read(l.path)
	if err != nil {
		return fmt.Errorf("todo: read %s: %w", l.path, err)
	}
	if exists != l.exists || sha256.Sum256(b) != l.sum {
		return ErrChanged
	}
	out := strings.Join(l.lines, "\n")
	if len(l.lines) > 0 && l.trailing {
		out += "\n"
	}
	if err := writeAtomic(l.path, []byte(out)); err != nil {
		return fmt.Errorf("todo: write %s: %w", l.path, err)
	}
	l.exists, l.sum = true, sha256.Sum256([]byte(out))
	return nil
}
