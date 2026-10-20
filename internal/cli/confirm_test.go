package cli

import (
	"bytes"
	"strings"
	"testing"
)

func confirmEnv(yes, tty bool, in string) (Env, *bytes.Buffer) {
	var errb bytes.Buffer
	return Env{Stderr: &errb, Stdin: strings.NewReader(in), IsTTY: func() bool { return tty }, Yes: yes}, &errb
}

func TestContractCLIConfirmYes(t *testing.T) {
	env, errb := confirmEnv(true, false, "")
	ok, err := confirm(env, "удалить?")
	if err != nil || !ok || errb.Len() != 0 {
		t.Fatalf("ok=%v err=%v out=%q", ok, err, errb)
	}
}

func TestContractCLIConfirmNoTTY(t *testing.T) {
	env, _ := confirmEnv(false, false, "y\n")
	ok, err := confirm(env, "удалить?")
	if ok || err == nil || !strings.Contains(err.Error(), "нужно подтверждение: --yes") {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestContractCLIConfirmAnswers(t *testing.T) {
	for in, want := range map[string]bool{"y\n": true, "YES\n": true, "\n": false, "n": false, "": false, "yep\n": false} {
		env, errb := confirmEnv(false, true, in)
		ok, err := confirm(env, "удалить?")
		if err != nil || ok != want {
			t.Errorf("%q: ok=%v err=%v", in, ok, err)
		}
		if errb.String() != "удалить? [y/N] " {
			t.Errorf("prompt %q", errb)
		}
	}
}

func TestContractCLIConfirmFlag(t *testing.T) {
	f := newFx(t)
	if code := f.run("--yes", "version"); code != 0 {
		t.Fatalf("code %d", code)
	}
}
