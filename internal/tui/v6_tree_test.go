package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
)

func TestContractTUIV6DiffStat(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Theme.NoColor = true })
	e.git.BranchesV["PROJ-1-task-number-1"] = gitbranch.BranchStatus{Name: "PROJ-1-task-number-1", Local: true}
	e.git.Stats = map[string]gitbranch.Stat{"PROJ-1-task-number-1": {Files: 3, Insertions: 12, Deletions: 4}}
	m.s.cursor = "i1"
	if strings.Contains(view(m), "Ветка: +") {
		t.Fatal("stat before load")
	}
	send(t, m, previewTickMsg{})
	send(t, m, previewTickMsg{})
	if v := view(m); !strings.Contains(v, "Ветка: +12 −4 (3 файла)") {
		t.Fatalf("no stat line:\n%s", v)
	}
	// задача без ветки — строки нет
	m.s.cursor = "i2"
	send(t, m, previewTickMsg{})
	send(t, m, previewTickMsg{})
	if v := view(m); strings.Contains(v, "Ветка: +") {
		t.Fatalf("stat for item without branch:\n%s", v)
	}
}

func TestContractTUIV6Bookmarks(t *testing.T) {
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Theme.NoColor = true })
	s := m.s
	s.cursor = "i3"
	keys(t, m, "B")
	s.cursor = "i2"
	keys(t, m, "B")
	rows := s.buildRows(map[string]bool{})
	if rows[0].it.ID != "i3" || rows[1].it.ID != "i2" || !rows[0].pinned || rows[2].pinned || rows[2].it.ID != "i1" {
		t.Fatalf("order: %v %v %v", rows[0].it.ID, rows[1].it.ID, rows[2].it.ID)
	}
	if v := view(m); !strings.Contains(v, "☆ PROJ-3") {
		t.Fatalf("no star:\n%s", v)
	}
	// поиск — закладки не закрепляются
	s.setQuery("Task number 1")
	if r := s.buildRows(map[string]bool{}); r[0].pinned {
		t.Fatal("pinned in search")
	}
	s.setQuery("")
	s.cursor = "i3"
	keys(t, m, "B")
	if r := s.buildRows(map[string]bool{}); r[0].it.ID != "i2" {
		t.Fatalf("unpin: %v", r[0].it.ID)
	}
}

func TestContractTUIV6Subtasks(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Theme.NoColor = true })
	items := e.api.ItemsV["p1"]
	items[1].ParentID, items[2].ParentID, items[3].ParentID = "i1", "i1", "i2"
	feed(t, m, itemsMsg{projectID: "p1", items: items})
	s := m.s
	s.cursor = "i1"
	v := view(m)
	if !strings.Contains(v, "▸ (2) Task number 1") || strings.Contains(v, "Task number 2") {
		t.Fatalf("collapsed:\n%s", v)
	}
	send(t, m, tea.KeyMsg{Type: tea.KeyRight})
	v = view(m)
	if !strings.Contains(v, "▾ Task number 1") || !strings.Contains(v, "  ▸ (1) Task number 2") || !strings.Contains(v, "Task number 3") {
		t.Fatalf("expanded:\n%s", v)
	}
	send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	send(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if !strings.Contains(view(m), "    Task number 4") {
		t.Fatalf("depth 2:\n%s", view(m))
	}
	send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	send(t, m, tea.KeyMsg{Type: tea.KeyLeft}) // на потомке — к родителю
	if s.cursor != "i2" {
		t.Fatalf("cursor %s", s.cursor)
	}
	send(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	if strings.Contains(view(m), "Task number 4") {
		t.Fatal("not collapsed")
	}
	// поиск — плоский список
	s.setQuery("Task number")
	if r := s.buildRows(map[string]bool{}); len(r) == 0 || r[0].depth != 0 || r[0].kids != 0 {
		t.Fatal("search not flat")
	}
}
