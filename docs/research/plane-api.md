---
type: research
status: active
module: tui
updated: 2026-10-20
summary: Публичный API Plane (v1, частично v2) под скоуп MVP tui - эндпоинты, пагинация, лимиты, ошибки; расхождения с C3
related: [docs/contracts/tui.md]
conclusion: Для MVP брать /api/v1 (покрывает всё, включая projects); лимит 60 запросов/мин на ключ; пагинация cursor; ряд полей и URL веб-интерфейса в документации не описан.
data: Только публичная документация developers.plane.so, прочитана 2026-10-20; живой инстанс не опрашивался.
applied_in: [internal/plane]
---

# Выжимка публичного API Plane под скоуп MVP

Источник - только публичная документация, каждое утверждение со ссылкой. Живой API не проверялся: всё, что страницы не описывают, помечено «не документировано» и должно быть подтверждено на инстансе (httptest-образцы PT-4 строить по этой выжимке, а не по догадкам).

## 1. Общее

- Базовый URL облака: `https://api.plane.so/`, к нему добавляется путь `/api/v1/...` ([введение](https://developers.plane.so/api-reference/introduction)).
- Self-hosted: базовый URL зависит от домена и настройки, формат страницей не задан ([введение](https://developers.plane.so/api-reference/introduction)); в примере проверки ключа для self-hosted используется `https://plane.yourcompany.com/api/v1/users/me/` ([поиск по MCP-документации](https://developers.plane.so/dev-tools/mcp-server)). Для v2 базовый URL self-hosted - адрес инстанса, например `https://plane.example.com` ([v2 introduction](https://developers.plane.so/api-reference/v2/introduction)). Вывод: `base_url` + `/api/v1/...`, клиент сам добавляет префикс.
- Авторизация: заголовок `X-API-Key: plane_api_<token>` (ключ - Profile Settings > Personal Access Tokens) или `Authorization: Bearer <OAuth token>` ([введение](https://developers.plane.so/api-reference/introduction)). В v2 регистр записан `X-Api-Key`, заголовки регистронезависимы ([миграция v1 -> v2](https://developers.plane.so/api-reference/v2/migrating-from-v1)).
- Лимит запросов: 60 запросов в минуту на API-ключ, счётчик сбрасывается каждую минуту. Заголовки: `X-RateLimit-Remaining` (остаток в окне), `X-RateLimit-Reset` (время сброса, UTC epoch seconds). `Retry-After` во введении v1 не упомянут, 429 - «повторить позже» ([введение](https://developers.plane.so/api-reference/introduction)). В документации v2 для 429 (`rate_limited`) заявлен заголовок `Retry-After` в секундах, лимиты считаются по классам токенов ([v2 errors](https://developers.plane.so/api-reference/v2/errors)); числовые значения и `X-RateLimit-*` там не названы. Вывод для клиента: на 429 ждать `Retry-After`, если он есть; иначе - до `X-RateLimit-Reset`, иначе backoff 1/2/4 c. Лимит self-hosted не документирован.
- Пагинация v1: параметры `per_page` (у списков projects/states/labels/work items/comments по умолчанию 20, максимум 100; во введении указано «по умолчанию 100, максимум 100» - расхождение внутри документации, задавать `per_page=100` явно) и `cursor`; формат курсора `value:offset:is_prev` (пример `20:1:0`). Поля ответа: `next_cursor`, `prev_cursor`, `next_page_results` (bool), `prev_page_results`, `count`, `total_pages`, `total_results`, `total_count`, `grouped_by`, `sub_grouped_by`, `extra_stats`, `results` ([введение](https://developers.plane.so/api-reference/introduction), [список work items](https://developers.plane.so/api-reference/issue/list-issues)). Цикл: пока `next_page_results == true`, подставлять `next_cursor` в `cursor`. Исключения: список участников workspace - голый JSON-массив без пагинации ([workspace members](https://developers.plane.so/api-reference/members/get-workspace-members)); для project-members пагинация не документирована ([project members](https://developers.plane.so/api-reference/members/get-project-members)).
- `fields` (список полей через запятую) и `expand` (связанные объекты inline, напр. `expand=assignees,state`); неизвестные значения - ошибка ([введение](https://developers.plane.so/api-reference/introduction)).
- Формат ошибок v1: страницы эндпоинтов v1 коды ошибок и тело не документируют (проверено на страницах me, projects, states, labels, work items, comments). В v2 ошибки - `application/problem+json` с полями `type`, `code`, `detail` и `errors[]` для 400 ([v2 errors](https://developers.plane.so/api-reference/v2/errors)). Клиент для v1 должен терпимо разбирать тело: искать `error`/`detail`/`message`, иначе текст статуса (согласуется с `APIError` в C3).
- Версии: v1 продолжает работать, срока отключения нет; v2 покрывает work items, comments, states, labels, members, текущего пользователя, одновременно использовать оба можно тем же ключом ([миграция](https://developers.plane.so/api-reference/v2/migrating-from-v1)). Ответы v2 используют конверт `data` + `pagination.style` и поля `*_id`/`*_ids` вместо вложенных/`assignees`.
- Работа по `/issues/` или `/work-items/`: в актуальной документации v1 путь `/work-items/` (страницы называются «work items»); пути `/issues/` в текущей документации нет, но URL страниц остались `.../api-reference/issue/...` ([список work items](https://developers.plane.so/api-reference/issue/list-issues)). Признаков, что `/issues/` объявлен устаревшим, на прочитанных страницах нет - не документировано; использовать только `/work-items/`.

## 2. Эндпоинты v1 (выбор для MVP)

Префикс `/api/v1`, `{ws}` - slug workspace, `{pid}` - UUID проекта. Все пути заканчиваются `/`.

| Метод | Путь | Параметры / ключевые поля | Источник |
|---|---|---|---|
| GET | /api/v1/users/me/ | scope `profile:read`; ответ: `id`, `first_name`, `last_name`, `email`, `avatar`, `avatar_url`, `display_name` | [me](https://developers.plane.so/api-reference/user/get-current-user) |
| GET | /api/v1/workspaces/{ws}/projects/ | `cursor`, `per_page`, `order_by`, `expand`, `fields`; scope `projects:read`; результат: `id`, `name`, `description`, `identifier`, `network` | [projects](https://developers.plane.so/api-reference/project/list-projects) |
| GET | /api/v1/workspaces/{ws}/projects/{pid}/states/ | пагинация как выше; scope `projects.states:read`; результат: `id`, `name`, `color`, `group` (в примере `started`, полный перечень не приведён), `sequence` | [states](https://developers.plane.so/api-reference/state/list-states) |
| GET | /api/v1/workspaces/{ws}/projects/{pid}/project-members/ | параметры не документированы; scope `projects.members:read`; поля: `id`, `first_name`, `last_name`, `display_name`, `email`, `avatar`; в примере массив вложен в массив - форма ответа неясна | [project members](https://developers.plane.so/api-reference/members/get-project-members) |
| GET | /api/v1/workspaces/{ws}/members/ | scope `workspaces.members:read`; голый массив; поля: `id`, `first_name`, `last_name`, `display_name`, `email`, `avatar`, `role` (int, в примере 20 и 15) | [workspace members](https://developers.plane.so/api-reference/members/get-workspace-members) |
| GET | /api/v1/workspaces/{ws}/projects/{pid}/labels/ | `cursor`, `per_page`, `order_by`, `expand`, `fields`; scope `projects.labels:read`; результат: `id`, `name`, `color`, `description` | [labels](https://developers.plane.so/api-reference/label/list-labels) |
| GET | /api/v1/workspaces/{ws}/projects/{pid}/work-items/ | `cursor`, `per_page`, `order_by`, `expand`, `fields`, `external_id`, `external_source`; scope `projects.work_items:read`; в примере: `id`, `name`, `description`, `priority`, `sequence_id`, `state{id,name,group}`, `assignees[]`, `labels[]`, `created_at` (полная схема, `description_html`, `updated_at`, `project` на странице не показаны) | [list work items](https://developers.plane.so/api-reference/issue/list-issues) |
| GET | /api/v1/workspaces/{ws}/projects/{pid}/work-items/{id}/ | `expand`, `fields`; ответ (из примера): `id`, `name`, `description`, `sequence_id`, `priority`, `assignees[]` (UUID), `labels[]` (UUID), `created_at`, `updated_at` | [get work item](https://developers.plane.so/api-reference/issue/get-issue-detail) |
| GET | /api/v1/workspaces/{ws}/work-items/{PROJ}-{N}/ | поиск по идентификатору проекта и номеру (`PROJ-123`), без UUID проекта; те же поля ответа | [by identifier](https://developers.plane.so/api-reference/issue/get-issue-sequence-id) |
| POST | /api/v1/workspaces/{ws}/projects/{pid}/work-items/ | тело JSON: `name` (обязательно), `description_html`, `description_stripped`, `priority` (`urgent`\|`high`\|`medium`\|`low`\|`none`), `state`, `assignees[]`, `labels[]`, `parent` и др.; scope `projects.work_items:write`; ответ 201 | [create](https://developers.plane.so/api-reference/issue/add-issue) |
| PATCH | /api/v1/workspaces/{ws}/projects/{pid}/work-items/{id}/ | тело: любые из `name`, `state`, `priority`, `assignees[]`, `labels[]`, `description_html`, ...; ответ 200 - обновлённая задача | [update](https://developers.plane.so/api-reference/issue/update-issue-detail) |
| GET | /api/v1/workspaces/{ws}/projects/{pid}/work-items/{id}/comments/ | `cursor`, `per_page`, `order_by`, `expand`, `fields`; scope `projects.work_items.comments:read`; схема комментария на странице списка не описана | [comments list](https://developers.plane.so/api-reference/issue-comment/list-issue-comments) |

Заметки по полям:
- Комментарий v1 (объект из обзора): `id`, `created_at`, `updated_at`, `edited_at`, `comment_html`, `comment_stripped`, `access` (`INTERNAL`/`EXTERNAL`), `created_by`, `updated_by`, `actor` (UUID), `issue`, `project`, `workspace`, `is_member`, `deleted_at` ([обзор комментариев](https://developers.plane.so/api-reference/issue-comment/overview)). Реальный состав ответа списка не подтверждён - не документировано.
- Значения приоритета подтверждены для тела POST/PATCH: `urgent`, `high`, `medium`, `low`, `none` ([create](https://developers.plane.so/api-reference/issue/add-issue)).
- Перечень `state.group` в документации v1 не приведён (в примерах - `started`); значения `backlog|unstarted|started|completed|cancelled` из C3 здесь не подтверждены.
- Формат `state`, `assignees` в теле POST/PATCH описан только как «State»/«Assignees»; по примеру - UUID ([update](https://developers.plane.so/api-reference/issue/update-issue-detail)). Какое именно поле (`state` или `state_id`) и вложены ли они в списке/get - объект или UUID - в ответе v1 не определено: в списке показан объект `state{id,name,group}`, в get-примере - только перечисленные выше поля. Клиент должен принимать и объект, и строку.
- Получение отдельной задачи с описанием: `description_html` в описании v1 get отсутствует (пример возвращает `description`); для v2 `description_html` приходит в одиночном ответе, в списке - только через `fields` ([v2 list](https://developers.plane.so/api-reference/v2/work-items/list-work-items)). Для текстового описания в v1 запрашивать `fields=...,description_html` и проверить на инстансе.
- Вложенность `assignees` в задаче можно раскрыть `expand=assignees` ([get work item](https://developers.plane.so/api-reference/issue/get-issue-detail)), но состав объекта неизвестен; для карточки задач имена брать из списка участников.

## 3. Эндпоинты v2 (альтернатива, не для MVP по умолчанию)

| Метод | Путь | Особенности | Источник |
|---|---|---|---|
| GET | /api/v2/users/me/ | `id`, `display_name`, `email`, `principal_kind`, `scopes`; без `first_name`/`avatar` | [v2 me](https://developers.plane.so/api-reference/v2/users/get-current-user) |
| GET | /api/v2/workspaces/{slug}/projects/{project_id}/work-items/ | фильтры `state_id`, `state_group`, `priority`, `assignee_id`, `label_id`, `search`, `sequence_id`; `per_page` 50 (макс 200), `offset` до 10000 или `paginate=cursor`; конверт `data`/`next`/`next_cursor`/`has_more`; поля `state_id`, `assignee_ids`, `label_ids`, `identifier`; `description_html` только через `fields` | [v2 list](https://developers.plane.so/api-reference/v2/work-items/list-work-items) |
| GET | /api/v2/workspaces/{slug}/work-items/{identifier}/ | поиск по `PROJ-142`, регистр не важен; полный объект с `description_html`; 404 на любую недоступность | [v2 by identifier](https://developers.plane.so/api-reference/v2/work-items/get-work-item-by-identifier) |
| GET | /api/v2/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/comments/ | поля `id`, `parent_id`, `comment_html`, `comment_stripped`, `access`, `actor_id`, `edited_at`, `hidden_at`, `created_at`; плоский список, ответы - отдельные строки (треды по `parent_id`); пагинация и порядок не описаны | [v2 comments](https://developers.plane.so/api-reference/v2/work-item-comments/overview) |
| GET | /api/v2/workspaces/{slug}/projects/{project_id}/states/ | пагинированный список состояний проекта | [v2 states](https://developers.plane.so/api-reference/v2/states/list-states) |
| GET | /api/v2/workspaces/{slug}/projects/{project_id}/members/ | `expand=member` добавляет имя и email | [v2 members](https://developers.plane.so/api-reference/v2/members/list-project-members) |

Рекомендация: MVP на v1 - покрывает все нужные операции, включая список проектов (в v2 проектный маршрут есть в таблице переезда, но страницу списка проектов подтвердить не удалось), v1 не отключается ([миграция](https://developers.plane.so/api-reference/v2/migrating-from-v1)). Единственное преимущество v2 - отдельные поля `*_id`, фильтры списка на стороне сервера (`state_group`, `search`) и поиск `PROJ-N` без различия регистра.

## 4. Ссылка на задачу в веб-интерфейсе

Формат URL просмотра задачи (`/browse/PROJ-12/` или `/projects/<id>/issues/<id>`) в публичной документации разработчика не найден (два поиска, [work items](https://developers.plane.so/api-reference/v2/work-items/overview)) - не документировано. Подтверждено лишь, что slug workspace виден в URL приложения вида `https://app.plane.so/my-team/projects/` ([get work item](https://developers.plane.so/api-reference/issue/get-issue-detail)). Вывод: по памяти формат не фиксируем; PT-4/PT-5 берут `WebURL` как `<web_url>/<workspace>/browse/<IDENT>-<seq>/` только как гипотезу, которую должен проверить пользователь на своём инстансе. Для self-hosted `web_url` из C1 обязателен на практике: API-адрес (`base_url`) может не совпадать с адресом интерфейса (для облака `api.plane.so` и `app.plane.so` различаются - [введение](https://developers.plane.so/api-reference/introduction)).

## 5. Расхождения с C3 (docs/contracts/tui.md)

Контракт не правится; для решения координатора.

1. Поля `Member`: в v1 нет единого имени; есть `display_name`, `email`, `first_name`, `last_name`, `avatar`, в project-members - без `role`. `Me` v1 даёт `display_name`, `email` - совпадает.
2. `Comment.Author`: в v1 автор - `actor` (UUID), также `created_by`; в v2 - `actor_id`. `HTML` = `comment_html` - совпадает; `CreatedAt` = `created_at` - совпадает.
3. `WorkItem.StateID`, `AssigneeIDs`, `LabelIDs`: в v1 список возвращает `state` объектом `{id,name,group}`, `assignees`/`labels` - массивы (по примеру UUID); в v2 - `state_id`, `assignee_ids`, `label_ids`. Wire-имена C3 (`state_id` и пр.) в v1 не подтверждены - клиент должен разбирать оба варианта. В PATCH/POST v1 поля называются `state`, `assignees`, `labels`, а не `state_id`/`assignee_ids`.
4. `WorkItem.DescHTML` (`description_html`): в примерах v1 есть только `description`; в теле POST/PATCH - `description_html`. Запрос `fields=description_html` на v1 не подтверждён.
5. `WorkItem.ProjectID`: в v1-примерах поля `project` нет; задавать из контекста запроса.
6. `WorkItem.Sequence` = `sequence_id` - совпадает. `Identifier` проекта берётся из `Project.Identifier` (`identifier` в projects); в v2 задача несёт `identifier` сама.
7. `State.Group`: перечень значений в документации v1 не приведён (в примере `started`); `State` в v1 содержит ещё `sequence`.
8. `Priority`: `urgent|high|medium|low|none` - совпадает с телом create/update.
9. Пагинация: C3 требует «все страницы» - v1 курсорная (`next_cursor`, `next_page_results`), `per_page` ≤ 100; список участников workspace без пагинации (массив), project-members - неясно. Клиент должен определять тип ответа: массив или объект с `results`.
10. 429: в v1 документированы лимит 60/мин и `X-RateLimit-Remaining`/`X-RateLimit-Reset`; `Retry-After` - только в v2-документации. Инвариант C3 «ждать `Retry-After` или backoff 1/2/4 c» нужно дополнить ожиданием до `X-RateLimit-Reset`. 503 в документации не упомянут (в C3 `MaxRetries` для 429/503).
11. `Members(projectID)`: в v1 путь `/project-members/`, не `/members/`; список workspace - `/workspaces/{ws}/members/`. Для назначения «на себя» достаточно `Me().ID`; членство проекта проверять не обязательно.
12. `WorkItem(ctx, projectID, id)`: для поиска по `PROJ-12` есть отдельный эндпоинт без UUID проекта - в C3 нет метода (при необходимости добавить `WorkItemByIdentifier`).
13. `APIError.Message` «из поля error/detail»: формат тела ошибок v1 не документирован; v2 использует `detail` и `code`.
14. `WebURL`: путь по PT-1 не определён документацией (см. §4).

## 6. Допущения и границы

- Страницы v1 в основном показывают только пример ответа, а не схему; типы выводились из примеров.
- Прочитано 2026-10-20 через публичную документацию; изменения API после этой даты не учтены.
- Не проверялись: поведение на self-hosted, реальные лимиты self-hosted, наличие `description_html` в ответе v1.

## 7. Что проверить на живом инстансе

- `sequence` и флаг default в ответе v1 списка состояний; порядок состояний в UI (§8.1).
- Есть ли `project`/`project_id` в ответе v1 по `PROJ-N` (§8.2).
- Семантика `search` по identifier в v1 `/work-items/search/`, значение `limit` по умолчанию (§8.3).
- Допустимость `order_by=-updated_at` и `expand=labels` в v1 (§8.4).
- Источник сведений - [developers.plane.so](https://developers.plane.so/api-reference/introduction); подробности и ссылки - §8 и §9.

## 8. Поиск, порядок состояний, задача по идентификатору

Источник - публичная документация developers.plane.so, прочитана 2026-10-20 (страницы пересмотрены для PT-14); код makeplane/plane не читался. Живой инстанс не опрашивался.

### 8.1. Состояния: `sequence`, `default`, порядок

- v1 `GET /api/v1/workspaces/{ws}/projects/{pid}/states/`: страница показывает только пример элемента: `id`, `name`, `color`, `group` (`started`), `sequence` (число, в примере `2`); флага default в ответе списка нет; схема, типы и порядок сортировки не описаны ([v1 list states](https://developers.plane.so/api-reference/state/list-states)). Параметры: `cursor`, `expand`, `fields`, `per_page` (20, максимум 100); `order_by` не документирован.
- v1 обзор State: `sequence` - «Auto generated sequence of the state for ordering» (на странице тип записан как string, в примере число - расхождение, разбирать терпимо); `default` (bool) - состояние, в котором создаются задачи без явного состояния; `group` - `backlog`, `unstarted`, `started`, `completed`, `cancelled` ([v1 state overview](https://developers.plane.so/api-reference/state/overview)). Имя флага в ответе списка v1 (`default`) на странице списка не показано - гипотеза, проверить на инстансе.
- v2 `GET /api/v2/workspaces/{slug}/projects/{project_id}/states/`: `sequence` - порядок рабочего процесса проекта: «Order by `sequence` when you are rendering the workflow to a user; it is the order the project itself uses» (примеры 15000, 25000, 35000, список приходит по возрастанию); `is_default` (bool); `group` из `backlog|unstarted|started|completed|cancelled|triage` (в v2 добавлен `triage`); фильтры `name`, `group`, `group__in`, `is_default`, `external_id`, `external_source`, `search`; `order_by` = `sequence|created_at|id` (с `-`, неизвестное значение молча игнорируется); `per_page` 50 (макс 200), `offset`, `paginate=cursor`. Рекомендация документации: ориентироваться на `group`, а не на имя состояния, так как состояние можно переименовать ([v2 list states](https://developers.plane.so/api-reference/v2/states/list-states)).
- Как UI Plane упорядочивает состояния (группа, затем `sequence`?) - в документации не найдено. Гипотеза для клиента: порядок групп `backlog, unstarted, started, completed, cancelled`, внутри группы по `sequence` по возрастанию; для задач v2 `order_by=state_group` сортирует «по порядку рабочего процесса» ([v2 list work items](https://developers.plane.so/api-reference/v2/work-items/list-work-items)). Подтвердить на инстансе.

### 8.2. Задача по идентификатору `PROJ-N`

- v1: `GET /api/v1/workspaces/{workspace_slug}/work-items/{project_identifier}-{issue_identifier}/` (пример `.../work-items/PROJ-123/`), scope `projects.work_items:read`; ответ: `id`, `name`, `description`, `sequence_id`, `priority`, `assignees`, `labels`, `created_at`, `updated_at`. Поля `project` в примере нет, страница о нём молчит - не документировано; `project_id` брать нельзя, искать по `Project.Identifier` из списка проектов ([v1 by identifier](https://developers.plane.so/api-reference/issue/get-issue-sequence-id)).
- v2: `GET /api/v2/workspaces/{slug}/work-items/{identifier}/`, регистр идентификатора не важен (часть до дефиса начинается с буквы и приводится к верхнему регистру); ответ полный: `id`, `name`, `description_html`, `identifier`, `sequence_id`, `priority`, `project_id` (только id, объекта проекта и `expand=project` нет), `state_id`, `assignee_ids`, `label_ids`, `created_at`, `updated_at` и др.; `expand`: `assignees, cycle, estimate_point, labels, modules, parent, state, type`; 404 `not_found` на любую недоступность (в т. ч. неверный формат и архивные) ([v2 by identifier](https://developers.plane.so/api-reference/v2/work-items/get-work-item-by-identifier)).

### 8.3. Серверный поиск

- v1, workspace-wide: `GET /api/v1/workspaces/{workspace_slug}/work-items/search/`; параметры: `search` (обязателен; «filter results by name, description, or identifier»), `limit` (необязателен, значение по умолчанию и максимум не указаны), `project_id` (необязателен - сужение до проекта); scope `projects.work_items:read`. Ответ: объект с массивом `issues`; элемент: `id`, `name`, `sequence_id`, `project__identifier`, `project_id`, `workspace__slug` ([v1 search](https://developers.plane.so/api-reference/issue/search-issues)). Состояния, приоритета и описания в ответе нет, пагинации нет (только `limit`); для карточки - дозапрос задачи. Поиск по всем проектам workspace - да (без `project_id`). Что значит поиск по identifier (`PROJ-12`, только `12`) - не документировано.
- v1, список проекта `.../projects/{pid}/work-items/`: параметра `search` и фильтра по `updated_at` нет; документированы только `cursor`, `expand`, `external_id`, `external_source`, `fields`, `order_by`, `per_page` (20, макс 100). Значения `order_by` и `expand` страница не перечисляет ([v1 list](https://developers.plane.so/api-reference/issue/list-issues)). `expand=assignees,state` встречается во введении v1 ([введение](https://developers.plane.so/api-reference/introduction)); `labels` в перечне не подтверждён.
- v2, проект `GET /api/v2/workspaces/{slug}/projects/{project_id}/work-items/` и workspace `GET /api/v2/workspaces/{slug}/work-items/` (все проекты; `project_id`, `project_id__in` сужают): `search` - свободный текст только по `name` (не по описанию и не по номеру); для номера есть фильтр `sequence_id`. Пагинация: offset (по умолчанию, `offset` до 10000, есть `total_count`) или `paginate=cursor` (`next_cursor`, `has_more`, без счётчика); `per_page` 50, максимум 200; `count=false` убирает `COUNT`. Фильтры (AND): `state_id[__in]`, `state_group[__in]`, `priority[__in]`, `assignee_id[__in|__isnull]`, `label_id[__in|__isnull]`, `archived` (`false`, `true`, `only`), `created_at__gte/lte`, `updated_at__gte/lte` и др. `order_by` по умолчанию `-created_at`; допустимы `created_at, updated_at, sequence_id, id, sort_order, priority, state_group`; `priority`, `state_group`, `sort_order` с cursor дают 400 `ordering_not_cursor_eligible`; неизвестное значение молча заменяется порядком по умолчанию. 429 - `Retry-After` ([v2 list](https://developers.plane.so/api-reference/v2/work-items/list-work-items), [v2 workspace list](https://developers.plane.so/api-reference/v2/work-items/list-workspace-work-items), [v2 overview](https://developers.plane.so/api-reference/v2/work-items/overview)).
- Отдельного поискового эндпоинта v2 нет; v1 `/work-items/search/` в таблице v2 не значится.

### 8.4. Опрос изменений и `expand`

- v1 список: фильтра по `updated_at` нет (не документирован). Вариант: `order_by=-updated_at` (значение `updated_at` не подтверждено документацией v1 - гипотеза) и остановка по известной метке.
- v2: `?updated_at__gte=<ISO8601>&order_by=updated_at`, документация прямо описывает это как инкрементальный опрос; архивирование и разархивирование `updated_at` не меняют - отслеживать отдельным запросом `archived=only`. Удалённые задачи таким опросом не видны (в документации не описано).
- `expand`: v2 - `assignees, cycle, estimate_point, labels, modules, parent, state, type` (объект добавляется рядом с `*_id`, неизвестное значение - 400); v1 - формально «comma-separated list of related fields», перечень не приведён; `assignees,state` показаны примером во введении, `labels` - не подтверждён.

### 8.5. Выводы для клиента

1. Поиск по тексту из UI: v1 `/work-items/search/` (один запрос по всему workspace, поля name, description, identifier, но ответ скудный) либо v2 `search` (только name, полные строки, пагинация, фильтры по состоянию). Совмещать v1 и v2 одним ключом допустимо ([миграция](https://developers.plane.so/api-reference/v2/migrating-from-v1)). Выбор - решение координатора.
2. Переход к задаче по `PROJ-N`: v1 или v2 identifier-эндпоинт без UUID проекта; `project_id` из v1-ответа не получить - сопоставлять по `identifier` проекта.
3. Табы состояний: упорядочивать по `group` (фиксированный порядок), затем `sequence`; не полагаться на имена состояний.
4. Расхождения с C3: добавить `WorkItemByIdentifier` и `Search`; `State.Default`, `sequence` в v1 - число/строка неясно.

## 9. Источники

- [Введение в API](https://developers.plane.so/api-reference/introduction)
- [Plane API v2](https://developers.plane.so/api-reference/v2/introduction)
- [Миграция v1 -> v2](https://developers.plane.so/api-reference/v2/migrating-from-v1)
- [Ошибки v2](https://developers.plane.so/api-reference/v2/errors)
- [Обзор комментариев v1](https://developers.plane.so/api-reference/issue-comment/overview)
- [MCP server (пример self-hosted)](https://developers.plane.so/dev-tools/mcp-server)
- [v1 state overview](https://developers.plane.so/api-reference/state/overview), [v1 search](https://developers.plane.so/api-reference/issue/search-issues), [v2 states](https://developers.plane.so/api-reference/v2/states/list-states), [v2 workspace list](https://developers.plane.so/api-reference/v2/work-items/list-workspace-work-items)

