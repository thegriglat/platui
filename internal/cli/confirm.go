package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
)

// errNeedConfirm means no confirmation: stdin is not a terminal and --yes is not set.
var errNeedConfirm = errors.New("confirmation required: --yes")

// confirm applies the confirmation rule for destructive actions (C10 v2).
// --yes → true without asking; no TTY → error; otherwise it prints
// "question [y/N] " to Stderr and reads the answer from Stdin (only y/yes,
// case-insensitive, → true).
func confirm(env Env, question string) (bool, error) {
	if env.Yes {
		return true, nil
	}
	if env.IsTTY == nil || !env.IsTTY() {
		return false, errNeedConfirm
	}
	fmt.Fprintf(env.Stderr, "%s [y/N] ", question)
	if env.Stdin == nil {
		return false, nil
	}
	line, _ := bufio.NewReader(env.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}
	return false, nil
}
