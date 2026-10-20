package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
)

// errNeedConfirm — нет подтверждения: stdin не терминал, флаг --yes не задан.
var errNeedConfirm = errors.New("нужно подтверждение: --yes")

// confirm — правило подтверждения деструктивных действий (C10 v2).
// --yes → true без вопроса; без TTY — ошибка; иначе вопрос «question [y/N] »
// в Stderr и ответ из Stdin (только y/yes без учёта регистра → true).
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
