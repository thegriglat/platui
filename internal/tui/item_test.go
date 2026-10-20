package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/text"
)

func openCard(t *testing.T, m *model, id string) {
	t.Helper()
	it, ok := m.s.itemByID(id)
	if !ok {
		t.Fatalf("нет задачи %s", id)
	}
	feed(t, m, pushMsgOf(t, openItem(m.s, it)))
	if m.top().Scope() != "item" {
		t.Fatalf("scope %s", m.top().Scope())
	}
}

func pushMsgOf(t *testing.T, c tea.Cmd) tea.Msg {
	t.Helper()
	return c()
}

// settle прогоняет два тика превью: комментарии запрашиваются, когда выбор стабилен.
func settle(t *testing.T, m *model) {
	t.Helper()
	feed(t, m, previewTickMsg{})
	feed(t, m, previewTickMsg{})
}

func resize(t *testing.T, m *model, w, h int) { feed(t, m, tea.WindowSizeMsg{Width: w, Height: h}) }

func widths(t *testing.T, v string, w int) {
	t.Helper()
	for _, l := range strings.Split(v, "\n") {
		if lipgloss.Width(l) > w {
			t.Fatalf("строка шире %d: %q", w, l)
		}
	}
}

func TestContractTUIItemRender(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	api.CommentsV["i1"] = []plane.Comment{{ID: "c1", Author: "u-an", HTML: "<p>hello <b>comment</b></p>", CreatedAt: t0}}
	for _, w := range []int{40, 80, 120} {
		resize(t, m, w, 30)
		m.s.cursor = "i1"
		settle(t, m)
		v := view(m)
		widths(t, v, w)
		if w >= wideWidth && (!strings.Contains(v, "desc") || !strings.Contains(v, "Anna") || !strings.Contains(v, "hello")) {
			t.Fatalf("превью %d: %s", w, v)
		}
	}
	resize(t, m, 40, 30)
	openCard(t, m, "i1")
	settle(t, m)
	v := view(m)
	widths(t, v, 40)
	for _, want := range []string{"PROJ-1", "Task number 1", "Backlog", "low", "Me", "desc", "Anna", "hello", "comment"} {
		if !strings.Contains(v, want) {
			t.Errorf("нет %q:\n%s", want, v)
		}
	}
	press(t, m, "down", "pgdown", "end", "home", "up", "pgup")
	press(t, m, "esc")
	if m.top().Scope() != "main" || m.s.openItem != "" {
		t.Fatal("esc не закрыл карточку")
	}
}

func TestContractTUIItemNoColor(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(old)
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Theme.NoColor = true })
	openCard(t, m, "i1")
	settle(t, m)
	if v := view(m); strings.Contains(v, "\x1b[3") || strings.Contains(v, "\x1b[4") || !strings.Contains(v, "desc") {
		t.Fatalf("цвета при NoColor: %q", v)
	}
	m2, _, _, _ := newTestModel(t)
	openCard(t, m2, "i1")
	if !strings.Contains(view(m2), "\x1b[") {
		t.Fatal("нет цвета")
	}
}

func TestContractTUIItemHighlight(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(old)
	m, _, _, _ := newTestModel(t)
	m.s.setQuery("login")
	m.s.cursor = "i5"
	v := view(m)
	if !strings.Contains(v, "\x1b[1;4;38;2") || strings.Contains(v, "login page") && !strings.Contains(v, "\x1b[0m page") {
		t.Fatalf("совпадения не подсвечены: %q", v)
	}
	if got := highlight(m.s, "abc", nil); got != "abc" {
		t.Fatal(got)
	}
}

func (m *model) patchOf(t *testing.T, id string) plane.WorkItem {
	t.Helper()
	it, ok := m.s.itemByID(id)
	if !ok {
		t.Fatal(id)
	}
	return it
}

func TestContractTUIItemPatches(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	m.s.cursor = "i1"
	pressD(t, m, "s")
	if m.top().Scope() != "dialog" {
		t.Fatal("нет пикера состояния")
	}
	pressD(t, m, "down", "enter")
	if m.top().Scope() != "main" || m.patchOf(t, "i1").StateID != "s2" {
		t.Fatalf("состояние: %+v %s", m.patchOf(t, "i1"), view(m))
	}
	pressD(t, m, "p", "esc")
	if m.top().Scope() != "main" || m.patchOf(t, "i1").Priority != "low" {
		t.Fatal("esc в пикере")
	}
	pressD(t, m, "p", "down", "enter")
	if m.patchOf(t, "i1").Priority != "high" {
		t.Fatalf("приоритет: %+v", m.patchOf(t, "i1"))
	}
	m.s.cursor = "i2"
	pressD(t, m, "a")
	if got := m.patchOf(t, "i2").AssigneeIDs; len(got) != 1 || got[0] != "u-me" {
		t.Fatalf("назначение: %v", got)
	}
	n := len(api.Calls)
	pressD(t, m, "a")
	if len(api.Calls) != n || !strings.Contains(m.s.status.text, "уже") {
		t.Fatalf("повторное назначение: %q", m.s.status.text)
	}
	calls := strings.Join(api.Calls, " ")
	if strings.Count(calls, "UpdateWorkItem:p1:i1") != 2 || !strings.Contains(calls, "UpdateWorkItem:p1:i2") {
		t.Fatalf("calls: %v", api.Calls)
	}
	// из полной карточки
	openCard(t, m, "i3")
	pressD(t, m, "p", "enter")
	if m.patchOf(t, "i3").Priority != "urgent" || m.top().Scope() != "item" {
		t.Fatalf("из карточки: %+v", m.patchOf(t, "i3"))
	}
	// метки: пикер общего использования
	var got plane.Label
	pushed := pickLabel(m.s, func(l plane.Label) { got = l })
	feed(t, m, pushScreenMsg{pushed})
	pressD(t, m, "down", "enter")
	if got.ID != "l2" {
		t.Fatalf("метка: %+v", got)
	}
	m.s.cursor = ""
	pressD(t, m, "esc", "s")
	if !m.s.status.isErr {
		t.Fatal("нет ошибки без задачи")
	}
}

func TestContractTUIItemOpenCopy(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	var opened, copied []string
	m.s.deps.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	m.s.deps.Clipboard = func(s string) error { copied = append(copied, s); return nil }
	m.s.cursor = "i1"
	pressD(t, m, "O")
	if len(opened) != 1 || !strings.HasSuffix(opened[0], "PROJ-1/") {
		t.Fatalf("OpenURL: %v", opened)
	}
	pressD(t, m, "y")
	if m.top().Scope() != "dialog" || !strings.Contains(view(m), "Markdown") {
		t.Fatal("нет меню копирования")
	}
	pressD(t, m, "u")
	if len(copied) != 1 || copied[0] != opened[0] || m.top().Scope() != "main" {
		t.Fatalf("u: %v", copied)
	}
	pressD(t, m, "y", "i")
	pressD(t, m, "y", "t")
	pressD(t, m, "y", "m")
	if copied[1] != "PROJ-1" || copied[2] != "Task number 1" || !strings.HasPrefix(copied[3], "[PROJ-1](") {
		t.Fatalf("copied: %v", copied)
	}
	pressD(t, m, "y", "b")
	if len(copied) != 5 || copied[4] != "PROJ-1-task-number-1" {
		t.Fatalf("ветка: %v %+v", copied, m.s.status)
	}
	m.s.deps.OpenURL = func(string) error { return errTest }
	pressD(t, m, "O")
	if !m.s.status.isErr {
		t.Fatal("ошибка браузера")
	}
}

var errTest = &testErr{}

type testErr struct{}

func (*testErr) Error() string { return "boom" }

// syncExec подменяет ExecProcess: команда редактора выполняется сразу.
func syncExec(t *testing.T) {
	t.Helper()
	old := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		err := sess.Command(editor).Run()
		return func() tea.Msg { return fn(err) }
	}
	t.Cleanup(func() { runEditor = old })
}

func TestContractTUIItemEdit(t *testing.T) {
	syncExec(t)
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Editor = `printf 'new **text**\n' >` })
	m.s.cursor = "i1"
	press(t, m, "e")
	if d := m.patchOf(t, "i1").DescHTML; !strings.Contains(d, "<strong>text</strong>") {
		t.Fatalf("описание: %q", d)
	}
	press(t, m, "c")
	cs := e.api.CommentsV["i1"]
	if len(cs) != 1 || !strings.Contains(cs[0].HTML, "new") || e.count("CreateComment:p1:i1") != 1 {
		t.Fatalf("комментарий: %+v", cs)
	}
	// пустая правка — отмена
	m.s.deps.Config.Editor = `true`
	press(t, m, "c")
	if e.count("CreateComment") != 1 || !strings.Contains(m.s.status.text, "отмен") {
		t.Fatalf("пустой: %q", m.s.status.text)
	}
	// падение редактора
	m.s.deps.Config.Editor = `false`
	press(t, m, "e")
	if !m.s.status.isErr {
		t.Fatal("ошибка редактора")
	}
	// из полной карточки
	m.s.deps.Config.Editor = `printf 'second\n' >`
	openCard(t, m, "i2")
	press(t, m, "c")
	if e.count("CreateComment:p1:i2") != 1 {
		t.Fatal("комментарий из карточки")
	}
}

func TestContractTUIItemTimer(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	m.s.cursor = "i1"
	press(t, m, "t")
	if _, ok := m.s.timerOf("i1"); !ok || !strings.Contains(m.s.status.text, "запущен") {
		t.Fatalf("старт: %q", m.s.status.text)
	}
	if v := view(m); !strings.Contains(v, "⏱") {
		t.Fatalf("нет таймера в превью: %s", v)
	}
	press(t, m, "t")
	if len(api.CommentsV["i1"]) != 1 || !strings.Contains(api.CommentsV["i1"][0].HTML, "Затрачено") {
		t.Fatalf("комментарий о времени: %+v", api.CommentsV["i1"])
	}
	if _, ok := m.s.timerOf("i1"); ok || !strings.Contains(m.s.status.text, "комментарий") {
		t.Fatalf("стоп: %q", m.s.status.text)
	}
	api.Err["CreateComment"] = errTest
	press(t, m, "t", "t")
	if _, ok := m.s.timerOf("i1"); !ok || !m.s.status.isErr {
		t.Fatalf("таймер должен остаться: %q", m.s.status.text)
	}
}

func TestContractTUIItemLoad(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	m.s.cursor = "i3"
	feed(t, m, previewTickMsg{})
	if e.count("Comments") != 0 {
		t.Fatal("загрузка без стабильного выбора")
	}
	feed(t, m, previewTickMsg{})
	feed(t, m, previewTickMsg{})
	if e.count("Comments:p1:i3") != 1 {
		t.Fatalf("calls: %v", e.api.Calls)
	}
	openCard(t, m, "i3")
	settle(t, m)
	if e.count("Comments:p1:i3") != 2 || e.count("WorkItem:p1:i3") != 1 {
		t.Fatalf("calls: %v", e.api.Calls)
	}
}
