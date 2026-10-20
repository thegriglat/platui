package config

import (
	"fmt"
	"strings"
)

// Command output modes.
const (
	OutputStatus = "status"
	OutputWindow = "window"
	OutputNone   = "none"
)

var commandPlaceholders = []string{"{id}", "{key}", "{branch}", "{url}", "{title}", "{mr_url}"}

// Command is a user-defined shell command (C1 v5).
type Command struct {
	Name    string `yaml:"name"`
	Key     string `yaml:"key"`
	Command string `yaml:"command"`
	Confirm bool   `yaml:"confirm"`
	Output  string `yaml:"output"`
}

func (c *Config) applyCommandsDefaults() {
	for i := range c.Commands {
		if c.Commands[i].Output == "" {
			c.Commands[i].Output = OutputStatus
		}
	}
}

func (c *Config) validateCommands() error {
	names, keys := map[string]bool{}, map[string]bool{}
	for i, cmd := range c.Commands {
		if strings.TrimSpace(cmd.Name) == "" {
			return fmt.Errorf("commands[%d]: name is required", i)
		}
		if names[cmd.Name] {
			return fmt.Errorf("commands: duplicate name %q", cmd.Name)
		}
		names[cmd.Name] = true
		if strings.TrimSpace(cmd.Command) == "" {
			return fmt.Errorf("commands %q: command is required", cmd.Name)
		}
		if err := checkTemplate(fmt.Sprintf("commands %q: command", cmd.Name), cmd.Command, commandPlaceholders, false); err != nil {
			return err
		}
		if err := checkNoQuotedPlaceholders(cmd.Name, cmd.Command); err != nil {
			return err
		}
		if !contains([]string{OutputStatus, OutputWindow, OutputNone}, cmd.Output) {
			return fmt.Errorf("commands %q: output %q must be status, window or none", cmd.Name, cmd.Output)
		}
		if cmd.Key != "" {
			if strings.TrimSpace(cmd.Key) != cmd.Key {
				return fmt.Errorf("commands %q: key %q has surrounding spaces", cmd.Name, cmd.Key)
			}
			if keys[cmd.Key] {
				return fmt.Errorf("commands: duplicate key %q", cmd.Key)
			}
			keys[cmd.Key] = true
		}
	}
	return nil
}

// CheckCommandKeys fails if a custom command key collides with a reserved (built-in) key.
func (c *Config) CheckCommandKeys(reserved []string) error {
	for _, cmd := range c.Commands {
		if cmd.Key != "" && contains(reserved, cmd.Key) {
			return fmt.Errorf("commands %q: key %q conflicts with a built-in key", cmd.Name, cmd.Key)
		}
	}
	return nil
}

// checkNoQuotedPlaceholders: values are substituted already single-quoted (RenderCommand),
// so a placeholder inside '…' or "…" breaks the escaping — a config error.
func checkNoQuotedPlaceholders(name, cmd string) error {
	var q rune
	rs := []rune(cmd)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\\' && q != '\'':
			i++
		case q == 0 && (r == '\'' || r == '"'):
			q = r
		case q != 0 && r == q:
			q = 0
		case q != 0 && r == '{':
			for _, p := range commandPlaceholders {
				if strings.HasPrefix(string(rs[i:]), p) {
					return fmt.Errorf("commands %q: placeholder %s inside quotes; values are already shell-quoted, remove the quotes", name, p)
				}
			}
		}
	}
	return nil
}
