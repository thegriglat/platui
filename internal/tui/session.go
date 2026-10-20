package tui

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/cli/browser"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/search"
	"github.com/thegriglat/platui/internal/text"
)

// Deps — зависимости TUI (C4 v3).
type Deps struct {
	App       *app.App
	Config    *config.Config
	OpenURL   func(url string) error  // nil → browser.OpenURL
	Clipboard func(text string) error // nil → OSC52 в os.Stderr
	Now       func() time.Time        // nil → time.Now
	Start     StartAt
}

// StartAt — с чего начать.
type StartAt struct {
	Project string // identifier; пусто — Config.DefaultProject, иначе экран проектов
	Item    string // "PROJ-42"; пусто — по текущей ветке
}

func (d Deps) withDefaults() Deps {
	if d.Config == nil {
		d.Config = &config.Config{}
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.OpenURL == nil {
		d.OpenURL = func(u string) error {
			browser.Stdout, browser.Stderr = io.Discard, io.Discard
			return browser.OpenURL(u)
		}
	}
	if d.Clipboard == nil {
		d.Clipboard = func(t string) error {
			_, err := io.WriteString(os.Stderr, text.OSC52(t, os.Getenv("TMUX") != ""))
			return err
		}
	}
	return d
}

// Порядок сортировки (sortBy).
const (
	sortAPI = iota
	sortPriority
	sortState
	sortUpdated
	sortID
)

// freshTTL — сколько новая/изменённая задача считается свежей.
const freshTTL = 10 * time.Minute

// session — общее состояние программы.
type session struct {
	deps          Deps
	st            styles
	width, height int
	me            plane.Member
	projects      []plane.Project
	project       *plane.Project
	meta          app.Meta
	items         []plane.WorkItem
	tabs          []plane.State
	tab           int
	mine          bool
	filter        int
	sortBy        int
	index         *search.Index
	query         string
	hits          map[string]search.Hit
	marks         map[string]bool
	branches      map[string]gitbranch.BranchStatus
	fresh         map[string]time.Time
	focus         int
	cursor        string
	loading       bool
	status        statusLine

	// служебное PT-5a
	searching  bool   // идёт ввод запроса (PT-5b): главный экран перехватывает клавиши
	searchLine string // готовая строка поиска от PT-5b; пусто — «/ запрос найдено N»
	live       bool   // данные проекта получены из API
	pending    string // ID задачи, которую нужно выбрать после загрузки
	openItem   string // ID задачи в открытой карточке (ставит openItem, снимает pop)
	pollArmed  bool
	polling    bool
	pollEvery  time.Duration
	bell       func()
	net        netState // связь и черновики (offline.go)

	panel panelState // раскладка панелей: ширина превью, zoom (panels.go)
	hist  history    // история открытых карточек (history.go)

	ctx    context.Context // отменяется при выходе из программы (свои команды)
	cancel context.CancelFunc
}

func newSession(d Deps) *session {
	ctx, cancel := context.WithCancel(context.Background())
	return &session{deps: d, st: newStyles(d.Config), ctx: ctx, cancel: cancel,
		panel: panelState{pct: clampPct(d.Config.PreviewPercent())}, marks: map[string]bool{}, fresh: map[string]time.Time{},
		branches: map[string]gitbranch.BranchStatus{}, bell: func() { _, _ = io.WriteString(os.Stderr, "\a") }}
}

// resetProject переключает проект: данные сбрасываются.
func (s *session) resetProject(p plane.Project) {
	s.project = &p
	s.meta, s.items, s.tabs, s.tab = app.Meta{Project: p}, nil, nil, 0
	s.index, s.query, s.hits, s.filter, s.cursor = nil, "", nil, 0, ""
	s.marks, s.fresh, s.branches = map[string]bool{}, map[string]time.Time{}, map[string]gitbranch.BranchStatus{}
	s.live, s.loading, s.searching, s.searchLine, s.polling, s.openItem = false, true, false, "", false, ""
}

// rebuildIndex перестраивает поисковый индекс (по заголовкам) и совпадения.
func (s *session) rebuildIndex() {
	if s.project == nil {
		return
	}
	s.index = search.New(app.SearchDocs(app.Meta{Project: *s.project}, s.items))
	s.setQuery(s.query)
}

// setQuery задаёт запрос (не длиннее search.MaxQuery рун); ключ-запрос фильтр не меняет.
func (s *session) setQuery(q string) {
	if r := []rune(q); len(r) > search.MaxQuery {
		q = string(r[:search.MaxQuery])
	}
	s.query, s.hits = q, nil
	if q == "" || s.index == nil {
		return
	}
	if _, _, ok := search.ParseKey(q); ok {
		return
	}
	s.hits = map[string]search.Hit{}
	for _, h := range s.index.Search(q) {
		s.hits[h.ID] = h
	}
}

// itemByID — задача проекта по ID.
func (s *session) itemByID(id string) (plane.WorkItem, bool) {
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return plane.WorkItem{}, false
}
