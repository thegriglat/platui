package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
)

// drive подаёт сообщение и выполняет команды, включая tea.Sequence (по порядку).
func drive(t *testing.T, m *model, msg tea.Msg) {
	t.Helper()
	_, cmd := m.Update(msg)
	driveCmd(t, m, cmd)
}

func driveCmd(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(200 * time.Millisecond):
		return
	}
	if msg == nil {
		return
	}
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(cmd) {
		for i := 0; i < v.Len(); i++ {
			if c, ok := v.Index(i).Interface().(tea.Cmd); ok {
				driveCmd(t, m, c)
			}
		}
		return
	}
	drive(t, m, msg)
}

func pressD(t *testing.T, m *model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		drive(t, m, keyMsg(k))
	}
}

func typeD(t *testing.T, m *model, s string) {
	for _, r := range s {
		pressD(t, m, string(r))
	}
}

func TestContractTUIPalette(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	pressD(t, m, "ctrl+p")
	typeD(t, m, "OTH-7")
	if !strings.Contains(view(m), "Открыть OTH-7") {
		t.Fatalf("нет перехода:\n%s", view(m))
	}
	pressD(t, m, "enter")
	if m.s.project.Identifier != "OTH" || e.count("WorkItems:p2") == 0 && countCalls(e, "ByKey") == 0 {
		t.Fatalf("переход по ключу: %v %v", m.s.project, e.api.Calls)
	}
	m, e = newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) {
		c.Filters = []config.Filter{{Name: "Срочные", Priorities: []string{"urgent"}}}
	})
	// открытие и действие из палитры
	pressD(t, m, "ctrl+p")
	if m.top().Scope() != "palette" {
		t.Fatalf("палитра не открылась")
	}
	if !strings.Contains(view(m), "Палитра") || !strings.Contains(view(m), "Только мои") {
		t.Fatalf("нет действий в палитре:\n%s", view(m))
	}
	typeD(t, m, "мои")
	pressD(t, m, "enter")
	if m.top().Scope() != "main" || !m.s.mine {
		t.Fatalf("действие «мои» не выполнено: scope=%s mine=%v", m.top().Scope(), m.s.mine)
	}
	pressD(t, m, "m")
	// таб
	pressD(t, m, "ctrl+p")
	typeD(t, m, "progress")
	pressD(t, m, "enter")
	if tl := m.s.tabList(); tl[m.s.tab].label != "In Progress" {
		t.Fatalf("таб: %s", tl[m.s.tab].label)
	}
	// фильтр
	pressD(t, m, "ctrl+p")
	typeD(t, m, "Срочн")
	pressD(t, m, "enter")
	if m.s.filter != 1 {
		t.Fatalf("фильтр не выбран: %d", m.s.filter)
	}
	m.s.filter = 0
	// проект
	pressD(t, m, "ctrl+p")
	typeD(t, m, "OTH Two")
	pressD(t, m, "enter")
	if m.s.project == nil || m.s.project.Identifier != "OTH" {
		t.Fatalf("проект не переключён: %+v", m.s.project)
	}
	pressD(t, m, "ctrl+p")
	typeD(t, m, "PROJ One")
	pressD(t, m, "enter")
	if m.s.project.Identifier != "PROJ" {
		t.Fatalf("проект назад")
	}
	// задача
	pressD(t, m, "ctrl+p")
	typeD(t, m, "login")
	if !strings.Contains(view(m), "Fix login page") {
		t.Fatalf("задача не найдена:\n%s", view(m))
	}
	pressD(t, m, "enter")
	if m.s.cursor != "i5" {
		t.Fatalf("курсор: %s", m.s.cursor)
	}
	if m.top().Scope() == "item" { // выбор задачи открывает карточку (PT-5c)
		pressD(t, m, "esc")
	}
	// не найдена
	pressD(t, m, "ctrl+p")
	typeD(t, m, "OTH-999")
	pressD(t, m, "enter")
	if !strings.Contains(m.s.status.text, "не найдена") {
		t.Fatalf("статус: %q", m.s.status.text)
	}
	// Esc закрывает
	pressD(t, m, "ctrl+p", "esc")
	if m.top().Scope() != "main" {
		t.Fatalf("Esc не закрыл палитру")
	}
}

func TestPaletteKeyMsgRoundTrip(t *testing.T) {
	for _, a := range registry {
		for _, k := range a.Keys {
			km, ok := keyMsgFor(k)
			if !ok || keyName(km) != k {
				t.Errorf("%s: %q → %v %v", a.ID, k, km, ok)
			}
		}
	}
}
