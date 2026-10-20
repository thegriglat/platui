---
type: "contract"
status: "closed"
module: "tui"
updated: "2026-10-20"
summary: "Вторая волна фич tui: diff stat, свои команды, закладки, дубли, ссылки, мышь, панели, подзадачи; C11 runner"
related: ["docs/contracts/tui.md", "docs/contracts/tui-plane.md", "docs/contracts/tui-services.md", "docs/contracts/tui-ui.md", "docs/plan/tui.md"]
contracts: [{"id": "C11", "version": 1}, {"id": "C12", "version": 1}]
---

# Вторая волна фич модуля tui

Решение пользователя 2026-10-20 (план §2, §4.1): фичи (4) diff stat ветки, (5) свои команды, (7) история и закладки, (8) поиск дублей при создании, (10) ссылки из описания и комментариев, (13) мышь, (14) размер панелей, (15) подзадачи деревом. Здесь — дельты к C1, C2, C3, C4, C5, C6 (их новые версии) и новый C11. Владелец — координатор. Структура — C0, деструктивное — по C4 v5 «Подтверждение».

## C11. runner v1 — пакет internal/runner (PT-35)

Листовой пакет: запуск shell-команды пользователя (свои команды, фича 5), чтобы `app` не импортировал `os/exec` (C0).
```go
type Result struct{ Stdout, Stderr string; ExitCode int }
type Shell func(ctx context.Context, dir, command string, env []string) (Result, error) // sh -c command; ошибка — только если не удалось запустить; ненулевой код — в ExitCode
func Default() Shell // os/exec, sh -c, cwd=dir, env = os.Environ()+env, ctx отменяет
```
Тест `TestContractRunner*`: вывод, код выхода, env, отмена ctx.

## Дельты к существующим контрактам

### C1 v5 — config: `ui`, `commands` (PT-32)
```yaml
ui:
  mouse: true          # по умолчанию true; false — без мыши (мешает выделению текста в терминале)
  preview_width: 45    # % ширины под превью при ≥ 100 колонок; 20..80; по умолчанию 45
commands:              # свои команды (как custom commands lazygit), необязательно
  - name: "Открыть в IDE"   # обязательно, уникально
    key: "ctrl+e"           # необязательно; формат tea.KeyMsg.String()
    command: "code {branch}"  # обязательно; плейсхолдеры {id} {key} {branch} {url} {title} {mr_url}; прочие {…} — ошибка
    confirm: false          # true — диалог y/N перед запуском (C4 v5)
    output: status          # status (последняя непустая строка stdout/stderr в статусе) | window (окно с выводом) | none; по умолчанию status
```
```go
type UI struct { Mouse *bool `yaml:"mouse"`; PreviewWidth int `yaml:"preview_width"` }   // nil → true; 0 → 45
type Command struct { Name, Key, Command string; Confirm bool; Output string }          // yaml: name,key,command,confirm,output
// Config v4 + UI UI `yaml:"ui"` + Commands []Command `yaml:"commands"`
func (c *Config) MouseEnabled() bool
func (c *Config) CheckCommandKeys(reserved []string) error // ключ своей команды совпал со встроенным (tui.BuiltinKeys) — ошибка с именем команды и клавишей
```
Ошибки: `preview_width` вне 20..80; пустые `name`/`command`; повтор `name` или `key`; `output` вне трёх; неизвестный плейсхолдер. Плейсхолдер внутри кавычек (`'{title}'`, `"{branch}"`) — ошибка конфига: значения подставляются уже экранированными одинарными кавычками (`RenderCommand`), кавычки вокруг ломают экранирование (ревью PT-35, 2026-10-20). Тест `TestContractConfigV5*`.

### C2 v5 — gitbranch: diff stat (PT-33)
```go
type Stat struct{ Files, Insertions, Deletions int }
func (r *Repo) DiffStat(base, branch string) (Stat, error) // git diff --shortstat <base>...<branch>; base пусто — как в Statuses (remote HEAD → main → master); пустой вывод — Stat{}; ветки нет — ошибка
```
В интерфейс `Git` и `gitbranchtest.Fake` (`Stats map[string]Stat`). Тест `TestContractGitbranchV5*` на временном репозитории (2 файла, +N/−M строк).

### C3 v6 — plane: родитель задачи (PT-31)
`WorkItem.ParentID string` — из `parent` (строка-UUID или объект `{id}`), иначе `parent_id`; нет/null → "". `NewWorkItem.ParentID` (POST `parent`, пусто — не отправлять). Fake хранит как есть. Тест `TestContractPlaneV6*`.

### C6 v2 — text: ссылки (PT-34)
```go
func ExtractLinks(html string) []string // href из <a> + голые URL из текста (mvdan.cc/xurls/v2, Strict); http/https, без дублей, в порядке появления
```
Библиотека `mvdan.cc/xurls/v2` (последняя с go ≤ 1.24.2; обоснование — критерии C0, в отчёте). Тест `TestContractTextLinks*`.

### C5 v6 — app (PT-35)
```go
// Deps v5 + Shell runner.Shell // nil → runner.Default()
func (a *App) DiffStat(p plane.Project, it plane.WorkItem) (gitbranch.Stat, bool, error) // ветка задачи (как BranchStatuses) есть локально → stat относительно cfg.Branch.Base; нет ветки/Git nil → ok=false
type CommandVars struct{ ID, Key, Branch, URL, Title, MRURL string }
func (a *App) CommandVars(p plane.Project, it plane.WorkItem) CommandVars // MRURL — из CachedMRs (первый), иначе ""
func RenderCommand(tmpl string, v CommandVars) string                      // значения экранируются для sh (одинарные кавычки)
func (a *App) RunCommand(ctx context.Context, c config.Command, v CommandVars) (runner.Result, error) // cwd — Toplevel (Git nil → текущий каталог)
// Закладки (фича 7): Store "<workspace>/bookmarks/<projectID>" — []string item ID в порядке добавления.
func (a *App) Bookmarks(projectID string) []string
func (a *App) ToggleBookmark(projectID, itemID string) (on bool, err error)
// Подзадачи (фича 15): дерево по ParentID внутри переданного списка; родитель вне списка — задача корневая.
type Node struct{ Item plane.WorkItem; Children []*Node; Depth int }
func Tree(items []plane.WorkItem) []*Node // порядок — как во входе; циклы разрываются (задача на цикле — корень)
```
Тест `TestContractAppV6*`.

### C4 v6 — tui (PT-36…PT-40)
Клавиши (дополнение таблицы «Клавиши» C4; vim-клавиш нет):

| Группа | Клавиши |
|---|---|
| задача | `B` закладка вкл/выкл, `l` ссылки из описания и комментариев (список, Enter — в браузер; решение 2026-10-20: `l`, не `u`) |
| навигация | `Ctrl+O` — назад к предыдущей открытой задаче (история в памяти, до 50), `→`/`←` в таблице — раскрыть/свернуть подзадачи |
| виды | `<`/`>` — ширина превью −/+5 % (20..80), `z` — превью на весь экран и обратно |
| свои | клавиши из `cfg.Commands` (и все команды — в палитре `Ctrl+P` под названием) |

- (4) Превью: строка «Ветка: +ins −del (n файлов)» по `app.DiffStat` (загрузка асинхронно при показе задачи, кэш в сессии на itemID+UpdatedAt; нет ветки — строки нет). Списка коммитов нет.
- (5) Своя команда: `app.CommandVars` → `confirm` при `Confirm: true` → `app.RunCommand`; `output`: status — последняя непустая строка в статусе (код ≠ 0 — красным), window — экран с выводом (viewport, Esc), none — только «готово/ошибка». `func BuiltinKeys() []string` — экспорт: все клавиши реестра; `cmd/platui` вызывает `cfg.CheckCommandKeys(tui.BuiltinKeys())` до запуска TUI.
- (7) Закладки — наверху таблицы текущего таба (☆ перед ID), в порядке добавления, затем остальные по сортировке; поиск — закладки не закрепляются.
- (8) Диалог `n`: под строкой ввода — до 5 похожих задач проекта (`search` по заголовку введённого названия без токенов `!@#`), по мере ввода; Enter по-прежнему создаёт; `↓` — перейти к списку, Enter на похожей — открыть её вместо создания.
- (10) `l`: `text.ExtractLinks(DescHTML)` + из всех комментариев; пусто — статус «ссылок нет».
- (13) Мышь (`cfg.MouseEnabled()` → `tea.WithMouseCellMotion()` в `Run`): клик по строке таблицы — курсор, двойной клик — карточка; клик по табу — таб; клик по панели — фокус; колёсико — прокрутка таблицы/превью под указателем.
- (14) Ширина превью — `cfg.UI.PreviewWidth`, меняется `<`/`>` на сессию; `z` — превью на весь экран.
- (15) Подзадачи: таблица строит `app.Tree` по видимым задачам; у родителя `▸ (n)`/`▾`, дети с отступом по `Depth`; по умолчанию свёрнуто; при активном поиске — плоский список.
Тесты: `TestContractTUIV6<Фича>*` на каждую фичу; мышь — `tea.MouseMsg` в `Update`.

## C12. todo v1 — пакет internal/todo (PT-41; решение пользователя 2026-10-20 «личный todo»)

Листовой пакет, без сети. Файл Markdown с чекбоксами; ручная правка файла — норма: чужие строки (заголовки, текст, пустые) и их порядок сохраняются.
```go
var itemRe = `^(\s*)[-*+] \[( |x|X)\] (.*)$` // строка-пункт; всё прочее — чужая строка
type Item struct { N int; Text string; Done bool } // N — номер пункта с 1 в порядке файла
type List struct{ … }                              // строки файла + индексы пунктов + отпечаток содержимого при Load
var ErrChanged = errors.New("todo file changed on disk")
var ErrNoItem  = errors.New("no such todo item")
func DefaultPath() (string, error)       // $XDG_DATA_HOME/platui/todo.md, иначе ~/.local/share/platui/todo.md
func Load(path string) (*List, error)    // нет файла — пустой список, без ошибки
func (l *List) Items() []Item
func (l *List) Add(text string) Item     // "- [ ] text" сразу после последнего пункта, нет пунктов — в конец файла
func (l *List) SetDone(n int, done bool) error // "[x]"/"[ ]", остальное в строке не трогается
func (l *List) Edit(n int, text string) error
func (l *List) Remove(n int) error       // вызывающий спрашивает подтверждение (C4 v5)
func (l *List) Move(n, to int) error     // переставить пункт на место пункта to; чужие строки остаются на местах
func (l *List) Save() error              // файл изменился с Load (отпечаток) → ErrChanged, ничего не пишется; иначе атомарно (renameio), 0644, каталог 0700; новый отпечаток
func Keys(text string) []string          // ссылки на задачи: (?:^|\W)([A-Z][A-Z0-9]*-[0-9]+)\b, по порядку, без дублей
```
Тест `TestContractTodo*`: разбор (`*`/`+`/`-`, `[X]`, отступ), чужие строки и порядок после Add/Edit/Remove/Move/Save, ErrChanged при правке файла между Load и Save, нет файла, Keys.

### C1 v6 — config: `todo` (PT-41)
```yaml
todo:
  path: ""   # пусто — todo.DefaultPath()
```
`type Todo struct{ Path string \`yaml:"path"\` }`; `func (c *Config) TodoPath() (string, error)`. Тест `TestContractConfigV6*`.

### C4 v7 — tui: вид Todo (PT-42)
`T` — экран Todo (Scope "todo"; `Esc`/`q` — назад). Клавиши в нём: `↑/↓`, `n` — новый пункт строкой, `space` — выполнен/нет, `e` — правка строкой, `d` — удалить (confirm), `Shift+↑/↓` — переставить, `C` — сделать задачей Plane (`app.CreateQuick` с текстом пункта, затем к тексту пункта дописывается ` (PROJ-N)`), `Enter` на пункте с ключом (`todo.Keys`) — `app.Find` → карточка. Каждое изменение — `Save`; `ErrChanged` → статус «todo.md изменён вне platui — перечитан», `Load` заново, изменение не применено. Тест `TestContractTUIV7Todo*` на временном файле.

### C10 v3 — cli: `todo` (PT-43)
`platui todo [ls]` — пункты `N. [x] текст`; `platui todo add <текст…>`; `platui todo done <N>`; `platui todo undone <N>`; путь — `cfg.TodoPath()`; конфиг без ключа API допустим для `todo` (ошибку ключа игнорировать, как не требующую сети). `ErrChanged` — повторить один раз (Load+применить+Save), снова — ошибка. Тест `TestContractCLITodo*`.

## Устойчивость к недоступности API (решение пользователя 2026-10-20; аудит PT-45 — docs/research/api-failures.md)

### C3 v7 — plane (PT-46; GAP-5, 6, 7)
- `Retry-After` — секунды **или** HTTP-date (`http.ParseTime`), не больше 60 s, как в v3.
- Проверка ответов: 200 без `results` у списочного эндпоинта (и не массив) → `APIError{Status:200, Message:"unexpected response"}`; объект без `id` там, где ждём задачу/участника → такая же ошибка; ошибки `Me` — с префиксом `plane:`, как остальные.
- Текст ошибки из тела ответа: ключ клиента (`Options.APIKey`) вырезается (`***`) перед вставкой в `APIError.Message`.
Тест `TestContractPlaneV7*`; GAP-тесты PT-45 `TestFailurePlane*` должны пройти без Skip.

### C1 v7 — config: `http` (PT-47; GAP-2)
```yaml
http:
  timeout: 30s   # таймаут одного запроса; по умолчанию 30s; 1s..5m
  retries: 3     # повторы 429/503; по умолчанию 3; 0..10
```
`type HTTP struct{ Timeout *time.Duration; Retries *int }` (nil → умолчания); `cmd/platui` передаёт в `plane.Options{HTTPClient: &http.Client{Timeout}, MaxRetries}`. `plane.Options.MaxRetries`: 0 → 3 (умолчание, как в v2), **< 0 → без повторов**; `retries: 0` в конфиге → `MaxRetries: -1` (C3 v7, отчёт PT-47). Тест `TestContractConfigV7*`.

### C10 v4 — cli: коды выхода (PT-48; GAP-3)
0 — успех; 1 — прочая ошибка; 2 — использование; **3 — авторизация** (`plane.ErrUnauthorized`: «ключ API отклонён — проверьте api_key/api_key_env»); **4 — сеть/таймаут/5xx/429 после повторов** (`url.Error`, `net.Error`, `context.DeadlineExceeded`, `io.ErrUnexpectedEOF` и обрыв соединения, ошибки TLS/x509, `APIError.Status >= 500`, `ErrRateLimited`: «Plane недоступен: …»). Ключ — никогда в выводе. Тест `TestContractCLIExit*`.

### C5 v7 — app: офлайн (PT-49; GAP-1 app, GAP-4)
- `Me`, `Projects`, `Meta`: при сетевой ошибке (как в C10 v4, код 4-класса) — данные из Store (`<ws>/me`, `<ws>/projects`, `<ws>/meta/<pid>`; пишутся при каждом успешном получении), если есть; ошибка авторизации — не подменяется кэшем.
- `func (a *App) Offline() (bool, time.Time)` — true, если последняя попытка сети не удалась и данные отданы из кэша; время — `saved` самого старого из отданных кэшей.
- `Find` при сетевой ошибке — поиск по кэшу проектов/задач.
- GAP-4: `replaceItem` (Update/Item) не меняет время кэша списка задач (сохраняет прежнее `saved`).
Тест `TestContractAppV7*`.

### C4 v8 — tui: офлайн и восстановление (PT-50; GAP-1 tui, 8, 9)
- Старт без сети: `bootstrap` не падает на `Me`/`Projects`; при `app.Offline()` — статусная строка «офлайн, данные от ЧЧ:ММ» (постоянная, жёлтым), пока не будет успешного обновления.
- Успешный опрос/`R` после ошибки — снять ошибку, статус «связь восстановлена» (один раз); фоновые ошибки не повторяются в статусе чаще одного раза на интервал опроса (`NextPoll`).
- `$EDITOR`: при ошибке отправки описания/комментария текст не теряется — файл `EditSession` не удаляется, статус «не отправлено, текст сохранён: <путь>»; повтор той же клавишей предлагает «использовать сохранённый текст? y/N».
Тест `TestContractTUIV8*`; GAP-тесты PT-45 `TestFailureTUI*` — без Skip.
