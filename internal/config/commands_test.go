package config

import "testing"

func TestCommandQuotedPlaceholder(t *testing.T) {
	bad := []string{`echo '{title}'`, `echo "{branch}"`, `echo "a {id} b"`, `echo 'x'"{key}"`}
	for _, c := range bad {
		if _, err := Parse([]byte("api_key: k\nworkspace: w\ncommands:\n  - name: X\n    command: |-\n      " + c + "\n")); err == nil {
			t.Errorf("%s: expected an error", c)
		}
	}
	good := []string{`echo {title}`, `echo "plain" {branch}`, `echo \"{id}`, `echo {id}"x"`}
	for _, c := range good {
		if _, err := Parse([]byte("api_key: k\nworkspace: w\ncommands:\n  - name: X\n    command: |-\n      " + c + "\n")); err != nil {
			t.Errorf("%s: %v", c, err)
		}
	}
}
