package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/forge/forgetest"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/text"
)

// gitModel — образцовая модель с MR-фейком, подключённым к app (testutil его не подключает).
func gitModel(t *testing.T) (*model, *testEnv, *forgetest.Fake, *[]string) {
	t.Helper()
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	fg := &forgetest.Fake{URL: "https://x/mr/1"}
	m.s.deps.App = app.New(app.Deps{API: e.api, Git: e.git, Forge: fg, Store: cache.NewMem(func() time.Time { return t0 }),
		Config: e.cfg, Now: func() time.Time { return t0 }})
	var opened []string
	m.s.deps.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	view(m) // выравнивает курсор
	return m, e, fg, &opened
}

func has(calls []string, prefix string) bool {
	return slices.ContainsFunc(calls, func(c string) bool { return strings.HasPrefix(c, prefix) })
}

func TestContractTUIGit(t *testing.T) {
	t.Run("n", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		old := runEditor
		defer func() { runEditor = old }()
		runEditor = func(sess *text.EditSession, _ string, fn tea.ExecCallback) tea.Cmd {
			return func() tea.Msg {
				_ = os.WriteFile(sess.Path, []byte("Описание **жирное**\n"), 0o600)
				return fn(nil)
			}
		}
		gpress(t, m, "n")
		if m.top().Scope() != "dialog" {
			t.Fatal("нет диалога")
		}
		gpress(t, m, "Brand new !high @Anna #Bug", "ctrl+e")
		if !strings.Contains(view(m), "описание: Описание") {
			t.Errorf("описание не показано:\n%s", view(m))
		}
		gpress(t, m, "enter")
		if !has(e.api.Calls, "CreateWorkItem:p1") || len(m.stack) != 1 || !strings.Contains(m.s.status.text, "создана PROJ-13") {
			t.Fatalf("calls=%v stack=%d status=%q", e.api.Calls, len(m.stack), m.s.status.text)
		}
		if it, ok := m.s.itemByID(m.s.cursor); !ok || it.Name != "Brand new" || it.Priority != "high" || it.DescHTML == "" {
			t.Errorf("cursor item %+v", it)
		}
		if it, _ := m.s.itemByID(m.s.cursor); !slices.Equal(it.AssigneeIDs, []string{"u-an"}) || !slices.Equal(it.LabelIDs, []string{"l1"}) {
			t.Errorf("исполнитель/метка: %+v", it)
		}
		// неизвестная метка: ошибка в диалоге, без вызова
		before := e.count("CreateWorkItem")
		gpress(t, m, "n", "x #nolabel", "enter")
		if len(m.stack) != 2 || !m.s.status.isErr || !strings.Contains(m.s.status.text, "nolabel") || e.count("CreateWorkItem") != before {
			t.Errorf("#unknown: stack=%d status=%+v", len(m.stack), m.s.status)
		}
		gpress(t, m, "esc")
		// ошибка разбора: диалог остаётся
		gpress(t, m, "n", "!bad", "enter")
		if len(m.stack) != 2 || !m.s.status.isErr {
			t.Errorf("stack=%d status=%+v", len(m.stack), m.s.status)
		}
		gpress(t, m, "esc")
		if len(m.stack) != 1 {
			t.Error("esc не закрыл")
		}
	})
	t.Run("b", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		it, _ := m.target()
		name, _ := m.s.deps.App.BranchName(*m.s.project, it)
		gpress(t, m, "b")
		if !strings.Contains(view(m), name) {
			t.Fatalf("нет имени %q:\n%s", name, view(m))
		}
		gpress(t, m, "enter")
		if !has(e.git.Calls, "Create"+name) || len(m.stack) != 1 || !strings.Contains(m.s.status.text, name) {
			t.Fatalf("git=%v status=%q", e.git.Calls, m.s.status.text)
		}
		if e.count("UpdateWorkItem") != 0 {
			t.Error("b изменила задачу в Plane")
		}
	})
	t.Run("b существующая", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		it, _ := m.target()
		name, _ := m.s.deps.App.BranchName(*m.s.project, it)
		e.git.BranchesV[name] = gitbranch.BranchStatus{Name: name, Local: true}
		gpress(t, m, "b", "enter")
		if m.top().Scope() != "dialog" || !strings.Contains(view(m), "переключиться") || has(e.git.Calls, "Switch") {
			t.Fatalf("нет вопроса или Switch раньше y: %v\n%s", e.git.Calls, view(m))
		}
		gpress(t, m, "enter") // Enter = нет
		if has(e.git.Calls, "Switch") || len(m.stack) != 1 || m.s.status.text != "отменено" {
			t.Fatalf("enter: %v stack=%d status=%q", e.git.Calls, len(m.stack), m.s.status.text)
		}
		gpress(t, m, "b", "enter", "n")
		if has(e.git.Calls, "Switch") || len(m.stack) != 1 {
			t.Fatalf("n: %v stack=%d", e.git.Calls, len(m.stack))
		}
		gpress(t, m, "b", "enter", "esc")
		if has(e.git.Calls, "Switch") || len(m.stack) != 1 {
			t.Fatalf("esc: %v", e.git.Calls)
		}
		gpress(t, m, "b", "enter", "Y")
		if !has(e.git.Calls, "Switch"+name) || len(m.stack) != 1 || e.count("UpdateWorkItem") != 0 {
			t.Errorf("y: %v stack=%d", e.git.Calls, len(m.stack))
		}
	})
	t.Run("b без репозитория", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		m.s.deps.App = app.New(app.Deps{API: e.api, Forge: &forgetest.Fake{}, Config: e.cfg})
		gpress(t, m, "b", "enter")
		if !m.s.status.isErr || !strings.Contains(m.s.status.text, "git") {
			t.Errorf("status=%+v", m.s.status)
		}
	})
	t.Run("S", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		it, _ := m.target()
		name, _ := m.s.deps.App.BranchName(*m.s.project, it)
		gpress(t, m, "S")
		if !has(e.git.Calls, "Create"+name) || e.count("UpdateWorkItem") != 1 {
			t.Fatalf("git=%v api=%v", e.git.Calls, e.api.Calls)
		}
		if !strings.Contains(m.s.status.text, "создана ветка") || !strings.Contains(m.s.status.text, "In Progress") {
			t.Errorf("status=%q", m.s.status.text)
		}
	})
	t.Run("W", func(t *testing.T) {
		m, e, fg, _ := gitModel(t)
		e.git.CurrentV = "PROJ-1-task"
		gpress(t, m, "W")
		if len(e.git.Pushed) != 1 || len(fg.Reqs) != 1 || fg.Reqs[0].Head != "PROJ-1-task" {
			t.Fatalf("pushed=%v reqs=%v", e.git.Pushed, fg.Reqs)
		}
		if !strings.Contains(m.s.status.text, "https://x/mr/1") {
			t.Errorf("status=%q", m.s.status.text)
		}
	})
	t.Run("M", func(t *testing.T) {
		m, _, fg, opened := gitModel(t)
		gpress(t, m, "M")
		if !strings.Contains(m.s.status.text, "MR не найдены") {
			t.Errorf("status=%q", m.s.status.text)
		}
		fg.MRs = []forge.MR{{IID: 1, Title: "one", URL: "https://x/mr/1", State: "opened"}}
		gpress(t, m, "M")
		if len(*opened) != 1 || (*opened)[0] != "https://x/mr/1" {
			t.Fatalf("opened=%v", *opened)
		}
		fg.MRs = append(fg.MRs, forge.MR{IID: 2, Title: "two", URL: "https://x/mr/2", State: "merged"})
		gpress(t, m, "M")
		if m.top().Scope() != "dialog" || !strings.Contains(view(m), "two") {
			t.Fatalf("нет выбора:\n%s", view(m))
		}
		gpress(t, m, "down", "enter")
		if len(*opened) != 2 || (*opened)[1] != "https://x/mr/2" || len(m.stack) != 1 {
			t.Errorf("opened=%v stack=%d", *opened, len(m.stack))
		}
		fg.Err = forge.ErrNotFound
		gpress(t, m, "M")
		if !m.s.status.isErr || !strings.Contains(m.s.status.text, "glab") {
			t.Errorf("status=%+v", m.s.status)
		}
	})
	t.Run("X", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		gpress(t, m, "X")
		if m.s.status.text != "нет веток завершённых задач" {
			t.Fatalf("status=%q", m.s.status.text)
		}
		e.git.BranchesV["PROJ-9-task-number-9"] = gitbranch.BranchStatus{Name: "PROJ-9-task-number-9", Local: true} // i9 — Done
		e.git.BranchesV["PROJ-6-task-number-6"] = gitbranch.BranchStatus{Name: "PROJ-6-task-number-6", Local: true} // In Progress
		gpress(t, m, "X")
		v := view(m)
		if m.top().Scope() != "dialog" || !strings.Contains(v, "PROJ-9-task-number-9") || strings.Contains(v, "PROJ-6-") {
			t.Fatalf("список:\n%s", v)
		}
		gpress(t, m, "enter") // ничего не отмечено
		if has(e.git.Calls, "Delete") || len(m.stack) != 2 {
			t.Fatalf("удаление без отметок: %v stack=%d", e.git.Calls, len(m.stack))
		}
		gpress(t, m, "j", "k", "q") // vim-клавиши не работают
		if len(m.stack) != 2 || m.top().(*staleList).cur != 0 || len(m.top().(*staleList).marked) != 0 {
			t.Fatalf("j/k/q: stack=%d", len(m.stack))
		}
		gpress(t, m, "esc")
		gpress(t, m, "X", " ", "enter")
		if !strings.Contains(view(m), "удалить 1 веток?") || has(e.git.Calls, "Delete") {
			t.Fatalf("нет подтверждения:\n%s", view(m))
		}
		gpress(t, m, "enter")
		if has(e.git.Calls, "Delete") || m.s.status.text != "отменено" {
			t.Error("Delete после Enter")
		}
		gpress(t, m, "esc")
		// ErrNotMerged → вопрос о force
		e.git.Err["Delete"] = gitbranch.ErrNotMerged
		gpress(t, m, "X", " ", "enter", "y")
		if !has(e.git.Calls, "DeletePROJ-9-task-number-9false") || !strings.Contains(view(m), "принудительно") {
			t.Fatalf("нет вопроса force: %v\n%s", e.git.Calls, view(m))
		}
		delete(e.git.Err, "Delete")
		gpress(t, m, "y")
		if !has(e.git.Calls, "DeletePROJ-9-task-number-9true") || !strings.Contains(m.s.status.text, "удалено веток: 1") {
			t.Errorf("git=%v status=%q", e.git.Calls, m.s.status.text)
		}
		// обычное удаление отмеченных
		e.git.BranchesV["PROJ-9-task-number-9"] = gitbranch.BranchStatus{Name: "PROJ-9-task-number-9", Local: true}
		gpress(t, m, "esc", "X", " ", "enter", "y")
		if !strings.Contains(m.s.status.text, "удалено веток: 1") {
			t.Errorf("status=%q", m.s.status.text)
		}
		// список после удаления без удалённых веток, отметки сняты
		if sl, ok := m.top().(*staleList); !ok || len(sl.list) != 0 || len(sl.marked) != 0 {
			t.Errorf("список после удаления: %T", m.top())
		}
		n := len(e.git.Calls)
		gpress(t, m, "enter")
		if len(e.git.Calls) != n {
			t.Error("повторное удаление")
		}
	})
}

// gdrain выполняет команду, разворачивая Batch и Sequence (в порядке); команда, не вернувшаяся за 1 с, — ошибка теста (молча не пропускается).
func gdrain(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(time.Second):
		t.Fatal("gdrain: команда не вернулась за 1 с")
	}
	if msg == nil {
		return
	}
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(cmd) {
		for i := 0; i < v.Len(); i++ { // BatchMsg и sequenceMsg
			gdrain(t, m, v.Index(i).Interface().(tea.Cmd))
		}
		return
	}
	_, next := m.Update(msg)
	gdrain(t, m, next)
}

// gpress — как press, но с поддержкой tea.Sequence.
func gpress(t *testing.T, m *model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		_, cmd := m.Update(keyMsg(k))
		gdrain(t, m, cmd)
	}
}
