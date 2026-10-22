---
type: "contract"
status: "closed"
module: "tui"
updated: "2026-10-20"
summary: "Контракт C3 пакета internal/plane — типы, интерфейс API, клиент REST API Plane v1 (wire), фейк planetest"
related: ["docs/contracts/tui.md", "docs/research/plane-api.md"]
contracts: [{"id": "C3", "version": 7}]
---

# Контракт клиента Plane модуля tui

Продолжение `docs/contracts/tui.md` (вынесено из-за лимита размера). Владелец — координатор. Структура — C0.

## C3. plane v7 — пакет internal/plane (v6, v7 — дельты в docs/contracts/tui-features.md)

Владелец кода: PT-4. Потребитель: PT-5. Wire-формат — `docs/research/plane-api.md` (PT-1, §2 и §5). v2 (2026-10-20): сверено с PT-1 — API v1, `Member` расширен, правила разбора и 429 уточнены (раздел «Wire» ниже). v3 (2026-10-20, ревью PT-4): путь `Me` — `/api/v1/users/me/` без workspace (PT-1 §2); Go-интерфейс не менялся. v4 (2026-10-20, план §4.1 — быстрое создание, массовые действия, `$EDITOR`, комментарии): `NewWorkItem`, `WorkItemPatch` расширены, `CreateComment` — блок «v4» ниже.

```go
type Project struct {
    ID, Identifier, Name string
}
type State struct {
    ID, Name, Group, Color string // Group: backlog|unstarted|started|completed|cancelled
}
type Member struct {
    ID, DisplayName, Email, FirstName, LastName string
}
func (m Member) Name() string // DisplayName, иначе "First Last", иначе Email
type Label struct {
    ID, Name, Color string
}
type Comment struct {
    ID        string
    Author    string    // ID участника: `actor`, иначе `created_by`
    HTML      string    // comment_html
    CreatedAt time.Time
}
type WorkItem struct {
    ID          string
    ProjectID   string
    Sequence    int      // sequence_id
    Name        string
    DescHTML    string   // description_html
    StateID     string
    Priority    string   // urgent|high|medium|low|none
    AssigneeIDs []string
    LabelIDs    []string
    CreatedAt, UpdatedAt time.Time
}
type WorkItemPatch struct { // nil — поле не отправляется
    StateID     *string
    Priority    *string
    AssigneeIDs *[]string
}
type NewWorkItem struct {
    Name, DescHTML string
}

type API interface {
    Me(ctx context.Context) (Member, error)
    Projects(ctx context.Context) ([]Project, error)                  // все страницы
    States(ctx context.Context, projectID string) ([]State, error)
    Members(ctx context.Context, projectID string) ([]Member, error)    // project-members
    Labels(ctx context.Context, projectID string) ([]Label, error)
    WorkItems(ctx context.Context, projectID string) ([]WorkItem, error) // все страницы
    WorkItemsUpdated(ctx context.Context, projectID string, since time.Time) ([]WorkItem, int, error) // order_by=-updated_at до первой задачи старше since; int — total_count проекта
    WorkItem(ctx context.Context, projectID, id string) (WorkItem, error)
    CreateWorkItem(ctx context.Context, projectID string, n NewWorkItem) (WorkItem, error)
    UpdateWorkItem(ctx context.Context, projectID, id string, p WorkItemPatch) (WorkItem, error)
    Comments(ctx context.Context, projectID, id string) ([]Comment, error)
    WebURL(p Project, it WorkItem) string                             // ссылка на задачу в веб-интерфейсе
}

type Options struct {
    BaseURL, APIKey, Workspace, WebURL string
    HTTPClient *http.Client  // nil → свой, Timeout 30s
    MaxRetries int           // 429/503: повтор; по умолчанию 3
    Sleep      func(time.Duration) // для тестов; nil → time.Sleep
}
func New(o Options) *Client // *Client реализует API

type APIError struct {
    Status  int
    Message string // из тела ответа (поле error/detail), иначе статус
}
func (e *APIError) Error() string
var ErrUnauthorized, ErrNotFound, ErrRateLimited error // errors.Is для 401/403, 404, 429 после исчерпания повторов
```

Инварианты: заголовок `X-API-Key` в каждом запросе, ключ не попадает в ошибки и логи; пагинация — до конца, все страницы; `ctx` отменяет запрос и ожидание.

Wire (API v1, по PT-1):
- URL: `Me` — `<base_url>/api/v1/users/me/` (без workspace, v3); остальные — `<base_url>/api/v1/workspaces/<workspace>/…`, все пути с завершающим `/`. Эндпоинты под workspace: `projects/`, `projects/<pid>/states/`, `projects/<pid>/project-members/`, `projects/<pid>/labels/`, `projects/<pid>/work-items/` (GET список, POST), `projects/<pid>/work-items/<id>/` (GET, PATCH), `projects/<pid>/work-items/<id>/comments/`. Путь `/issues/` не использовать.
- Пагинация: `per_page=100`, `cursor=<next_cursor>`, пока `next_page_results == true`. Ответ-массив (без конверта) принимается как одна страница; вложенный массив (`[[…]]`) — расплющивается.
- Разбор задачи терпимый: `state` — строка-UUID или объект `{id,…}` (берётся `id`); `state_id` — если есть; `assignees`/`labels` — массив строк или объектов `{id}`; также `assignee_ids`/`label_ids`. `DescHTML` — `description_html`, иначе `description`, если это строка. `ProjectID` — из поля `project`, иначе из аргумента запроса. Неизвестные поля игнорируются.
- Тело POST: `{"name", "description_html"}`; PATCH: только заданные поля под именами `state`, `priority`, `assignees` (массив UUID).
- `State.Group` — строка как пришла (ожидаемые `backlog|unstarted|started|completed|cancelled`, неизвестное значение не ошибка). `Priority` — `urgent|high|medium|low|none`; пусто/null → `none`.
- Ошибки: `APIError.Message` — первое непустое из `detail`, `error`, `message` тела JSON, иначе текст статуса; 401/403 → `ErrUnauthorized`, 404 → `ErrNotFound` (через `errors.Is`, `*APIError` достаётся `errors.As`).
- 429 и 503: до `MaxRetries` повторов; ожидание — `Retry-After` (секунды, не больше 60 s — v3), иначе до `X-RateLimit-Reset` (epoch seconds, не больше 60 s), иначе 1s, 2s, 4s; исчерпано → `ErrRateLimited` (для 429) / `APIError` (для 503). Лимит облака — 60 запросов/мин на ключ.
- `WebURL`: база — `Options.WebURL`, иначе `https://app.plane.so` при `base_url` = `https://api.plane.so`, иначе `base_url`; путь — `<база>/<workspace>/browse/<Identifier>-<Sequence>/`. Формат в документации не подтверждён (PT-1 §4) — гипотеза, README говорит, что при неверной ссылке задать `web_url`.
`internal/plane/html.go`: `func HTMLToText(s string) string` — абзацы/`<br>`/`<li>` → переводы строк, сущности раскодированы, теги убраны.

v4 — дополнения (обратно совместимы на уровне исходников: новые поля — нулевые значения не отправляются):
```go
type NewWorkItem struct {
    Name, DescHTML string
    StateID, Priority     string   // пусто — не отправляется
    AssigneeIDs, LabelIDs []string // nil/пусто — не отправляется
}
type WorkItemPatch struct { // v3 + поля; nil — не отправляется
    StateID, Priority, Name, DescHTML *string
    AssigneeIDs, LabelIDs             *[]string
}
// API v3 + метод:
CreateComment(ctx context.Context, projectID, id, html string) (Comment, error)
```
Wire v4: POST `work-items/` — `name`, `description_html`, `state`, `priority`, `assignees`, `labels` (только заданные); PATCH — `state`, `priority`, `name`, `description_html`, `assignees`, `labels` (только не-nil); POST `projects/<pid>/work-items/<id>/comments/` с телом `{"comment_html": …}`, ответ 201 — комментарий, разбор как в списке (страница add-issue-comment v1, PT-1 её не сверял — гипотеза; при 4xx — `APIError`). Повтор 429/503 для POST — как в v3.

v5 (2026-10-20, решения пользователя о табах-состояниях и поиске; выжимка PT-14 — `docs/research/plane-api.md` §8; задача PT-17):
```go
type State struct { // v4 + поля
    ID, Name, Group, Color string
    Sequence float64 // `sequence`: число или строка с числом; нет/не число → 0
    Default  bool    // `default`, иначе `is_default`
}
type SearchHit struct {
    ID, Name          string
    Sequence          int    // sequence_id
    ProjectID         string // project_id
    ProjectIdentifier string // project__identifier
}
// API v4 + методы:
WorkItemByKey(ctx context.Context, identifier string, seq int) (WorkItem, error)
Search(ctx context.Context, query, projectID string, limit int) ([]SearchHit, error)
```
Wire v5:
- `WorkItemByKey`: `GET <base_url>/api/v1/workspaces/<ws>/work-items/<IDENTIFIER>-<seq>/` (identifier — верхний регистр); разбор — как у `WorkItem`; `ProjectID` — из поля `project`, если есть, иначе пусто (вызывающий сопоставляет проект по identifier); 404 → `ErrNotFound`.
- `Search`: `GET <base_url>/api/v1/workspaces/<ws>/work-items/search/?search=<q>&limit=<n>[&project_id=<pid>]`; `limit<=0` → 50; ответ — объект с массивом `issues` (поля выше); также принимается голый массив; без пагинации. Ищет по name, description, identifier (по документации), по всем проектам при пустом `projectID`.
- Порядок состояний для UI (гипотеза PT-14 §8.1, документация не задаёт): группы `backlog, unstarted, started, completed, cancelled`, внутри — `Sequence` по возрастанию; неизвестная группа (напр. `triage`) — перед `backlog`. Упорядочивает потребитель (app), клиент отдаёт как пришло.
- Fake: `WorkItemByKey` ищет по `ProjectsV[].Identifier` и `Sequence`; `Search` — подстрока без учёта регистра в `Name` по всем `ItemsV` (или одного проекта), `limit` соблюдается.
Контрактный тест (PT-17): `TestContractPlaneV5*`: `sequence` числом и строкой, `default`/`is_default`, путь и регистр `WorkItemByKey`, 404, `Search` — путь, параметры (с `project_id` и без, `limit` по умолчанию 50), конверт `issues` и голый массив, Fake для обоих методов.

Фейк для потребителей (v3, C0): пакет `internal/plane/planetest`:
```go
type Fake struct {
    Mu        sync.Mutex
    MeV       plane.Member
    ProjectsV []plane.Project
    StatesV, MembersV, LabelsV map[string][]...   // по projectID: []plane.State, []plane.Member, []plane.Label
    ItemsV    map[string][]plane.WorkItem         // по projectID; Create/Update меняют его, Sequence — следующий номер
    CommentsV map[string][]plane.Comment          // по work item ID
    Err       map[string]error                    // имя метода ("WorkItems", …) → ошибка, которую вернуть
    Calls     []string                            // журнал вызовов: "UpdateWorkItem:<pid>:<id>"
    Web       string                              // база для WebURL; пусто — "https://app.plane.so/ws"
}
func NewFake() *Fake // пустые карты
var _ plane.API = (*Fake)(nil)
```
Все методы потокобезопасны (Mu), возвращают копии срезов; `CreateComment` добавляет в `CommentsV`; методы, добавленные в API позже, Fake реализует в той же задаче, что меняет API.

Контрактный тест (PT-4): `internal/plane/contract_test.go` — `TestContractPlane*` на `httptest` (все пункты ниже — в тестах с этим префиксом): путь `Me` = `/api/v1/users/me/`; заголовок ключа, пагинация из ≥ 2 страниц, 429 → повтор, 404 → `ErrNotFound`, поля `WorkItem` из образца JSON в формате выжимки PT-1.
