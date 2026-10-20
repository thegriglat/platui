---
type: "contract"
status: "closed"
module: "tui"
updated: "2026-10-20"
summary: "Стыки сервисного слоя tui — пакеты text, forge, cache, search и app (сценарии поверх plane и gitbranch для TUI и CLI)"
related: ["docs/contracts/tui.md", "docs/plan/tui.md"]
contracts: [{"id": "C6", "version": 2}, {"id": "C7", "version": 2}, {"id": "C8", "version": 1}, {"id": "C5", "version": 7}, {"id": "C9", "version": 4}, {"id": "C10", "version": 4}]
---

# Контракты сервисного слоя модуля tui

Продолжение `docs/contracts/tui.md` (C0–C4), вынесено из-за лимита размера. Владелец — координатор модуля tui. Структура кода — C0 (слои, ≤ 300 строк на файл, фейки в `<пакет>test`).
Слои: `text`, `forge`, `cache`, `search` — листовые (не импортируют `platui/internal/...`); `app` → config, gitbranch, plane, text, forge, cache, search.

## C6. text v2 — пакет internal/text (PT-11; v2 — дельта в docs/contracts/tui-features.md)

Чистые функции без сети и без состояния. Потребители: app (PT-9), tui (PT-5x), cli (PT-10).

```go
// Быстрое создание: "fix login !high @me @anna #bug #ui"
type Quick struct {
    Title     string   // оставшиеся слова через один пробел, обрезаны края
    Priority  string   // "" или urgent|high|medium|low|none
    Assignees []string // без "@", в порядке появления, без повторов; "me" — особое значение
    Labels    []string // без "#", в порядке появления, без повторов
}
var ErrEmptyTitle = errors.New("empty title")
func ParseQuick(s string) (Quick, error)
```
Правила `ParseQuick`: токены — по пробельным символам; `!x` — приоритет: полные имена и однобуквенные `u|h|m|l|n` (регистр не важен), неизвестное `!x` и второй приоритет — ошибка с токеном; `@x`/`#x` — непустое `x`, иначе токен идёт в название; `\!`, `\@`, `\#` в начале токена — буквальный символ в названии; пустое название → `ErrEmptyTitle`.

```go
func MarkdownToHTML(md string) string   // goldmark, CommonMark + GFM (таблицы, зачёркивание, списки задач, автоссылки); сырой HTML экранируется (без WithUnsafe)
func HTMLToMarkdown(h string) string    // для правки существующего описания в $EDITOR: p, br, h1–h6, strong/b, em/i, s/del, code, pre, a, ul/ol/li (вложенность), blockquote, hr; прочие теги — только текст; сущности раскодированы
```
Обратимость: `HTMLToMarkdown(MarkdownToHTML(md))` для md из перечисленных конструкций даёт эквивалентный md (тест на таблице примеров сравнивает повторный `MarkdownToHTML`).

```go
// $EDITOR: файл создаётся, команда собирается, TUI запускает её через tea.ExecProcess, CLI — Run().
type EditSession struct{ Path string }               // временный файл *.md в os.TempDir()
func NewEditSession(initial, hint string) (*EditSession, error) // hint — строки-комментарии «# …» в конце файла
func (s *EditSession) Command(editor string) *exec.Cmd          // sh -c '<editor> "$1"' platui-edit <Path>; editor — config.EditorCmd()
func (s *EditSession) Result() (string, error)                  // текст без строк-подсказок, края обрезаны; пусто → ErrEmptyEdit
func (s *EditSession) Close() error                             // удаляет файл
var ErrEmptyEdit = errors.New("empty text, cancelled")

// Буфер обмена через OSC 52.
func OSC52(text string, tmux bool) string // "\x1b]52;c;<base64>\x07"; tmux → обёртка "\x1bPtmux;\x1b<seq>\x1b\\"
```
Зависимость: `github.com/yuin/goldmark` (версия с `go` ≤ 1.24 в go.mod). Контрактный тест: `internal/text/contract_test.go`, `TestContractText*`: таблица ParseQuick (все правила выше, ошибки), MarkdownToHTML (заголовок, список, ссылка, код, таблица, `<script>` экранирован), HTMLToMarkdown (образец выше), обратимость, EditSession с editor-фейком (`sh -c 'printf "new" > "$1"'`-скрипт), OSC52 с tmux и без.

## C7. forge v2 — пакет internal/forge (PT-12; v2 — PT-23)

Только GitLab через CLI `glab` (решение пользователя 2026-10-20: gh/GitHub убрать). Потребитель: app. v2 заменяет v1 целиком (gh, `New(tool,…)`, `ErrDisabled` для `none` — удаляются).

```go
type MRRequest struct {
    Dir          string // каталог репозитория (cwd команды)
    Head         string // исходная ветка
    Target       string // пусто — по умолчанию проекта
    Title, Body  string
    Draft        bool
}
type MR struct {
    IID          int
    Title, URL   string // URL — web_url
    State        string // opened|merged|closed (как отдаёт glab)
    SourceBranch, TargetBranch string
    UpdatedAt    time.Time
}
type Options struct {
    Glab string // бинарь; "" → "glab"
    Host string // "" — не задавать; иначе env GITLAB_HOST=<Host> для команд
    Repo string // "" — glab берёт из remote каталога; иначе флаг -R <Repo>
    Run  Runner // nil → ExecRunner()
}
type Tool interface {
    CreateMR(ctx context.Context, r MRRequest) (url string, err error)
    FindMRs(ctx context.Context, dir string, branches []string, key string) ([]MR, error)
}
type Runner func(ctx context.Context, dir string, env []string, name string, args ...string) (stdout string, err error) // v2: + env (доп. переменные)
func ExecRunner() Runner              // os/exec, LC_ALL=C + env; ошибка содержит stderr; exec.ErrNotFound → ErrNotFound
func New(o Options) *GitLab           // *GitLab реализует Tool
var ErrNotFound = errors.New("glab not installed")
```
Команды (флаги сверены с `glab mr list --help` в среде, 2026-10-20):
- `CreateMR`: `glab mr create --source-branch <Head> --title <T> --description <B> [--target-branch <Target>] [--draft] --yes [-R <Repo>]`; URL — первое `https?://\S+` в stdout; «already exists» в тексте ошибки/вывода → `glab mr list -s <Head> -A -F json [-R]` → `web_url` первого; URL не найден — ошибка с выводом команды.
- `FindMRs`: для каждой ветки из `branches` — `glab mr list -s <ветка> -A -F json [-R]`; если `key` не пуст — `glab mr list -A --search <key> -F json [-R]` (заголовок и описание) и `glab mr list -A -o updated_at -P 100 -F json [-R]` с отбором тех, у кого `source_branch` содержит `key` без учёта регистра; объединение по IID; state `closed` отбрасывается (нужны открытые и влитые); порядок — opened раньше merged, затем `UpdatedAt` по убыванию. Поля JSON: `iid`, `title`, `web_url`, `state`, `source_branch`, `target_branch`, `updated_at` (RFC3339). Пустой вывод/`[]` — пустой список без ошибки.
Фейк: `internal/forge/forgetest.Fake{URL string; MRs []forge.MR; Err error; Reqs []forge.MRRequest; Finds [][]string}`. Контрактный тест (`TestContractForge*`, v1-тесты про gh — удалить): точные аргументы и env (Host/Repo заданы и нет), разбор URL, «already exists» → list, FindMRs — объединение, отбор по source_branch, отбрасывание closed, порядок, пустой вывод, ErrNotFound.

## C8. cache v1 — пакет internal/cache (PT-13)

Кэш на диске для старта с кэша и состояние таймеров. Потребитель: app. Без знания типов plane — значения любые JSON-сериализуемые.

```go
type Store interface {
    Load(key string, v any) (saved time.Time, ok bool, err error) // нет файла или битый/чужая версия → ok=false, err=nil
    Save(key string, v any) error
}
type FileStore struct{ Dir string }      // реализует Store
func NewFileStore(dir string) *FileStore // каталог создаётся при первом Save (0700)
var _ Store = (*FileStore)(nil)
type Mem struct{ … }                     // в памяти, для тестов потребителей; реализует Store
func NewMem(now func() time.Time) *Mem
```
Формат файла `<Dir>/<key>.json` (key — `[a-zA-Z0-9._-]+`, `/` в ключе → подкаталог; иное → ошибка): `{"v":1,"saved":"<RFC3339Nano>","data":<значение>}`; запись — временный файл в том же каталоге + `os.Rename`; права 0600. Конкурентные `Save` одного ключа из двух процессов не портят файл (последний выигрывает). Контрактный тест: `internal/cache/contract_test.go`, `TestContractCache*`: круг Save/Load, отсутствие, битый файл, `"v":2` → ok=false, недопустимый ключ, подкаталоги, права 0600, атомарность (нет `*.tmp` после Save), Mem.

## C5. app v7 — пакет internal/app (v6, v7 — дельты в docs/contracts/tui-features.md; (PT-9, v2 — PT-20, v3 — PT-24, v4 — PT-5d, v5 — PT-29; дополнено 2026-10-20 до выдачи: Reload, StateTabs, SearchDocs, ServerSearch, Find по всем проектам)

Сценарии, общие для TUI и CLI. Все внешние эффекты — через зависимости `Deps`. Потребители: tui (PT-5x), cli (PT-10).

```go
type Deps struct {
    API    plane.API
    Git    gitbranch.Git   // nil — не git-репозиторий: git-сценарии → ErrNoRepo
    Forge  forge.Tool      // v3: nil → forge.New(forge.Options{Glab: cfg.GitLab.Glab, Host: cfg.GitLab.Host, Repo: cfg.GitLab.Repo})
    Store  cache.Store     // nil → без кэша
    Config *config.Config
    Now    func() time.Time // nil → time.Now
}
func New(d Deps) *App
var (
    ErrNoRepo        = errors.New("not in a git repository")
    ErrNoItemForBranch = errors.New("current branch does not reference a work item")
    ErrItemNotFound  = errors.New("work item not found")
    ErrProjectNotFound = errors.New("project not found")
    ErrUnknownMember = errors.New("unknown member")   // текст ошибки содержит имя
    ErrUnknownLabel  = errors.New("unknown label")
    ErrNoTimer       = errors.New("no running timer")
)

type Meta struct { // кэшируется в памяти App на время жизни процесса; Invalidate(projectID) сбрасывает
    Project plane.Project
    States  []plane.State
    Members []plane.Member
    Labels  []plane.Label
}
func (a *App) Me(ctx) (plane.Member, error)                          // один запрос за жизнь App
func (a *App) Projects(ctx) ([]plane.Project, error)
func (a *App) Project(ctx, identifier string) (plane.Project, error) // без учёта регистра; нет → ErrProjectNotFound
func (a *App) Meta(ctx, projectID string) (Meta, error)
func (a *App) Invalidate(projectID string)

// Задачи и кэш (ключ Store: "<workspace>/items/<projectID>").
type Changes struct{ New, Updated []plane.WorkItem; Removed []string } // по ID; Updated — UpdatedAt изменился
func Diff(old, cur []plane.WorkItem) Changes
func (a *App) CachedItems(projectID string) ([]plane.WorkItem, time.Time, bool)
func (a *App) RefreshItems(ctx, projectID string) ([]plane.WorkItem, Changes, error) // API → Diff с прошлым (память, иначе кэш) → Save
func NextPoll(base, prev time.Duration, err error) time.Duration // err — ErrRateLimited: max(prev,base)*2, ≤ 10m; иная ошибка: max(prev,base)*2, ≤ 5m; nil — base

func (a *App) Find(ctx, identifier string, seq int) (plane.Project, plane.WorkItem, error) // проект по identifier (Projects); задача — из памяти/кэша проекта, иначе plane.WorkItemByKey (любой проект workspace); нет → ErrItemNotFound / ErrProjectNotFound

// Обновление в обход кэша (клавиша R, решение пользователя 2026-10-20): Invalidate → Meta из сети → все страницы задач → Diff с прошлым → Save кэша (meta и items).
func (a *App) Reload(ctx, projectID string) (Meta, []plane.WorkItem, Changes, error) // ошибка — прежние данные не трогаются (ни память, ни кэш)

// Табы-состояния (C1 v3): cfg.Tabs.States непусто — состояния с этими именами (без учёта регистра) в порядке конфига, отсутствующие пропускаются;
// иначе все, кроме групп cfg.Tabs.HideGroups, по группе (неизвестная → перед backlog; backlog, unstarted, started, completed, cancelled), затем Sequence, затем Name.
func StateTabs(cfg *config.Config, states []plane.State) []plane.State

// Поиск (C9; решения пользователя 2026-10-20 и координатора о v1 search):
func SearchDocs(m Meta, items []plane.WorkItem) []search.Doc // C9 v3: только ID, Key "<Identifier>-<Sequence>", Title (описание не грузить)
func (a *App) ServerSearch(ctx, query, projectID string) ([]plane.SearchHit, error) // plane.Search, limit 50; projectID пусто — весь workspace
// В TUI не используется (поиск — только по заголовку локально, решение 2026-10-20); остаётся для `platui search --server`.
func ParseRef(s, defaultProject string) (identifier string, seq int, ok bool) // "PROJ-42", "proj-42", "42" (с defaultProject)
func (a *App) CurrentItem(ctx) (plane.Project, plane.WorkItem, error) // Git.Current + gitbranch.ParseBranch(cfg.Branch.Template)

// v2 (2026-10-20, PT-20; запрос PT-10 и нужды TUI): одиночные операции над задачей.
func (a *App) Item(ctx, projectID, itemID string) (plane.WorkItem, error)                 // из сети; заменяет задачу в памяти App (и в кэше items, если он загружен)
func (a *App) Update(ctx, projectID, itemID string, p plane.WorkItemPatch) (plane.WorkItem, error) // UpdateWorkItem + замена в памяти/кэше
func (a *App) Comments(ctx, projectID, itemID string) ([]plane.Comment, error)            // все страницы, по CreatedAt

// Фильтры config v2 (семантика — C1 v2).
func Match(f config.Filter, it plane.WorkItem, m Meta, meID string) bool

// Быстрое создание: text.ParseQuick → "me"→Me().ID, @x → участник по DisplayName, затем по Email до "@" (без учёта регистра; неоднозначно или нет → ErrUnknownMember), #x → метка по имени (→ ErrUnknownLabel); Priority "" → не отправляется.
func (a *App) CreateQuick(ctx, projectID, line, descMarkdown string) (plane.WorkItem, error)
func (a *App) Comment(ctx, projectID, itemID, markdown string) (plane.Comment, error)      // text.MarkdownToHTML
func (a *App) SetDescription(ctx, projectID, itemID, markdown string) (plane.WorkItem, error)

// Массовые действия: по одной задаче последовательно; итог — ошибка по ID (nil — успех); отмена ctx — остаток с ctx.Err().
func (a *App) BulkPatch(ctx, projectID string, ids []string, p plane.WorkItemPatch) map[string]error
func (a *App) BulkAssign(ctx, projectID string, items []plane.WorkItem, memberID string, add bool) map[string]error
func (a *App) BulkLabel(ctx, projectID string, items []plane.WorkItem, labelID string, add bool) map[string]error

// Ветки.
func (a *App) BranchName(p plane.Project, it plane.WorkItem) (string, error) // gitbranch.Render(cfg.Branch.Template, …)
func (a *App) BranchStatuses(p plane.Project, items []plane.WorkItem) (map[string]gitbranch.BranchStatus, error) // ключ — item ID; имя по шаблону, плюс локальные ветки, у которых ParseBranch даёт тот же ID
type Stale struct{ Branch string; Item plane.WorkItem }
func (a *App) StaleBranches(ctx, p plane.Project) ([]Stale, error) // локальные ветки задач проекта в группах completed|cancelled, кроме текущей
func (a *App) DeleteBranches(names []string, force bool) map[string]error

// v4 (2026-10-20, клавиша b): только ветка, без изменений в Plane.
// name пусто — BranchName; worktree nil — cfg.Branch.Worktree; ErrBranchExists → Switch только при switchExisting, иначе — Result{Name, Created:false} и ErrBranchExists; Git nil → ErrNoRepo.
func (a *App) Branch(p plane.Project, it plane.WorkItem, name string, worktree *bool, switchExisting bool) (gitbranch.Result, error)

// «Старт работы».
type StartOptions struct{ Branch string; Worktree *bool } // Branch пусто — BranchName; Worktree nil — cfg.Branch.Worktree
type StartResult struct {
    Branch    gitbranch.Result
    State     string   // имя нового состояния; "" — не менялось
    Assigned  bool
    Warnings  []string // напр. чужой хук prepare-commit-msg, грязное дерево
}
func (a *App) Start(ctx, p plane.Project, it plane.WorkItem, o StartOptions) (StartResult, error)
```
v5 (2026-10-20, подтверждение деструктивных): в шаге (2) `Start` сообщение коммита пишется только если файла для ветки нет (`gitbranch.ReadCommitMsg` → ok=false) или его текст совпадает; иначе не перезаписывается и Warning «сообщение коммита для <ветки> уже есть — не перезаписано».
`Start`: (1) git: worktree → `AddWorktree(name, gitbranch.WorktreePath(cfg.Branch.WorktreeDir, top, name), {Base, Checkout})`, иначе `Create`; `ErrBranchExists` → `Switch` (с `cfg.Branch.Checkout`); ошибка git — возврат без изменений в Plane; (2) `WriteCommitMsg(name, CommitMessage(cfg.Branch.CommitTemplate, …))` и `InstallHook()`; `ErrHookExists` → Warning; (3) один `UpdateWorkItem`: состояние — `cfg.Workflow.StartState` по имени (без учёта регистра), иначе первое из `Meta.States` с группой `started`; не найдено или уже такое — не отправлять; исполнители — добавить `Me().ID`, если нет; патч пуст — запроса нет. Ошибка API после git — `StartResult` с заполненным `Branch` и ошибка.

```go
// «Готово к ревью».
type ReviewResult struct{ MRURL, State string; Warnings []string }
func (a *App) Review(ctx, p plane.Project, it plane.WorkItem) (ReviewResult, error)
```
`Review` (v3, только GitLab): ветка — `Git.Current()` (detached → ошибка); `Push(cfg.Branch.Remote, branch)`; `Forge.CreateMR{Dir: Toplevel, Head: branch, Target: cfg.GitLab.TargetBranch, Title: CommitMessage(cfg.Branch.CommitTemplate, …), Body: "<WebURL задачи>", Draft: cfg.GitLab.Draft}`; URL есть → `CreateComment` с HTML `<p>MR: <a href="URL">URL</a></p>`; `ReviewResult.PRURL` переименовать в `MRURL`; состояние — `cfg.Workflow.ReviewState` по имени, иначе «In Review» без учёта регистра; не найдено → Warning. Ошибка push — возврат сразу; ошибка MR (в т.ч. `forge.ErrNotFound`) — Warning и продолжение (без комментария).

```go
// v3 «Go to MR» (решение пользователя 2026-10-20): MR задачи через glab.
// Ветки: BranchName(p, it); текущая ветка, если ParseBranch даёт ту же задачу; локальные ветки с тем же ID (как в BranchStatuses). key — "<Identifier>-<Sequence>".
// Кэш: Store "<workspace>/mrs/<itemID>" на cfg.GitLab.MRCache (0 — без кэша); fresh=true — в обход кэша.
func (a *App) MRs(ctx, p plane.Project, it plane.WorkItem, fresh bool) ([]forge.MR, error) // Git nil → ветки только BranchName; Forge ErrNotFound — ошибка
func (a *App) CachedMRs(itemID string) ([]forge.MR, bool)                                  // только кэш/память, без glab — для признака MR в таблице/превью

// Учёт времени: состояние в Store под ключом "<workspace>/timers" (map item ID → {project_id, started}); без Store — только память.
type Timer struct{ ProjectID, ItemID string; Started time.Time }
func (a *App) StartTimer(projectID, itemID string) error              // уже идёт — no-op
func (a *App) Timers() []Timer                                        // по Started
func (a *App) StopTimer(ctx, projectID, itemID, note string) (time.Duration, error) // нет → ErrNoTimer; длительность округлена до минуты вверх, ≥ 1m; комментарий "<p>Затрачено: 1h 25m</p>" + MarkdownToHTML(note); таймер удаляется только после успешного комментария
func FormatDuration(d time.Duration) string                           // "1h 25m", "45m"

// Строка для промпта/tmux: только кэш и git, без сети; нет данных — "".
func (a *App) Prompt() string // "PROJ-42 In Progress" (имя состояния — из кэша Meta "<workspace>/meta/<projectID>", если есть; иначе только ID)

// Копирование.
type CopyItems struct{ ID, URL, Title, Branch, Markdown string } // Markdown — "[PROJ-42](url)"
func (a *App) CopyItems(p plane.Project, it plane.WorkItem) CopyItems
```
Meta сохраняется в Store (`"<workspace>/meta/<projectID>"`) при каждом `Meta` из сети — для `Prompt`.
Фейки зависимостей в тестах: `planetest.Fake`, `gitbranchtest.Fake`, `forgetest.Fake`, `cache.NewMem`. Контрактный тест: `internal/app/contract_test.go`, `TestContractApp*`: Reload (обходит кэш, ошибка не портит прежние данные), StateTabs (конфиг-порядок, умолчание с hide_groups, неизвестная группа, Sequence), SearchDocs, Find в другом проекте через WorkItemByKey, ServerSearch; Start (новая ветка, существующая → Switch, worktree, старт-состояние по имени и по группе, уже назначен, чужой хук → Warning, ошибка git → нет вызовов UpdateWorkItem), Review (push, CreateMR с Target/Draft из cfg.GitLab, комментарий «MR:», review-состояние, ошибка MR → Warning); v3: MRs (ветки задачи + key → forgetest.Finds, кэш и fresh, CachedMRs без вызовов), CreateQuick (me, @, #, неизвестные), Diff, NextPoll, RefreshItems с кэшем, Match по каждому критерию, StaleBranches, Bulk*, таймер (Start/Stop/комментарий/персистентность через Mem), Prompt без сети, ParseRef, CurrentItem.

## C9. search v4 — пакет internal/search (PT-15)

Мгновенный нечёткий поиск по мере ввода **только по заголовку задачи** (решения пользователя 2026-10-20: поиск — ключевая фича; пересмотр — только заголовок, без описания, меток, исполнителей; ответ на Q1 — запрос ≤ 32 рун). Запрос вида `PROJ-42` или номер — не нечёткий поиск, а точный переход к задаче. Листовой пакет (не импортирует `platui/internal/...`); документы собирает app (C5 `SearchDocs`). Потребители: app, tui (поиск `/`, палитра PT-5e — тот же `Fuzzy`), cli.

```go
const MaxQuery = 32                  // руны; Search и ParseKey обрезают запрос до 32 рун
type Doc struct {
    ID    string // ID задачи (UUID)
    Key   string // "PROJ-42"
    Title string
}
type Hit struct {
    ID    string
    Score int
    Pos   []int // позиции совпавших рун в Title (для подсветки), по возрастанию, без повторов
}
type Index struct{ … }               // неизменяемый после New; безопасен для чтения из горутин
func New(docs []Doc) *Index          // заранее: Title → []rune в нижнем регистре (unicode), границы слов
func (ix *Index) Search(q string) []Hit // пустой/пробельный q или ParseKey(q) ok → nil
func (ix *Index) Len() int
func ParseKey(q string) (identifier string, seq int, ok bool) // "PROJ-42", "proj-42" → ("PROJ", 42); "42", "#42" → ("", 42); края обрезаются; иначе ok=false
func (ix *Index) Lookup(identifier string, seq int) (id string, ok bool) // точное совпадение Key (identifier без учёта регистра); identifier "" — первый doc с Key, оканчивающимся на "-<seq>"
func Fuzzy(pattern, s string) (score int, pos []int, ok bool)
func Highlight(s string, pos []int, mark func(string) string) string // подряд идущие совпавшие руны — в один mark
```
Правила `Search`: запрос обрезается до `MaxQuery` рун, → нижний регистр (unicode), слова по пробелам; задача подходит, если **каждое** слово нечётко совпало с `Title` (И); `Score` — сумма оценок слов; `Pos` — объединение позиций слов; сортировка — `Score` по убыванию, при равенстве — порядок `docs`. Описание, метки, исполнители в индекс не попадают.
`Fuzzy` (v4, решение пользователя 2026-10-20): **тонкая обёртка над `github.com/junegunn/fzf/src/algo`** — `algo.FuzzyMatchV2(caseSensitive=false, normalize=true, forward=true, &util.Chars из s, шаблон в нижнем регистре []rune, withPos=true, slab)`; `score` — `Result.Score`, `pos` — позиции из fzf (руны, по возрастанию), `ok` — `Result.Start >= 0`. Свой алгоритм сопоставления не писать; бонусы и ранжирование — как у fzf (схема `algo.Init("default")`, если в выбранной версии есть `Init`). Слэб — `util.MakeSlab` из `sync.Pool` (Index безопасен для параллельного чтения). Версия модуля `github.com/junegunn/fzf` — последняя, у которой `go` в go.mod ≤ 1.24 (исключение 2026-10-20: v0.74.2 — с v0.74.3…v0.74.5 `go test -cover` на go1.24.2 паникует, подтверждено PT-15 и ревью; пересмотреть при обновлении Go); шаблон перед `FuzzyMatchV2` — `algo.NormalizeRunes` от нижнего регистра (как в fzf pattern.go), иначе при `normalize=true` текст с диакритикой не совпадает сам с собой; нет такой или пакет `src/algo` недоступен для импорта — шлюз (строка `PT-15 blocked: …`), не обходить своим кодом.
Производительность (порог приёмки): `BenchmarkSearch5000` — 5000 документов (заголовки 3–10 слов из словаря ~200 слов латиницей и кириллицей, seed фиксирован), запрос из 2 слов — ≤ 10 мс/op; `BenchmarkNew5000` — ≤ 50 мс/op; `BenchmarkFuzzyLong3000`/`9000` (текст 3000/9000 рун без подстроки шаблона, разреженное совпадение, шаблон 32 руны) — ≤ 2 мс/op.
Контрактный тест: `internal/search/contract_test.go`, `TestContractSearch*`: И по словам; подряд идущее совпадение выше разреженного и начало слова выше середины (как ранжирует fzf); `Fuzzy` совпадает с прямым вызовом `algo.FuzzyMatchV2` на таблице примеров (оценка и позиции); кириллица и регистр; точные `Pos` (сравнение срезов); `Highlight`; пустой запрос; `ParseKey` (PROJ-42, proj-42, 42, #42, мусор) и что `Search` для них → nil; `Lookup` с identifier и без; `MaxQuery` (40 рун ≡ первые 32); стабильность порядка при равных оценках.

## C10. cli v4 — пакет internal/cli и cmd/platui (PT-10; v2 — подтверждение деструктивных действий; v3 — todo; v4 — коды выхода; дельта в docs/contracts/tui-features.md)

Режим без интерфейса и сборка бинаря. `internal/cli` → app, config, plane, gitbranch, text, search (не импортирует `tui`); `cmd/platui/main.go` — только сборка зависимостей (C0).

```go
type Env struct {
    Stdout, Stderr io.Writer
    Getenv  func(string) string
    Getwd   func() (string, error)
    Build   func(cfg *config.Config, repoDir string) (*app.App, error) // main: plane.New, gitbranch.Open (ErrNotRepo → Git nil), cache.NewFileStore(cfg.CacheDir()) если CacheEnabled, app.New
    RunTUI  func(a *app.App, cfg *config.Config, start StartAt) error  // main: tui.Run(tui.Deps{…})
}
type StartAt struct{ Project, Item string } // main переводит в tui.StartAt
func Run(ctx context.Context, args []string, env Env) int // код выхода
```
Глобальные флаги (до подкоманды): `--config <путь>` (иначе `config.DefaultPath()`), `--repo <каталог>` (иначе cwd), `--project <IDENT>` (иначе `default_project`). Без подкоманды — `RunTUI` (`--project`, `--item PROJ-42` → `StartAt`).

| Подкоманда | Поведение |
|---|---|
| `ls [--mine] [--state <имя>] [--all] [--json]` | задачи проекта; по умолчанию — без групп `cfg.Tabs.HideGroups`, `--all` — все; порядок — `StateTabs`, затем ID; текст: `PROJ-42  In Progress  high  Название` (колонки выровнены); JSON — массив объектов `key,id,name,state,state_group,priority,assignees[имена],labels[имена],url,updated_at` |
| `show <ref> [--json] [--comments]` | карточка: ключ, название, состояние, приоритет, исполнители, метки, URL, описание (`HTMLToText`), комментарии; `ref` — `PROJ-42` или `42` (`app.ParseRef`, `app.Find`) |
| `search <запрос…> [--json] [--server]` | по заголовкам: локально (`RefreshItems`/кэш проекта → `SearchDocs` + `search.New`); запрос-ключ (`search.ParseKey`) → как `show`; `--server` — `ServerSearch` по workspace (сервер ищет и по описанию — помечено в справке); вывод как `ls` |
| `branch [<ref>] [--print] [--worktree]` | имя по шаблону; `--print` — только напечатать; иначе создать/переключиться (`Create`, при `ErrBranchExists` — `Switch`; `--worktree` — `AddWorktree`); без `ref` — ошибка использования |
| `start <ref> [--worktree]` | `app.Start`; вывод: ветка, состояние, назначение, предупреждения |
| `review [<ref>]` | `app.Review`; без `ref` — задача текущей ветки (`CurrentItem`); вывод: URL MR, состояние, предупреждения |
| `mr [<ref>] [--open] [--json]` | v3 «Go to MR»: `app.MRs(fresh=true)`; без `ref` — задача текущей ветки; вывод: `!IID  state  title  url` по строке; `--open` — один MR → открыть в браузере (`github.com/cli/browser`), несколько — список и код 0 без открытия (выбор — в TUI), нет — «MR не найдены», код 1 |
| `prompt [--format <шаблон>]` | `app.Prompt()` без сети (по умолчанию `{key} {state}`; плейсхолдеры `{key}`, `{state}`, `{title}`); нет данных или любая ошибка (в т.ч. загрузки конфига) — пустой вывод, код 0 всегда, ничего в stderr; сеть не используется |
| `hook install` | `InstallHook`; `ErrHookExists` — сообщение и код 1 |
| `version` | `platui <версия>` (`var Version = "dev"`, задаётся `-ldflags`) |

Деструктивные действия (v2, решение пользователя 2026-10-20; список — C4 v5 «Подтверждение»): только после вопроса в терминале `… [y/N]` (по умолчанию «нет»); stdin не TTY — отказ (код 1, «нужно подтверждение: --yes») без глобального флага `--yes`. Сейчас деструктивных подкоманд нет: `start` не перезаписывает существующее сообщение коммита (C5 v5), `hook install` не трогает чужой хук; новая деструктивная подкоманда — только с этим правилом и тестом (`TestContractCLIConfirm*`). Глобальный флаг `--yes` и хелпер `confirm(env, question) (bool, error)` (Env + `Stdin io.Reader`, `IsTTY func() bool`) — добавить сейчас (PT-30), чтобы правило было исполняемым.
Коды выхода: 0 — успех; 1 — ошибка выполнения (`platui: <текст>` в stderr); 2 — ошибка использования (неизвестная подкоманда/флаг, нет аргумента; краткая справка в stderr). `-h`/`--help` у корня и подкоманд — справка в stdout, код 0. Разбор — `github.com/spf13/cobra` v1.10.2 (ответ пользователя на Q3; флаги допускаются после аргументов: `show PROJ-42 --json`); `SilenceUsage`/`SilenceErrors`, коды выхода — как выше (ошибки разбора флагов/аргументов → 2, через `FlagErrorFunc` и `Args`-валидаторы); автодополнение cobra (`completion`) — оставить встроенным.
Контрактный тест: `internal/cli/contract_test.go`, `TestContractCLI*`: каждая подкоманда на фейках (`Build` возвращает `app.New` на planetest/gitbranchtest/forgetest/cache.Mem), JSON-поля `ls`/`show`, коды 0/1/2, `prompt` без сети и без ключа, без подкоманды вызывается `RunTUI` с `StartAt`. `cmd/platui`: `go build` и `go vet`; тест `TestMainWiring` не нужен.
