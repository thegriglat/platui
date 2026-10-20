---
type: "contract"
status: "closed"
module: "tui"
updated: "2026-10-20"
summary: "Контракт TUI (C4 v3): Deps, раскладка (табы-состояния, список+превью), клавиши без vim, поиск, архитектура internal/tui, разбиение PT-5"
related: ["docs/contracts/tui.md", "docs/contracts/tui-services.md", "docs/contracts/tui-plane.md", "docs/plan/tui.md"]
contracts: [{"id": "C4", "version": 8}]
---

# Контракт TUI модуля tui

Продолжение `docs/contracts/tui.md` (C4 v1 там — исторический). Владелец — координатор. Структура — C0.
v3 (2026-10-20) учитывает решения пользователя того же дня: раскладка «список + превью», табы — состояния, поиск — ключевая фича (только по заголовку), `R` — обновление из API, справка `?`/`h`, без vim-клавиш.

## C4. tui v8 — пакет internal/tui (v6–v8 — дельты в docs/contracts/tui-features.md) (v4 2026-10-20: только GitLab, `M` — Go to MR; v5: подтверждение деструктивных действий)

Библиотеки (реестр PT-18, `docs/research/libraries.md` §4): `github.com/charmbracelet/bubbletea` v1.3.10, `bubbles` v1.0.0 (`textinput` — строки поиска и ввода, `viewport` — превью/карточка; `key`/`help` — по желанию; `table`/`list` не брать — нет подсветки ячеек), `lipgloss` v1.1.0, `glamour` v1.0.0; v2-ветки требуют go ≥ 1.25 — не брать. `bubbles` v1.0.0 объявляет `go 1.24.2` — директива go модуля поднимается до `1.24.2` (тулчейн 1.24.2 есть). Открытие URL — `github.com/cli/browser` v1.3.0 (`Stdout`/`Stderr` = `io.Discard`). Зависимости пакета: `app`, `config`, `plane`, `gitbranch`, `text`, `search` (C0); сеть, git, gh, `$EDITOR` — только через `app`/`text`.

### Внешний интерфейс

```go
type Deps struct {
    App       *app.App               // в тестах — app.New на фейках planetest/gitbranchtest/forgetest/cache.Mem
    Config    *config.Config
    OpenURL   func(url string) error // nil → browser.OpenURL (github.com/cli/browser)
    Clipboard func(text string) error // nil → text.OSC52(text, os.Getenv("TMUX") != "") в os.Stderr
    Now       func() time.Time       // nil → time.Now
    Start     StartAt
}
type StartAt struct {
    Project string // identifier; пусто — Config.DefaultProject, иначе экран проектов
    Item    string // "PROJ-42"; пусто — по текущей ветке (app.CurrentItem), если нашлась
}
func New(d Deps) tea.Model
func Run(d Deps) error // tea.NewProgram(New(d), tea.WithAltScreen()).Run()
```

### Раскладка

```
 PROJ  Все 42 │ Backlog 7 │ To Do 9 │ In Progress 5 │ Test Dev 3 │ Done 18      [Мои] [фильтр: …]   ← табы-состояния со счётчиками
 / login▏                                                                       найдено 6         ← строка поиска (когда активен)
 ID       Название                Сост.    Пр.   Исп.   Ветка │ PROJ-42 Fix login                ← таблица задач таба | превью
 PROJ-42  Fix [login] page        In Pr…   high  AL,MK  ● ↑2  │ In Progress · high · AL, MK
 PROJ-17  …                                                   │ описание (glamour) … комментарии …
 ↑↓ ход  Enter карточка  / поиск  s состояние  b ветка  m мои  v канбан  R обновить  ? справка      ← строка подсказок
 ✓ обновлено: 42 задачи, 18:40                                                                    ← статусная строка
```
- **Табы** — состояния проекта: первый «Все» (если `cfg.TabsAll()`), затем `app.StateTabs(cfg, meta.States)`; у каждого — счётчик задач (при активном поиске — счётчик совпадений). Под табом — таблица задач этого состояния. Смена таба: `Ctrl+←/→`, `Alt+1…9` (номер таба, «Все» — 1).
- **«Мои»** — переключатель-фильтр `m` (исполнитель — я), признак `[Мои]` в строке табов; действует на все табы и счётчики. Сохранённые фильтры `cfg.Filters` — меню `F` (выбор стрелками/Enter, `Alt+0` — сброс) и палитра; активный — справа в строке табов.
- **Канбан** — отдельный вид `v` (не таб): колонки — те же состояния, что табы (без «Все»); `v` или `Esc` — назад к таблице.
- **Панели**: таблица слева, превью карточки задачи под курсором справа; `Tab`/`Shift+Tab` — фокус между панелями (в превью `↑↓`, `PgUp/PgDn` прокручивают текст); ширина < 100 колонок — одна панель (таблица; `Enter` — карточка на весь экран, `Esc`/`q` — назад); ≥ 100 — 55/45.
- **Таблица** — плоская: ID, название, состояние, приоритет, исполнители (инициалы), признак ветки (`●` локальная, `↑n` ahead, `☁` удалённая) и MR (`⇄`/`✓` по `app.CachedMRs`); ширины под терминал: ID/приоритет/ветка фиксированы, название забирает остаток, состояние и исполнители обрезаются с `…`; при нехватке места скрываются сначала исполнители, затем состояние. Сортировка — `o` циклом: порядок API → приоритет → состояние → обновлено (новые сверху) → ID; текущая — `▲/▼` в заголовке колонки. Новые/изменённые при опросе (`fresh`) подсвечены.
- **Строка подсказок** и **статусная строка** внизу (подсказки — из таблицы привязок текущей области, поле `Hint`, до ширины окна).
- Действия над задачей работают и в таблице/канбане (задача под курсором), и в полной карточке.

### Поиск — ключевая фича (приоритет выше прочих пунктов §4.1)

- `/` — строка поиска (не больше `search.MaxQuery` = 32 рун — ответ на Q1; лишний ввод не принимается); поиск **по мере ввода** (каждое нажатие → `search.Index.Search`, без запросов к API) **только по заголовку** (решение 2026-10-20; описание, метки, исполнители — нет) по **всем** состояниям: таблица текущего таба показывает только совпадения в порядке `Score`, счётчики табов — число совпадений; совпавшие руны заголовка подсвечены (`search.Highlight`) в таблице и в превью.
- Запрос-ключ (`search.ParseKey` ok: `PROJ-42`, `42`): фильтр таблицы не меняется, строка поиска показывает «Enter — открыть PROJ-42»; `Enter` → `Index.Lookup` в текущем проекте, иначе `app.Find` (любой проект workspace) → карточка (проект переключается); не найдено — статус «PROJ-42 не найдена».
- Курсор по результатам — `↑/↓`; `Enter` (не ключ) — закрыть строку поиска, оставив фильтр; `Esc` — сброс поиска; повторный `/` — правка запроса.
- Индекс (`app.SearchDocs` → `search.New`) строится при загрузке/обновлении задач проекта (все страницы, кэш C8), перестраивается при `itemsMsg`. Серверный поиск в TUI не используется.
- `Ctrl+G` — тот же переход по ключу (ввод `42` или `PROJ-42`); то же из палитры.

### Клавиши (решения пользователя 2026-10-20: без vim)

Навигация — стрелки, `PgUp/PgDn`, `Home/End`, `Enter`, `Esc`, `Tab/Shift+Tab` и сочетания с `Shift/Ctrl/Alt`; `j/k`, `gg/G`, `n/N`, `h/l`, числовые префиксы — **не используются**.

| Группа | Клавиши |
|---|---|
| навигация | `↑/↓` строка, `PgUp/PgDn`, `Home/End`, `Enter` карточка, `Esc` назад/закрыть, `Tab/Shift+Tab` панель, `Ctrl+←/→` и `Alt+1…9` таб, `P` проект |
| поиск | `/` поиск, `Ctrl+G` переход по ID, `Ctrl+P` палитра |
| задача | `s` состояние, `p` приоритет, `a` назначить на себя, `e` описание в `$EDITOR`, `c` комментарий в `$EDITOR`, `n` новая задача, `y` копировать, `O` открыть в браузере, `t` таймер, `space` отметить (массовые — `s`/`a`/`L` при отмеченных; `L` — метки, решение 2026-10-20) |
| git | `b` ветка, `S` старт работы, `W` готово к ревью (MR в GitLab), `M` перейти к MR, `X` уборка веток |
| виды | `v` канбан, `m` мои, `F` сохранённые фильтры, `o` сортировка, `R` обновить из API, `?`/`h` справка, `q` выход/назад |
| канбан | `←/→` колонка, `↑/↓` задача, `Shift+←/→` перенос в соседнее состояние |

(`O` — браузер, т.к. `o` — сортировка; `W` — ревью, т.к. `R` — обновление.)
- `R` (решение 2026-10-20): `app.Reload` текущего проекта в обход кэша + перезагрузка открытой карточки и комментариев; индикатор загрузки в статусной строке; итог «обновлено: N задач, ЧЧ:ММ»; ошибка — в статусной строке, прежние данные остаются.
- `M` — «Go to MR» (решение 2026-10-20): `app.MRs(fresh=false)` задачи под курсором/открытой; один MR — открыть в браузере (`Deps.OpenURL`), несколько — список выбора (Enter — открыть), нет — «MR не найдены» в статусе; glab не установлен — сообщение в статусе. Признак MR (`⇄` opened, `✓` merged) — в таблице и превью только по `app.CachedMRs` (без вызовов glab при перемещении курсора).
- `?` или `h` — оверлей справки: все привязки текущей области по группам (как в таблице выше), из того же реестра действий, что и строка подсказок; закрытие — `Esc`, `q`, `?`, `h`.

### Подтверждение деструктивных действий (v5, решение пользователя 2026-10-20)

Деструктивное действие выполняется только после диалога `y/N`: по умолчанию «нет» — `y`/`Y` подтверждает, `Enter`, `Esc`, `n` и любая другая клавиша отменяют (статус «отменено»). Диалог — общий `confirm.go` (PT-28): `func confirm(s *session, question string, onYes tea.Cmd) tea.Cmd` (push экрана Scope "dialog", CapturesInput). Текст вопроса называет объект и число («Удалить 3 ветки?», «Перевести PROJ-42 в Done?»).

| Действие | Где | Условие диалога |
|---|---|---|
| удаление веток (`X`) | PT-5d | всегда; число отмеченных |
| принудительное удаление (`ErrNotMerged`) | PT-5d | всегда, отдельный вопрос |
| переключение на существующую ветку (`b`) | PT-5d | всегда (не деструктивно, но вопрос по решению Q4) |
| массовые `s`/`a`/`L` | PT-5e | всегда; число задач |
| смена состояния на группу `completed`/`cancelled` (`s`, перенос в канбане `Shift+←/→`, массовое `s`) | PT-5c, PT-5f, PT-5e | только если целевая группа completed/cancelled |
| перезапись описания (`e`) | PT-5c | если `app.Item` перед сохранением даёт `UpdatedAt` новее загруженного — «описание изменилось в Plane после загрузки, перезаписать?» |
| перезапись шаблона сообщения коммита / хука | app/gitbranch | не выполняется вовсе: существующий файл сообщения не перезаписывается, чужой хук — `ErrHookExists` (C2 v4, C5 v5) → Warning в статусе |
| удаление worktree, комментариев, задач, очистка кэша | — | действий нет; появятся — только с диалогом |

Тест на каждое действие таблицы (`TestContractTUIConfirm*`): без `y` вызова фейка нет; `Enter` = нет; после `y` — вызов есть.

### Внутренняя архитектура (неэкспортируемая, общая для подзадач PT-5x)

Файлы PT-5a (каркас); остальные подзадачи их не правят, новые возможности — в своих файлах:
- `model.go` — корневая модель `model`: стек экранов, глобальные клавиши, маршрутизация сообщений, размер окна;
- `session.go` — `session` (общее состояние, один указатель на программу);
- `screen.go` — интерфейс `screen`, `pushScreenMsg`/`popScreenMsg`;
- `msgs.go` — общие сообщения и команды-загрузчики;
- `actions.go` — реестр действий; `keys.go` — нормализация клавиш (`tea.KeyMsg.String()`);
- `nav.go` — `cursorList`: курсор, прокрутка, `↑/↓`, `PgUp/PgDn`, `Home/End`;
- `layout.go` — рамка: строка табов, строка поиска, панели, строка подсказок, статусная строка; `tabs.go` — табы-состояния, счётчики, `Ctrl+←/→`, `Alt+цифра`;
- `styles.go` — стили из `config.Color`, `ColorsDisabled` → без цветов; `status.go`; `help.go` — оверлей `?`/`h`.

```go
type session struct {
    deps     Deps
    st       styles
    width, height int
    me       plane.Member
    projects []plane.Project
    project  *plane.Project
    meta     app.Meta
    items    []plane.WorkItem               // все задачи проекта, порядок API
    tabs     []plane.State                  // app.StateTabs
    tab      int                            // 0 — «Все» (если включён), далее по tabs
    mine     bool                           // фильтр «Мои»
    filter   int                            // 0 — нет, 1…9 — cfg.Filters
    sortBy   int
    index    *search.Index                  // по items, перестраивается при itemsMsg
    query    string                         // текущий поиск; "" — нет
    hits     map[string]search.Hit          // по item ID, при query != ""
    marks    map[string]bool
    branches map[string]gitbranch.BranchStatus
    fresh    map[string]time.Time
    focus    int                            // 0 — левая панель, 1 — превью
    cursor   string                         // item ID под курсором
    loading  bool
    status   statusLine
}
// Видимые задачи текущего таба с учётом mine/filter/query/sortBy — одна функция (PT-5a, tabs.go):
func (s *session) visible() []plane.WorkItem
func (s *session) tabCount(i int) int        // с учётом mine/filter/query

type screen interface {
    Update(s *session, msg tea.Msg) (screen, tea.Cmd)
    View(s *session) string
    Title() string
    Scope() string        // "projects"|"main"|"kanban"|"item"|"dialog"|"palette"|"help"
    CapturesInput() bool  // текстовый ввод: глобальные клавиши, кроме Ctrl+C/Esc, не перехватываются
}
type pane interface {     // панели основного экрана
    Update(s *session, msg tea.Msg) (pane, tea.Cmd)
    View(s *session, width, height int) string
}
var newTable   func(s *session) pane  // PT-5b присваивает в init(); PT-5a — заглушка
var newPreview func(s *session) pane  // PT-5c; nil — превью не показывается
var newKanban  func(s *session) screen // PT-5f; nil — `v` сообщает «нет вида»

type action struct {
    ID     string     // "item.state", "view.mine", …
    Title  string     // для палитры и справки
    Group  string     // "навигация"|"поиск"|"задача"|"git"|"виды"|"канбан" — группы справки
    Keys   []string   // как tea.KeyMsg.String(): "s", "ctrl+p", "alt+1", "shift+right"
    Hint   string     // подпись в строке подсказок; пусто — не показывать
    Scopes []string   // где доступно; пусто — везде
    Bulk   bool       // действие над отмеченными
    Run    func(m *model) tea.Cmd
}
func registerAction(a action) // из init(); дубль ID или конфликт клавиш в общей области (кроме пары Bulk/не-Bulk) — panic, тест TestActionsUnique
```
Сообщения (`msgs.go`): `itemsMsg{projectID string; items []plane.WorkItem; changes app.Changes; fromCache, reload bool; err error}`, `metaMsg{meta app.Meta; err error}`, `itemMsg{item plane.WorkItem; err error}`, `commentsMsg{itemID string; comments []plane.Comment; err error}`, `branchesMsg{m map[string]gitbranch.BranchStatus; err error}`, `statusMsg{text string; isErr bool}`, `pollMsg{}`. Команды: `loadItems(s, useCache bool)` (сначала `CachedItems` → `itemsMsg{fromCache:true}`, затем `RefreshItems`), `reload(s)` (`app.Reload` → `metaMsg` + `itemsMsg{reload:true}`), `loadBranches`, `patchItem(s, id, plane.WorkItemPatch)`. Ошибки команд → статусная строка (красным), программа не падает.
Автообновление (PT-5a): `tea.Tick(cfg.RefreshInterval())`, следующий — `app.NextPoll`; изменения → `fresh`; `cfg.NotifyEnabled()` — `\a` при непустых New/Updated; `0` — выкл.

### Разбиение PT-5

| Подзадача | Файлы | Что |
|---|---|---|
| PT-5a каркас | перечисленные выше, `go.mod` (`cmd/platui` — не трогает: сборку делает PT-10 по C10) | корень, сессия, раскладка, табы со счётчиками и переключением, `visible()`, реестр, nav, стили, статус, справка `?`/`h`, `R`, `m`, автообновление, старт по ветке, заглушки панелей |
| PT-5b таблица и поиск | `projects.go`, `table.go`, `columns.go`, `searchbar.go`, `filters.go`, `sort.go` | экран проектов `P`; таблица (колонки, ширины, признак ветки, подсветка `fresh` и совпадений); поиск `/` целиком (индекс по заголовкам, по мере ввода, счётчики, запрос-ключ → переход); `Ctrl+G`; `F`; `o`; `space` |
| PT-5c карточка | `item.go`, `preview.go`, `pickers.go`, `copy.go`, `timer.go` | рендер карточки (превью и полный экран): glamour (`HTMLToMarkdown` → glamour по ширине; `ColorsDisabled` → notty), подсветка совпадений поиска в заголовке, метаданные, комментарии; `s`, `p`, `a`, `O`, `y` (`CopyItems`: i/u/t/b/m), `e`, `c` (`text.EditSession` + `tea.ExecProcess`), `t` |
| PT-5d git и создание | `create.go`, `branch.go`, `workflow.go`, `stale.go`, `mr.go` | `n` (строка `ParseQuick` + `Ctrl+E` описание в `$EDITOR`), `b`, `S` (`app.Start`), `W` (`app.Review`, URL MR в статусе), `M` (`app.MRs` → открыть/выбор), `X` (`StaleBranches` → подтверждение → `DeleteBranches`) |
| PT-5e палитра и массовые | `palette.go`, `bulk.go` | `Ctrl+P`: `search.Fuzzy` по действиям (реестр, текущая область), задачам, проектам, табам, фильтрам; `PROJ-N` в палитре → `app.Find`; массовые `s`/`a`/`L` при отмеченных — `BulkPatch`/`BulkAssign`/`BulkLabel`, итог «n ок, m ошибок» |
| PT-5f канбан | `kanban.go` | вид `v`: колонки-состояния, `←/→`, `↑/↓`, `Shift+←/→` перенос (`patchItem`), горизонтальная прокрутка колонок на узком экране |

Порядок: PT-5a, затем PT-5b…PT-5f параллельно (после PT-9 app). PT-5b — первой по приоритету (поиск).

### Тесты

Модели — без терминала: `m.Update(tea.KeyMsg{…})`; хелпер `run(t, cmd) []tea.Msg` разворачивает `tea.Batch`/`tea.Sequence`, `tea.Tick` не ждёт. Хелперы PT-5a (`testutil_test.go`): `newTestModel(t) (*model, *planetest.Fake, *gitbranchtest.Fake, *forgetest.Fake)` с образцовым проектом (5 состояний разных групп, в т.ч. cancelled; 2 участника; 2 метки; 12 задач, у 3 — описание), `press(m, keys ...string)`, `view(m) string`. Контрактный тест каждой подзадачи — `TestContractTUI<Подзадача>*` (`Base`, `Table`, `Item`, `Git`, `Palette`, `Kanban`): каждое действие из её строки — нажатие → ожидаемый вызов фейка (`Calls`) и/или состояние сессии/текст `View`. Для поиска (PT-5b) — дополнительно: ввод по символу меняет `visible()` и счётчики табов без вызовов API; совпадение только по заголовку (слово из описания/метки не находит); `PROJ-N` в другом проекте → `WorkItemByKey` в `Calls`; 33-я руна не принимается; `Esc` сбрасывает.
