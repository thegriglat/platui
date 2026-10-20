package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// statEntry — результат app.DiffStat для задачи (loading — запрос в пути).
type statEntry struct {
	stat    gitbranch.Stat
	ok      bool
	loading bool
}

type diffStatMsg struct {
	key  string
	stat gitbranch.Stat
	ok   bool
	err  error
}

// statKey — ключ кэша сессии: задача и время обновления.
func statKey(it plane.WorkItem) string { return fmt.Sprintf("%s|%d", it.ID, it.UpdatedAt.UnixNano()) }

// loadDiffStat запрашивает статистику ветки асинхронно.
func loadDiffStat(s *session, it plane.WorkItem) tea.Cmd {
	a, p := s.deps.App, *s.project
	return func() tea.Msg {
		st, ok, err := a.DiffStat(p, it)
		return diffStatMsg{key: statKey(it), stat: st, ok: ok, err: err}
	}
}

// wantStat: если для задачи ещё нет записи в кэше — помечает «загружается» и возвращает запрос.
func (c *cardView) wantStat(s *session, id string) tea.Cmd {
	it, ok := s.itemByID(id)
	if !ok || s.deps.App == nil || s.project == nil {
		return nil
	}
	k := statKey(it)
	if _, ok := c.stats[k]; ok {
		return nil
	}
	c.stats[k] = statEntry{loading: true}
	return loadDiffStat(s, it)
}

// onStat запоминает результат; ошибка — запись «нет ветки» (строки в превью не будет).
func (c *cardView) onStat(m diffStatMsg) {
	c.stats[m.key] = statEntry{stat: m.stat, ok: m.ok && m.err == nil}
}

// statLine — «Ветка: +ins −del (n файлов)»; пусто, если ветки нет или данные не пришли.
func (c *cardView) statLine(it plane.WorkItem) string {
	e, ok := c.stats[statKey(it)]
	if !ok || e.loading || !e.ok {
		return ""
	}
	return fmt.Sprintf("Ветка: +%d −%d (%s)", e.stat.Insertions, e.stat.Deletions, filesWord(e.stat.Files))
}

func filesWord(n int) string {
	switch m := n % 100; {
	case m >= 11 && m <= 14:
		return fmt.Sprintf("%d файлов", n)
	case n%10 == 1:
		return fmt.Sprintf("%d файл", n)
	case n%10 >= 2 && n%10 <= 4:
		return fmt.Sprintf("%d файла", n)
	}
	return fmt.Sprintf("%d файлов", n)
}
