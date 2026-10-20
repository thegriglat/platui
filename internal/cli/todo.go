package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/todo"
)

// todoPath is the todo file path; no API key needed (todo works offline).
func todoPath(s *session) (string, error) {
	if s.cfg != nil {
		return s.cfg.TodoPath()
	}
	path := s.configPath
	if path == "" {
		var err error
		if path, err = config.DefaultPath(); err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return todo.DefaultPath()
	}
	cfg, err := config.LoadNoKey(path)
	if err != nil {
		return "", err
	}
	return cfg.TodoPath()
}

// todoApply runs Load, modify, Save; retries once on ErrChanged.
func todoApply(path string, fn func(l *todo.List) error) error {
	for try := 0; ; try++ {
		l, err := todo.Load(path)
		if err != nil {
			return err
		}
		if err := fn(l); err != nil {
			return err
		}
		err = l.Save()
		if errors.Is(err, todo.ErrChanged) && try == 0 {
			continue
		}
		return err
	}
}

func todoCmd(s *session) *cobra.Command {
	ls := func(cmd *cobra.Command, _ []string) error {
		path, err := todoPath(s)
		if err != nil {
			return err
		}
		l, err := todo.Load(path)
		if err != nil {
			return err
		}
		for _, it := range l.Items() {
			mark := " "
			if it.Done {
				mark = "x"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d. [%s] %s\n", it.N, mark, it.Text)
		}
		return nil
	}
	c := &cobra.Command{Use: "todo", Short: "Personal todo list (local file, no network)", Args: argsRange(0, 0), RunE: ls}
	c.AddCommand(&cobra.Command{Use: "ls", Short: "List todo items", Args: argsRange(0, 0), RunE: ls})
	c.AddCommand(&cobra.Command{
		Use: "add <text…>", Short: "Add a todo item", Args: argsRange(1, 1<<20),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := todoPath(s)
			if err != nil {
				return err
			}
			text := strings.TrimSpace(strings.Join(args, " "))
			if text == "" {
				return usagef("todo add: empty text")
			}
			var it todo.Item
			if err := todoApply(path, func(l *todo.List) error { it = l.Add(text); return nil }); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d. [ ] %s\n", it.N, it.Text)
			return nil
		},
	})
	for _, d := range []struct {
		name  string
		done  bool
		short string
	}{{"done", true, "Mark an item done"}, {"undone", false, "Mark an item not done"}} {
		c.AddCommand(&cobra.Command{
			Use: d.name + " <N>", Short: d.short, Args: argsRange(1, 1),
			RunE: func(cmd *cobra.Command, args []string) error {
				n, err := strconv.Atoi(args[0])
				if err != nil {
					return usagef("todo %s: invalid number %q", d.name, args[0])
				}
				path, err := todoPath(s)
				if err != nil {
					return err
				}
				return todoApply(path, func(l *todo.List) error { return l.SetDone(n, d.done) })
			},
		})
	}
	return c
}
