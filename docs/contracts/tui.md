---
type: "contract"
status: "closed"
module: "tui"
updated: "2026-10-20"
summary: "Стыки модуля tui — формат YAML-конфига, API пакетов config, gitbranch, plane (Go-интерфейсы), зависимости TUI"
related: ["docs/plan/tui.md", "docs/contracts/tui-services.md", "docs/contracts/tui-plane.md", "docs/contracts/tui-ui.md"]
contracts: [{"id": "C0", "version": 1}, {"id": "C1", "version": 7}, {"id": "C2", "version": 5}, {"id": "C3", "version": 4}, {"id": "C4", "version": 1}]
---

# Контракты модуля tui

Владелец всех стыков — координатор модуля tui. Меняет только он: версия +1, потребителям — `task note`.
Go-модуль `github.com/thegriglat/platui` (решение пользователя 2026-10-20; ранее `platui`), go 1.24.2, код — `internal/`, бинарь — `cmd/platui`, установка — `go install github.com/thegriglat/platui/cmd/platui@latest`.

## C0. Структура кода v1 (решение пользователя 2026-10-20: код — для удобной поддержки)

Обязательно для всех задач модуля; ревьюер проверяет при приёмке. Проверки карточек: `maxlines` и `imports` (ниже).

Слои и зависимости (стрелка — «может импортировать»; обратные и боковые импорты запрещены, циклов нет):
- листовые пакеты: `internal/config`, `internal/gitbranch`, `internal/plane` — не импортируют `platui/internal/...` вовсе;
- `internal/app` (сервисный слой, C5) → config, gitbranch, plane; внешние процессы и окружение — через интерфейсы/функции в опциях;
- `internal/tui` (C4) → app, config, plane, gitbranch (типы и интерфейсы; конкретные `*plane.Client`/`*gitbranch.Repo` не создаёт);
- `internal/cli` (PT-10) → app, config, plane, gitbranch (так же);
- `cmd/platui` — только сборка: разбор флагов, `config.Load`, создание конкретных реализаций, запуск tui/cli; без логики сценариев.

Правила:
- зависимости между слоями — через интерфейсы потребителя или контракта (`plane.API`, `gitbranch.Git`/`Brancher`, интерфейсы C5); у каждого интерфейса — проверка `var _ I = (*Impl)(nil)` рядом с реализацией;
- файл — одна ответственность (типы, транспорт, разбор, эндпоинты, рендер, …); не-тестовый `.go` ≤ 300 строк, функция ≤ 60 строк (кроме таблиц данных);
- тесты — рядом с кодом (`*_test.go` в том же каталоге); фейки интерфейсов для соседних пакетов — в `internal/<пакет>/<пакет>test` (напр. `planetest.Fake`), без зависимостей на тестовые пакеты в продакшен-коде;
- внешние процессы (`git`, `gh`, `glab`, `$EDITOR`), время, сон, переменные окружения, сеть — за интерфейсом или функцией в опциях, чтобы тест подменял; глобального изменяемого состояния нет;
- ошибки — `fmt.Errorf("…: %w", err)`; ожидаемые случаи — экспортируемые `Err…` (проверка `errors.Is`); паники только для ошибок программиста;
- у экспортируемых идентификаторов — doc-комментарий; `go vet` чист.
- зрелая библиотека вместо своего кода (решение пользователя 2026-10-20): есть поддерживаемая библиотека, совместимая с go 1.24 (`go` в её go.mod ≤ 1.24), — брать её; карточка называет библиотеку; свой код — только для предметной логики (сценарии, разбор строк быстрого создания, wire Plane). Реестр выбранных библиотек — `docs/research/libraries.md` (PT-18). Критерии (решение пользователя 2026-10-20): много пользователей (звёзды, импортёры pkg.go.dev), живая поддержка (релизы/коммиты за год, реакция на issues), больше одного активного мейнтейнера или организация, стабильный API (v1+ или известный проект), без тяжёлых транзитивных зависимостей; проект одного человека или заброшенный — не брать, оставить своё; обоснование (цифры, ссылки) — в отчёте задачи.

Проверки (координатор кладёт в карточку для пакетов задачи `<p>`):
- `maxlines`: `awk 'FNR==1{if(n>m)m=n;n=0}{n++}END{if(n>m)m=n;print "maxlines:",m}' $(find <p> -name '*.go' ! -name '*_test.go')` → `num:maxlines: ([0-9]+) <=300`;
- `imports` (листовые пакеты): `go list -f '{{join .Imports " "}}' ./<p>` (без `/...`: `<p>test` импортирует свой пакет) → `!re:platui/internal`.

## C1. config v7 — формат YAML-конфига и пакет internal/config (v5–v7 — дельты в docs/contracts/tui-features.md)

Владелец кода: PT-2. Потребители: PT-5 (`cmd/platui`, tui), PT-6 (`config.example.yml`, README).

Путь: флаг `--config <путь>`; иначе `$XDG_CONFIG_HOME/platui/config.yml`; `XDG_CONFIG_HOME` пуст — `~/.config/platui/config.yml`.

```yaml
base_url: https://api.plane.so   # необязательно; по умолчанию https://api.plane.so; без завершающего «/» (срезается)
api_key: plane_api_xxx           # ключ API; ИЛИ
api_key_env: PLANE_API_KEY       # имя переменной окружения с ключом
workspace: my-team               # slug workspace, обязательно
default_project: PROJ            # identifier проекта, необязательно
web_url: ""                      # необязательно: базовый URL веб-интерфейса; пусто — выводится из base_url (правило — в C3)
branch:
  template: "{identifier}-{sequence}-{slug}"  # по умолчанию
  base: ""                       # необязательно; пусто — от текущего HEAD
  checkout: true                 # по умолчанию true
```

Пакет `internal/config`:

```go
type Branch struct {
    Template string `yaml:"template"`
    Base     string `yaml:"base"`
    Checkout *bool  `yaml:"checkout"` // nil в файле → после Load всегда не nil (true)
}
type Config struct {
    BaseURL        string `yaml:"base_url"`
    APIKey         string `yaml:"api_key"`
    APIKeyEnv      string `yaml:"api_key_env"`
    Workspace      string `yaml:"workspace"`
    DefaultProject string `yaml:"default_project"`
    WebURL         string `yaml:"web_url"`
    Branch         Branch `yaml:"branch"`
}
const DefaultBaseURL = "https://api.plane.so"
const DefaultBranchTemplate = "{identifier}-{sequence}-{slug}"

func DefaultPath() (string, error)              // правило пути выше
func Load(path string) (*Config, error)         // чтение + Parse
func Parse(data []byte) (*Config, error)        // YAML → умолчания → ResolveKey → Validate
func (c *Config) Validate() error
func (c *Config) Key() string                   // итоговый ключ API (после Load не пуст)
func (c *Config) CheckoutEnabled() bool
```

Инварианты и ошибки:
- неизвестные ключи YAML — ошибка (строгий разбор, `KnownFields(true)`), в тексте — имя ключа;
- заданы и `api_key`, и `api_key_env` — ошибка; не задан ни один — ошибка;
- `api_key_env` задан, переменная пуста/не задана — ошибка с именем переменной;
- `workspace` пуст — ошибка; `base_url`/`web_url` не `http(s)://…` — ошибка;
- `branch.template` должен содержать хотя бы один из `{identifier}`, `{sequence}`, `{slug}`; неизвестный плейсхолдер `{…}` — ошибка;
- файла нет — ошибка, в тексте путь и подсказка про `--config`;
- ключ API никогда не попадает в текст ошибок.
- YAML-библиотека: `gopkg.in/yaml.v3`.

Контрактный тест (PT-2): `internal/config/contract_test.go` — `TestContractConfig*`: полный пример выше разбирается в ожидаемую структуру; умолчания; каждая ошибка из списка. PT-6: `TestContractConfigExample` парсит `config.example.yml` из корня репозитория.

### C1 v2 (2026-10-20, план §4.1, задача PT-7) — дополнения к v1

v1 сохраняется целиком: любой файл v1 разбирается в те же значения полей v1. `TestContractConfig*` v1 не меняются, кроме одного допустимого: в ожидаемую структуру тестов, сравнивающих `Config` целиком, добавляются умолчания v2 явными значениями-константами (не копированием из результата) (уточнение 2026-10-20, ревью PT-7). Потребители v2: PT-5x (tui), PT-6, PT-9 (app), PT-10 (CLI).

```yaml
# Поля подключения — как в v1, один workspace (несколько workspace/профилей — отложено, решение пользователя 2026-10-20).
branch:                          # v1 + новые поля
  worktree: false                # true — «старт работы»/`b` создают ветку в git worktree
  worktree_dir: "../{repo}-{branch}"   # по умолчанию; плейсхолдеры {repo}, {branch}; относительный — от корня репозитория
  commit_template: "{identifier}-{sequence}: {title}"  # по умолчанию; плейсхолдеры {identifier},{sequence},{title},{slug}
  remote: origin                 # по умолчанию
workflow:
  start_state: ""                # имя состояния для «старт работы»; пусто — первое состояние группы started
  review_state: ""               # имя состояния для «готово к ревью»; пусто — состояние с именем "In Review" (без учёта регистра), нет — не менять
pr:                               # УДАЛЕНО в v4 — см. «C1 v4» (секция gitlab)
  tool: auto                     # auto|gh|glab|none; по умолчанию auto (host remote содержит "gitlab" → glab, иначе gh)
  draft: false
  base: ""                       # целевая ветка PR; пусто — по умолчанию инструмента
filters:                         # ≤ 9; клавиши 1…9 в порядке списка
  - name: Мои в работе           # обязательно
    project: ""                  # identifier; пусто — текущий проект
    mine: true                   # исполнитель — я
    state_groups: [started]      # из backlog|unstarted|started|completed|cancelled
    states: []                   # имена состояний (без учёта регистра)
    priorities: [urgent, high]   # из urgent|high|medium|low|none
    labels: []                   # имена меток (без учёта регистра)
    text: ""                     # подстрока в названии (без учёта регистра)
theme:
  no_color: false                # true или непустая переменная NO_COLOR → без цветов
  colors:                        # необязательно, переопределяет умолчания по ключу
    accent: "#7D56F4"            # "#RRGGBB" или "0".."255" (ANSI 256)
editor: ""                       # пусто → $VISUAL → $EDITOR → "vi"
refresh:
  interval: 2m                   # по умолчанию 2m; 0 — выкл; иначе ≥ 30s (лимит API 60/мин)
  notify: true                   # по умолчанию true: подсветка и звонок о новых/изменённых задачах
cache:
  enabled: true                  # по умолчанию true
  dir: ""                        # пусто → $XDG_CACHE_HOME/platui, иначе ~/.cache/platui
```

```go
// Branch v1 + поля:
//   Worktree bool `yaml:"worktree"`; WorktreeDir string `yaml:"worktree_dir"`;
//   CommitTemplate string `yaml:"commit_template"`; Remote string `yaml:"remote"`
type Workflow struct{ StartState, ReviewState string } // yaml: start_state, review_state
type PR struct {
    Tool  string `yaml:"tool"` // auto|gh|glab|none
    Draft bool   `yaml:"draft"`
    Base  string `yaml:"base"`
}
type Filter struct {
    Name        string   `yaml:"name"`
    Project     string   `yaml:"project"`
    Mine        bool     `yaml:"mine"`
    StateGroups []string `yaml:"state_groups"`
    States      []string `yaml:"states"`
    Priorities  []string `yaml:"priorities"`
    Labels      []string `yaml:"labels"`
    Text        string   `yaml:"text"`
}
type Theme struct {
    NoColor bool              `yaml:"no_color"`
    Colors  map[string]string `yaml:"colors"`
}
type Refresh struct {
    Interval *time.Duration `yaml:"interval"` // nil в файле → после Load не nil (2m)
    Notify   *bool          `yaml:"notify"`   // nil → true
}
type Cache struct {
    Enabled *bool  `yaml:"enabled"` // nil → true
    Dir     string `yaml:"dir"`
}
// Config v1 + поля:
//   Workflow Workflow `yaml:"workflow"`; PR PR `yaml:"pr"`; Filters []Filter `yaml:"filters"`;
//   Theme Theme `yaml:"theme"`; Editor string `yaml:"editor"`; Refresh Refresh `yaml:"refresh"`; Cache Cache `yaml:"cache"`

const DefaultWorktreeDir = "../{repo}-{branch}"
const DefaultCommitTemplate = "{identifier}-{sequence}: {title}"
const DefaultRemote = "origin"
const DefaultRefresh = 2 * time.Minute
const MinRefresh = 30 * time.Second
var ColorKeys = []string{"accent", "muted", "text", "error", "success", "warning",
    "selection", "border", "priority_urgent", "priority_high", "priority_medium", "priority_low"}

func (c *Config) Color(key string) string        // из theme.colors, иначе умолчание пакета; неизвестный ключ — ""
func (c *Config) ColorsDisabled() bool           // theme.no_color || os.Getenv("NO_COLOR") != ""
func (c *Config) EditorCmd() string              // editor → $VISUAL → $EDITOR → "vi" (на момент вызова)
func (c *Config) CacheDir() (string, error)      // cache.dir → $XDG_CACHE_HOME/platui → ~/.cache/platui
func (c *Config) CacheEnabled() bool
func (c *Config) RefreshInterval() time.Duration // 0 — выкл
func (c *Config) NotifyEnabled() bool
```

Инварианты и ошибки v2 (в дополнение к v1; все — с именем ключа/фильтра в тексте, ключ API не в тексте):
- ключи `profile`/`profiles` в v2 не поддерживаются — ошибка неизвестного ключа (строгий разбор v1);
- `branch.worktree_dir`: допустимы только `{repo}`, `{branch}`; `branch.commit_template`: только `{identifier}`, `{sequence}`, `{title}`, `{slug}`, хотя бы один из них; неизвестный — ошибка;
- `pr.tool` вне `auto|gh|glab|none` — ошибка; пусто → `auto`;
- фильтров > 9, фильтр без `name`, неизвестная группа состояния или приоритет — ошибка;
- `theme.colors`: ключ не из `ColorKeys` — ошибка; значение не `#RRGGBB` (hex, любой регистр) и не целое 0..255 — ошибка;
- строгий разбор (неизвестный ключ — ошибка) действует на всех уровнях, в т.ч. внутри секций со своим `UnmarshalYAML` (`refresh` и др.);
- `refresh.interval` — длительность Go (`90s`, `2m`); `< 0` или `0 < x < 30s` — ошибка;
- семантика фильтра (применяют tui/app): непустые критерии — через И, значения внутри списка — через ИЛИ; `mine` — `me.ID` в `AssigneeIDs`.

Контрактный тест (PT-7): `internal/config/contract_v2_test.go` — `TestContractConfigV2*`: полный пример v2 выше (поля подключения — из примера v1) → ожидаемая структура; файл v1 → те же значения, что в v1, и умолчания v2 (`worktree_dir`, `commit_template`, `remote`, `pr.tool=auto`, `refresh` 2m/notify, `cache` вкл); каждая ошибка из списка v2; `ColorsDisabled` с `NO_COLOR`; `EditorCmd`/`CacheDir` по переменным окружения.

### C1 v3 (2026-10-20, решение пользователя о табах-состояниях; задача PT-16) — дополнение к v2

```yaml
tabs:
  states: []              # имена состояний (без учёта регистра) в порядке табов; пусто — все состояния проекта, кроме групп из hide_groups, по group (backlog, unstarted, started, completed, cancelled), затем по sequence
  hide_groups: [cancelled] # по умолчанию [cancelled]; применяется только при пустом states; значения — группы состояний
  all: true               # по умолчанию true: первый таб «Все»
```
```go
type Tabs struct {
    States     []string `yaml:"states"`
    HideGroups []string `yaml:"hide_groups"` // nil в файле → после Load ["cancelled"]; [] явно — ничего не скрывать
    All        *bool    `yaml:"all"`         // nil → true
}
// Config v2 + поле Tabs Tabs `yaml:"tabs"`; метод:
func (c *Config) TabsAll() bool
```
Ошибки: `hide_groups` — значение не из пяти групп; `states` — пустое имя или повтор (без учёта регистра). Имена состояний, которых нет в проекте, — не ошибка конфига (tui их пропускает). Сохранённые фильтры (`filters`) — без изменений; в TUI переключаются не цифрами (цифры — табы), а меню `F` и палитрой (C4).
Контрактный тест (PT-16): `TestContractConfigV3*` в `internal/config/contract_v3_test.go`: умолчания (`HideGroups==["cancelled"]`, `TabsAll()`), явный `hide_groups: []`, `states` в порядке файла, ошибки; v1/v2-тесты проходят (в ожидаемые структуры v1/v2 допустимо добавить умолчания v3 явными константами — то же правило, что в v2).

### C1 v4 (2026-10-20, решение пользователя: только GitLab через glab) — замена секции `pr`

Секция `pr` (v2) **удалена** (ключ `pr` — ошибка неизвестного ключа), вместо неё:
```yaml
gitlab:
  host: ""            # пусто — host из URL remote (cfg.Branch.Remote); иначе передаётся glab через GITLAB_HOST
  repo: ""            # group/project; пусто — из URL remote
  target_branch: ""   # цель MR; пусто — ветка по умолчанию проекта
  draft: false
  glab: glab          # путь/имя бинаря glab; по умолчанию "glab"
  mr_cache: 10m       # сколько хранить найденные MR задачи (Go to MR); 0 — не кэшировать; по умолчанию 10m
```
```go
type GitLab struct {
    Host         string         `yaml:"host"`
    Repo         string         `yaml:"repo"`
    TargetBranch string         `yaml:"target_branch"`
    Draft        bool           `yaml:"draft"`
    Glab         string         `yaml:"glab"`      // "" → "glab" после Load
    MRCache      *time.Duration `yaml:"mr_cache"`  // nil → 10m; < 0 — ошибка
}
// Config v3 − поле PR + поле GitLab GitLab `yaml:"gitlab"`; тип PR и константы pr.tool удаляются.
```
Ошибки: `gitlab.repo` не вида `a/b[/c…]` (непустые сегменты) — ошибка; `gitlab.host` с схемой (`https://…`) — ошибка (только имя хоста[:порт]); `mr_cache < 0` — ошибка. Контрактный тест (PT-22): `TestContractConfigV4*` (`contract_v4_test.go`): умолчания, пример с заполненной секцией, ошибки, `pr:` → ошибка неизвестного ключа; v2/v3-тесты на `pr` — удалить/заменить (допустимое изменение старых тестов: только то, что касается `pr`).

## C2. gitbranch v5 — пакет internal/gitbranch (v5 — дельта в docs/contracts/tui-features.md)

Владелец кода: PT-3. Потребитель: PT-5.

```go
// Данные задачи для шаблона.
type Item struct {
    Identifier string // identifier проекта, напр. "PROJ"
    Sequence   int    // sequence_id задачи, напр. 12
    Title      string
}
func Slug(title string, max int) string              // max<=0 → 40
func Render(template string, it Item) (string, error) // {identifier} → как есть; {sequence}; {slug} → Slug(Title, 40)
func ValidName(name string) error                     // через `git check-ref-format --branch`

type Options struct {
    Base     string // пусто — текущий HEAD
    Checkout bool
}
type Result struct {
    Name     string
    Created  bool // false — ветка уже была
    Switched bool // выполнен checkout
    Dirty    bool // рабочее дерево было грязным (не мешает)
}
var (
    ErrNotRepo      = errors.New("not a git repository")
    ErrBranchExists = errors.New("branch already exists")
)
type Repo struct{ Dir string }
func Open(dir string) (*Repo, error)                    // не репозиторий → ErrNotRepo (errors.Is)
func (r *Repo) Exists(name string) (bool, error)
func (r *Repo) Dirty() (bool, error)
func (r *Repo) Create(name string, o Options) (Result, error) // ветка есть → Result{Name, Created:false}, ErrBranchExists
func (r *Repo) Switch(name string) (Result, error)             // переключиться на существующую

// Для TUI (фейк в тестах PT-5):
type Brancher interface {
    Create(name string, o Options) (Result, error)
    Switch(name string) (Result, error)
}
```

Правила slug: нижний регистр; кириллица — транслитерация (ГОСТ 7.79-2000 схема Б без диакритики, таблица в коде: щ→shch, ж→zh, х→kh, ц→ts, ч→ch, ш→sh, ю→yu, я→ya, ё→yo, й→y, ъ/ь→пусто, ы→y, э→e); всё, кроме `[a-z0-9]`, → `-`; повторные `-` схлопываются; `-` по краям срезаются; обрезка до `max` по границе символа, затем срез хвостового `-`; пустой результат → `task`.
`Render`: `{identifier}` подставляется как есть (`PROJ-12-fix-login`); результат проходит `ValidName`.
Git — только CLI `git` через `os/exec` с `-C <Dir>`, без go-git; stderr git — в тексте ошибки.

Контрактный тест (PT-3): `internal/gitbranch/contract_test.go` — `TestContractGitbranch*`: Slug на таблице примеров (латиница, кириллица, пунктуация, длина), Render по умолчательному шаблону, Create/Exists/Switch/Dirty/ErrNotRepo на временном репозитории (`t.TempDir()`, `git init`, локальные `user.name/email`).

### C2 v2 (2026-10-20, план §4.1, задача PT-8) — дополнения к v1

v3 (2026-10-20, по отчёту PT-8): файл сообщения коммита — в общем git-каталоге (`--git-common-dir`), а не per-worktree `--git-path`, иначе хук в worktree, созданном «стартом работы», его не видит; `Statuses` с пустым `names` — все локальные ветки; в `ParseBranch` шаблон без любого из `{identifier}`/`{sequence}` → сразу запасной разбор.

v1 сохраняется целиком (`TestContractGitbranch*` v1 проходят без правок). Потребители v2: PT-9 (app), PT-5x (tui), PT-10 (CLI). Структура — C0: файлы по ответственности (`slug.go`, `render.go`, `parse.go`, `repo.go`, `worktree.go`, `status.go`, `hook.go`, `remote.go`, `exec.go`), фейк — `internal/gitbranch/gitbranchtest`.

```go
// Обратный разбор: ID задачи из имени ветки.
// 1) регулярка из template: {identifier} → ([A-Za-z][A-Za-z0-9]*), {sequence} → ([0-9]+), {slug} → ([a-z0-9-]*), прочее — буквально; якоря ^…$;
// 2) не совпало или в шаблоне нет {identifier}/{sequence} — первое вхождение (?:^|[/_-])([A-Z][A-Z0-9]*)-([0-9]+)(?:$|[/_-]).
// Identifier возвращается в верхнем регистре; Title пуст.
func ParseBranch(template, name string) (Item, bool)

// Сообщение коммита: {identifier}, {sequence}, {title} (как есть, одна строка), {slug}; неизвестный плейсхолдер — ошибка.
func CommitMessage(template string, it Item) (string, error)

// Путь worktree: {repo} → базовое имя корня репозитория, {branch} → имя ветки с "/" → "-";
// относительный путь — от корня репозитория; результат — абсолютный, очищенный.
func WorktreePath(template, toplevel, branch string) string

type BranchStatus struct {
    Name          string
    Local         bool   // refs/heads/<name>
    Remote        bool   // refs/remotes/<remote>/<name>
    Current       bool   // ветка HEAD этого рабочего дерева
    Worktree      string // путь worktree, где ветка извлечена (кроме текущего); пусто — нигде
    Ahead, Behind int    // коммитов относительно base; base не найден или ветки нет локально — 0, 0
}

var (
    ErrDetached      = errors.New("HEAD is detached")
    ErrCurrentBranch = errors.New("cannot delete the current branch")
    ErrNotMerged     = errors.New("branch is not fully merged")
    ErrHookExists    = errors.New("foreign prepare-commit-msg hook exists")
    ErrWorktreeBusy  = errors.New("branch is checked out in another worktree")
)

func (r *Repo) Toplevel() (string, error)                         // git rev-parse --show-toplevel
func (r *Repo) Current() (string, error)                          // detached → ErrDetached
func (r *Repo) Branches() ([]string, error)                       // локальные, по имени
// Статусы пачкой: один `git for-each-ref` (refs/heads, refs/remotes/<remote>) + `git worktree list --porcelain`
// + `git rev-list --left-right --count <base>...<name>` только для существующих локальных.
// base пусто → refs/remotes/<remote>/HEAD, иначе main, иначе master; не найден — Ahead/Behind = 0.
func (r *Repo) Statuses(names []string, remote, base string) (map[string]BranchStatus, error) // names пусто — все локальные (v3)
// Ветка в отдельном worktree: ветки нет — создаётся от o.Base (пусто — HEAD) в path; есть — `git worktree add <path> <name>`;
// уже извлечена в другом worktree → Result{Name, Path: тот путь}, ErrWorktreeBusy. Result.Switched = false всегда.
func (r *Repo) AddWorktree(name, path string, o Options) (Result, error)
// Удаление: текущая → ErrCurrentBranch; `git branch -d`, при отказе «not fully merged» → ErrNotMerged; force → `-D`.
// Ветка извлечена в другом worktree → ErrWorktreeBusy (worktree не трогается).
func (r *Repo) Delete(name string, force bool) error
// Сообщение коммита для ветки: файл `<git-common-dir>/platui/commit-msg/<name>` (`git rev-parse --path-format=absolute --git-common-dir`; общий для всех worktree — v3), подкаталоги для "/", запись через temp+rename.
func (r *Repo) WriteCommitMsg(branch, msg string) error
// v4 (2026-10-20, подтверждение деструктивных): чтение без записи; нет файла — ok=false, err=nil. В интерфейс Git и gitbranchtest.Fake (по CommitMsgs).
func (r *Repo) ReadCommitMsg(branch string) (msg string, ok bool, err error)
// Хук: путь — `git rev-parse --git-path hooks/prepare-commit-msg` (учитывает core.hooksPath), права 0755.
// Нет файла → пишется; есть с маркером `# platui prepare-commit-msg v2` (или v1 — старый свой) → перезаписывается (идемпотентно, installed=false если байты те же);
// есть чужой → ErrHookExists, файл не меняется.
// Поведение хука (POSIX sh, без вызова platui): $2 пусто или "template"; ветка HEAD; файл сообщения для неё есть;
// файл ищется в `$(git rev-parse --git-common-dir)/platui/commit-msg/<ветка>` (v3: работает и из связанного worktree); первая строка $1 ещё не начинается с текста файла → текст файла + "\n" вставляются в начало $1. Иначе — ничего.
func (r *Repo) InstallHook() (path string, installed bool, err error)
func (r *Repo) Push(remote, name string) error                     // git push -u <remote> <name>; stderr — в ошибке
func (r *Repo) RemoteURL(remote string) (string, error)            // git remote get-url <remote>

// Result v1 + поле Path string — путь worktree (пусто для обычных веток).

// Для app/tui/cli (фейк — gitbranchtest.Fake):
type Git interface {
    Brancher
    Exists(name string) (bool, error)
    Dirty() (bool, error)
    Toplevel() (string, error)
    Current() (string, error)
    Branches() ([]string, error)
    Statuses(names []string, remote, base string) (map[string]BranchStatus, error)
    AddWorktree(name, path string, o Options) (Result, error)
    Delete(name string, force bool) error
    WriteCommitMsg(branch, msg string) error
    InstallHook() (string, bool, error)
    Push(remote, name string) error
    RemoteURL(remote string) (string, error)
}
var _ Git = (*Repo)(nil)
```

Фейк `internal/gitbranch/gitbranchtest`: `type Fake struct` — состояние в полях (`Mu sync.Mutex`, `BranchesV map[string]BranchStatus`, `CurrentV`, `TopV`, `RemoteURLV`, `DirtyV`, `Err map[string]error` по имени метода, `Calls []string`, `Pushed []string`, `CommitMsgs map[string]string`, `HookInstalled bool`); `NewFake() *Fake`; `var _ gitbranch.Git = (*Fake)(nil)`; Create/Switch/AddWorktree/Delete меняют `BranchesV`/`CurrentV` по правилам v1/v2 (существует → `ErrBranchExists` и т.д.).

Контрактный тест (PT-8): `internal/gitbranch/contract_v2_test.go` — `TestContractGitbranchV2*` на временных репозиториях (`t.TempDir()`, `git init`, локальные `user.name/email`; remote — `git init --bare` + `git remote add origin`): ParseBranch по таблице (шаблон по умолчанию, `feature/PROJ-12-x`, без ID → false, нижний регистр identifier в шаблоне → верхний); CommitMessage; WorktreePath (относительный, `/` в имени); Current и ErrDetached; Statuses (локальная без remote, после Push — Remote=true, ahead 2 / behind 1 от base, Worktree); AddWorktree (новая, существующая, ErrWorktreeBusy); Delete (ErrCurrentBranch, ErrNotMerged, force); InstallHook (новый, повтор → installed=false, чужой → ErrHookExists, `core.hooksPath`); хук на деле: `git commit` без `-m` с `GIT_EDITOR=true` даёт первую строку из WriteCommitMsg, с `-m` — не трогает; Push в bare; RemoteURL.

## C3. plane v4 — в docs/contracts/tui-plane.md

Вынесено из-за лимита размера; читать: `.dpcli/dpcli docs section docs/contracts/tui-plane.md C3`. Пакет `internal/plane`: типы, `API`, `*Client`, wire API v1, фейк `planetest`.

## C4. tui v1 — пакет internal/tui

Владелец: PT-5. Зависимости — только через `plane.API` и `gitbranch.Brancher` + функции `gitbranch.Render`/`ValidName`; открытие браузера — через `func(url string) error` в опциях (фейк в тестах).
Библиотеки: `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss` — последняя стабильная мажорная версия, у которой `go` в go.mod ≤ 1.24; если такая требует решений вне плана — шлюз.

```go
type Deps struct {
    API      plane.API
    Git      gitbranch.Brancher   // nil — каталог не git-репозиторий: `b` сообщает об этом в статусной строке
    Config   *config.Config
    OpenURL  func(string) error
}
func New(d Deps) tea.Model
```

## C5–C8. Сервисный слой — в docs/contracts/tui-services.md

Вынесено из-за лимита размера; читать: `.dpcli/dpcli docs section docs/contracts/tui-services.md <C5|C6|C7|C8>`.
- C5 app v1 — `internal/app` (PT-9): сценарии для TUI и CLI.
- C6 text v1 — `internal/text` (PT-11): быстрое создание, Markdown↔HTML, `$EDITOR`, OSC 52.
- C7 forge v1 — `internal/forge` (PT-12): PR/MR через `gh`/`glab`.
- C8 cache v1 — `internal/cache` (PT-13): кэш на диске.
