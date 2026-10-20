package cli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/thegriglat/platui/internal/forge"
)

func TestContractCLIMR(t *testing.T) {
	f := newFx(t)
	var opened []string
	old := openURL
	openURL = func(u string) error { opened = append(opened, u); return nil }
	t.Cleanup(func() { openURL = old })

	// нет MR: код 1
	if c := f.run("mr", "PROJ-1"); c != 1 || !strings.Contains(f.err.String(), "MR не найдены") {
		t.Fatalf("none: %d %s", c, f.err.String())
	}
	f.forge.MRs = []forge.MR{{IID: 7, State: "opened", Title: "Fix login", URL: "https://gl/mr/7"}}
	if c := f.run("mr", "PROJ-1"); c != 0 || strings.TrimSpace(f.out.String()) != "!7  opened  Fix login  https://gl/mr/7" {
		t.Fatalf("one: %d %q %s", c, f.out.String(), f.err.String())
	}
	if len(opened) != 0 {
		t.Fatal("opened without --open")
	}
	if c := f.run("mr", "PROJ-1", "--open"); c != 0 || len(opened) != 1 || opened[0] != "https://gl/mr/7" {
		t.Fatalf("open: %d %v", c, opened)
	}
	// JSON
	if c := f.run("mr", "1", "--json"); c != 0 {
		t.Fatal(c)
	}
	var got []mrOut
	if err := json.Unmarshal(f.out.Bytes(), &got); err != nil || len(got) != 1 || got[0].IID != 7 || got[0].URL != "https://gl/mr/7" {
		t.Fatalf("json: %v %+v", err, got)
	}
	// несколько: список, без открытия
	f.forge.MRs = append(f.forge.MRs, forge.MR{IID: 8, State: "merged", Title: "Older", URL: "https://gl/mr/8"})
	opened = nil
	if c := f.run("mr", "PROJ-1", "--open"); c != 0 || len(opened) != 0 || strings.Count(f.out.String(), "\n") != 2 {
		t.Fatalf("many: %d %q %v", c, f.out.String(), opened)
	}
	// без ref: текущая ветка
	f.git.CurrentV = "PROJ-1-fix-login"
	if c := f.run("mr"); c != 0 || !strings.Contains(f.out.String(), "!7") {
		t.Fatalf("current: %d %s %s", c, f.out.String(), f.err.String())
	}
	f.git.CurrentV = "main"
	if c := f.run("mr"); c != 1 {
		t.Fatalf("no item: %d", c)
	}
	if c := f.run("mr", "a", "b"); c != 2 {
		t.Fatal(c)
	}
	if c := f.run("mr", "bad ref"); c != 2 {
		t.Fatalf("bad ref: %d", c)
	}
	// ошибка glab
	f.forge.Err = errors.New("glab down")
	if c := f.run("mr", "PROJ-1"); c != 1 || !strings.Contains(f.err.String(), "glab down") {
		t.Fatalf("err: %d %s", c, f.err.String())
	}
	// ошибка открытия
	f.forge.Err = nil
	f.forge.MRs = f.forge.MRs[:1]
	openURL = func(string) error { return errors.New("no browser") }
	if c := f.run("mr", "PROJ-1", "--open"); c != 1 {
		t.Fatal(c)
	}
}
