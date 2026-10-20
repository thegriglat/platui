package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func TestContractTUIV6Links(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	var opened []string
	m.s.deps.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	// нет ссылок
	m.s.cursor = "i4"
	press(t, m, "l")
	if m.s.status.text != "ссылок нет" || m.top().Scope() != "main" {
		t.Fatalf("status=%q scope=%s", m.s.status.text, m.top().Scope())
	}
	// ссылки из описания и комментариев, без дублей
	it := api.ItemsV["p1"][0]
	it.DescHTML = `<p><a href="https://a.example/1">x</a> https://b.example/2</p>`
	api.ItemsV["p1"][0] = it
	for i := range m.s.items {
		if m.s.items[i].ID == "i1" {
			m.s.items[i].DescHTML = it.DescHTML
		}
	}
	api.CommentsV["i1"] = []plane.Comment{{ID: "c1", HTML: `<p>see https://c.example/3 and https://a.example/1</p>`}}
	m.s.cursor = "i1"
	press(t, m, "l")
	ls, ok := m.top().(*linksScreen)
	if !ok || len(ls.links) != 3 {
		t.Fatalf("top=%T %v", m.top(), m.top())
	}
	press(t, m, "down", "enter")
	if len(opened) != 1 || opened[0] != ls.links[1] {
		t.Fatalf("opened=%v links=%v", opened, ls.links)
	}
	if !strings.Contains(view(m), "https://c.example/3") {
		t.Error("нет ссылки в экране")
	}
	press(t, m, "esc")
	if m.top().Scope() != "main" {
		t.Fatal("Esc не закрыл")
	}
}

func ctrlO(t *testing.T, m *model) { feed(t, m, tea.KeyMsg{Type: tea.KeyCtrlO}) }

func TestContractTUIV6History(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	open := func(id string) {
		it, _ := m.s.itemByID(id)
		feed(t, m, openItemMsg(m.s, it))
	}
	open("i1")
	press(t, m, "esc")
	open("i2")
	open("i3")
	if m.s.openItem != "i3" {
		t.Fatal(m.s.openItem)
	}
	ctrlO(t, m)
	if m.s.openItem != "i2" || m.top().Scope() != "item" {
		t.Fatalf("после Ctrl+O: %s", m.s.openItem)
	}
	ctrlO(t, m)
	if m.s.openItem != "i1" {
		t.Fatalf("второй Ctrl+O: %s", m.s.openItem)
	}
	ctrlO(t, m)
	if m.s.openItem != "i1" || !strings.Contains(m.s.status.text, "история пуста") {
		t.Fatalf("пустая история: %s %q", m.s.openItem, m.s.status.text)
	}
	// ограничение 50
	h := &history{}
	for i := 0; i < 80; i++ {
		h.visit(itoa(i))
	}
	if len(h.stack) != historyMax || h.stack[len(h.stack)-1] != "78" {
		t.Fatalf("stack=%d", len(h.stack))
	}
}

func TestContractTUIV6Dupes(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	press(t, m, "n")
	for _, r := range "login !high @me" {
		press(t, m, string(r))
	}
	c := m.top().(*inputScreen)
	if len(c.sim) != 1 || !strings.Contains(c.sim[0], "Fix login page") {
		t.Fatalf("sim=%v", c.sim)
	}
	if !strings.Contains(view(m), "Fix login page") {
		t.Error("похожие не показаны")
	}
	// Enter по-прежнему создаёт
	n := len(api.ItemsV["p1"])
	press(t, m, "enter")
	if len(api.ItemsV["p1"]) != n+1 {
		t.Fatal("не создана")
	}
	// ↓ + Enter открывает похожую вместо создания
	press(t, m, "n")
	for _, r := range "Task number" {
		press(t, m, string(r))
	}
	c = m.top().(*inputScreen)
	if len(c.sim) != 5 {
		t.Fatalf("sim=%d (ожидалось ≤5)", len(c.sim))
	}
	gpress(t, m, "down", "enter")
	if m.top().Scope() != "item" || len(api.ItemsV["p1"]) != n+1 {
		t.Fatalf("scope=%s items=%d", m.top().Scope(), len(api.ItemsV["p1"]))
	}
}
