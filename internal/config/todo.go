package config

import "github.com/thegriglat/platui/internal/todo"

// Todo configures the personal todo file (C1 v6).
type Todo struct {
	Path string `yaml:"path"` // empty → todo.DefaultPath()
}

// TodoPath returns the todo file path.
func (c *Config) TodoPath() (string, error) {
	if c.Todo.Path != "" {
		return c.Todo.Path, nil
	}
	return todo.DefaultPath()
}
