package config

import (
	"strings"
	"testing"
)

const v5Base = "api_key: k\nworkspace: w\n"

func TestContractConfigV5Defaults(t *testing.T) {
	c, err := Parse([]byte(v5Base))
	if err != nil {
		t.Fatal(err)
	}
	if !c.MouseEnabled() || c.PreviewPercent() != 45 || len(c.Commands) != 0 {
		t.Fatalf("%+v", c.UI)
	}
}

func TestContractConfigV5Full(t *testing.T) {
	c, err := Parse([]byte(v5Base + "ui:\n  mouse: false\n  preview_width: 60\ncommands:\n  - name: IDE\n    key: ctrl+e\n    command: code {branch} {id}\n    confirm: true\n    output: window\n  - name: Open\n    command: xdg-open {url}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.MouseEnabled() || c.UI.PreviewWidth != 60 || len(c.Commands) != 2 {
		t.Fatalf("%+v", c)
	}
	if a := c.Commands[0]; a.Key != "ctrl+e" || !a.Confirm || a.Output != "window" {
		t.Fatalf("%+v", a)
	}
	if c.Commands[1].Output != "status" {
		t.Fatalf("%+v", c.Commands[1])
	}
	if err := c.CheckCommandKeys([]string{"q", "ctrl+p"}); err != nil {
		t.Fatal(err)
	}
	err = c.CheckCommandKeys([]string{"q", "ctrl+e"})
	if err == nil || !strings.Contains(err.Error(), "IDE") || !strings.Contains(err.Error(), "ctrl+e") {
		t.Fatalf("%v", err)
	}
}

func TestContractConfigV5Errors(t *testing.T) {
	cases := []struct{ name, yaml, sub string }{
		{"width low", "ui:\n  preview_width: 19\n", "preview_width"},
		{"width high", "ui:\n  preview_width: 81\n", "preview_width"},
		{"empty name", "commands:\n  - command: ls\n", "name"},
		{"empty command", "commands:\n  - name: a\n", "command"},
		{"dup name", "commands:\n  - {name: a, command: ls}\n  - {name: a, command: ls}\n", "duplicate name"},
		{"dup key", "commands:\n  - {name: a, key: x, command: ls}\n  - {name: b, key: x, command: ls}\n", "duplicate key"},
		{"bad output", "commands:\n  - {name: a, command: ls, output: foo}\n", "output"},
		{"placeholder", "commands:\n  - {name: a, command: 'ls {nope}'}\n", "{nope}"},
	}
	for _, tc := range cases {
		_, err := Parse([]byte(v5Base + tc.yaml))
		if err == nil || !strings.Contains(err.Error(), tc.sub) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}
