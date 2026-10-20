package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/text"

	tea "github.com/charmbracelet/bubbletea"
)

// C4 v8: старт без сети, пометка «офлайн», восстановление, сохранение текста $EDITOR.
func TestContractTUIV8OfflineAndRecovery(t *testing.T) {
	_, e := flEnvNew(t, 0, nil)
	e.failAll(flNet)
	m := flModel(t, e.api, e.store, 0, nil)
	want := "офлайн, данные от " + t0.Format("15:04")
	if len(m.s.items) != 5 || !strings.Contains(m.s.status.text, want) || !strings.Contains(view(m), want) {
		t.Fatalf("задач %d, статус %+v", len(m.s.items), m.s.status)
	}
	// не-ошибочный статус не вытесняет пометку
	m.s.setStatus("отменено", false)
	if v := view(m); !strings.Contains(v, want) || !strings.Contains(v, "отменено") {
		t.Fatalf("пометка пропала:\n%s", v)
	}
	// сеть вернулась: R снимает офлайн, сообщение «обновлено» (reload) затем фон — «связь восстановлена» один раз
	e.failAll(nil)
	press(t, m, "R")
	if !strings.Contains(m.s.status.text, "связь восстановлена") || !strings.Contains(m.s.status.text, "обновлено") {
		t.Fatalf("R: %q", m.s.status.text)
	}
	if m.s.net.offline || m.s.status.isErr || strings.Contains(view(m), "офлайн") {
		t.Fatalf("офлайн не снят: %+v", m.s.status)
	}
	e.setErr("WorkItems", flNet)
	feed(t, m, pollMsg{})
	e.setErr("WorkItems", nil)
	feed(t, m, pollMsg{})
	if m.s.status.text != "связь восстановлена" || m.s.status.isErr {
		t.Fatalf("статус %+v", m.s.status)
	}
	feed(t, m, pollMsg{})
	if m.s.status.text != "связь восстановлена" {
		t.Fatal("повторная проверка")
	}
	m.s.setStatus("x", false)
	feed(t, m, pollMsg{})
	if m.s.status.text != "x" {
		t.Fatalf("«связь восстановлена» повторилась: %q", m.s.status.text)
	}
}

func TestContractTUIV8EditorDraftRetry(t *testing.T) {
	syncExec(t)
	var path string
	inner := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		path = sess.Path
		return inner(sess, editor, fn)
	}
	m, e := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `sed -i '1i precious text'` })
	t.Cleanup(func() {
		if path != "" {
			os.Remove(path)
		}
	})
	e.setErr("CreateComment", flNet)
	m.s.cursor = "i1"
	press(t, m, "c")
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "текст сохранён: "+path) {
		t.Fatalf("статус %+v", m.s.status)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "precious text") {
		t.Fatalf("файл: %v", err)
	}
	// повтор: предложение использовать сохранённый текст, y — отправка без редактора
	e.setErr("CreateComment", nil)
	press(t, m, "c")
	if _, ok := m.top().(*confirmScreen); !ok {
		t.Fatalf("нет предложения: %T", m.top())
	}
	pressD(t, m, "y")
	if m.s.status.isErr || !strings.Contains(m.s.status.text, "комментарий добавлен") {
		t.Fatalf("статус %+v", m.s.status)
	}
	e.api.Mu.Lock()
	cs := e.api.CommentsV["i1"]
	e.api.Mu.Unlock()
	if len(cs) != 1 || strings.Contains(cs[0].HTML, "Комментарий к") || !strings.Contains(cs[0].HTML, "precious text") {
		t.Fatalf("в Plane ушли подсказки или не тот текст: %+v", cs)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("файл не удалён после успеха")
	}
}

// N — файл черновика удалён, редактор открывается с описанием из Plane.
func TestContractTUIV8EditorDraftDecline(t *testing.T) {
	syncExec(t)
	var paths []string
	inner := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		paths = append(paths, sess.Path)
		return inner(sess, editor, fn)
	}
	m, e := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `sed -i '1i keep me'` })
	t.Cleanup(func() {
		for _, p := range paths {
			os.Remove(p)
		}
	})
	e.api.Mu.Lock()
	e.api.ItemsV["p1"][0].DescHTML = "<p>from plane</p>"
	e.api.Mu.Unlock()
	m.s.items[0].DescHTML = "<p>from plane</p>"
	e.setErr("UpdateWorkItem", flNet)
	m.s.cursor = "i1"
	press(t, m, "e")
	e.setErr("UpdateWorkItem", nil)
	press(t, m, "e")
	pressD(t, m, "n")
	if len(paths) != 2 || paths[0] == paths[1] {
		t.Fatalf("ждали новую сессию: %v", paths)
	}
	e.api.Mu.Lock()
	d := e.api.ItemsV["p1"][0].DescHTML
	e.api.Mu.Unlock()
	if !strings.Contains(d, "from plane") || strings.Count(d, "keep me") != 1 {
		t.Fatalf("в редактор пришёл не текст из Plane: %q %+v %v", d, m.s.status, paths)
	}
	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Fatal("файл черновика не удалён")
	}
}

// Отказ от «описание изменилось, перезаписать?» — текст сохраняется.
func TestContractTUIV8OverwriteDeclineKeepsText(t *testing.T) {
	syncExec(t)
	var path string
	inner := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		path = sess.Path
		e0 := inner(sess, editor, fn)
		return e0
	}
	m, _ := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `sed -i '1i my text'` })
	t.Cleanup(func() {
		if path != "" {
			os.Remove(path)
		}
	})
	m.s.cursor = "i1"
	m.s.items[0].DescHTML = "<p>old</p>"
	m.s.items[0].UpdatedAt = t0.Add(-time.Hour) // «загружено» раньше изменения в Plane
	press(t, m, "e")
	if _, ok := m.top().(*confirmScreen); !ok {
		t.Fatalf("нет вопроса: %T %+v", m.top(), m.s.status)
	}
	pressD(t, m, "n")
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "текст сохранён: "+path) {
		t.Fatalf("статус %+v", m.s.status)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "my text") {
		t.Fatalf("файл: %v", err)
	}
}

// Первый успешный опрос после ошибки — «связь восстановлена», в том числе с изменениями.
func TestContractTUIV8RecoveryWithChanges(t *testing.T) {
	m, e := flEnvNew(t, 30*time.Second, nil)
	e.setErr("WorkItems", flNet)
	feed(t, m, pollMsg{})
	e.setErr("WorkItems", nil)
	e.api.Mu.Lock()
	e.api.ItemsV["p1"][0].Name = "Renamed"
	e.api.ItemsV["p1"][0].UpdatedAt = t0.Add(time.Hour)
	e.api.Mu.Unlock()
	feed(t, m, pollMsg{})
	if !strings.HasPrefix(m.s.status.text, "связь восстановлена; изменения") || m.s.status.isErr {
		t.Fatalf("статус %+v", m.s.status)
	}
}
