---
type: "registry"
status: "active"
module: ""
updated: "2026-10-20"
summary: "Точка входа: модули docs/ по подпроектам, прочие документы по типам."
related: []
generated: true
---

# Индекс документации

Точка входа; руками не править (`.dpcli/dpcli docs index`). Читать по кусочку, не целиком:
- найти — `.dpcli/dpcli docs find <слова>` (путь — summary), выводы — `.dpcli/dpcli docs findings <слова>` ([реестр выводов](registry/findings.md)), журналы модулей — `.dpcli/dpcli search`;
- читать — `.dpcli/dpcli docs section <путь>` (оглавление), затем `.dpcli/dpcli docs section <путь> <номер|заголовок>`.

Модули — по подпроектам (area), внутри — по дате плана: строка на модуль, под ней — его прочие документы.

## Модули без area

- **tui** — TUI-клиент таск-менеджера Plane на Go (Bubble Tea): проекты, задачи, правка, создание git-ветки из задачи; конфиг в YAML [closed 2026-10-20] — [план](plan/tui.md) · [контракты](contracts/tui.md) · [решения](plan/tui/decisions.md) · [сводка](plan/tui/summary.md) · полный журнал: `archive/tui`
  - [docs/contracts/tui-features.md](contracts/tui-features.md) — Вторая волна фич tui: diff stat, свои команды, закладки, дубли, ссылки, мышь, панели, подзадачи; C11 runner [closed]
  - [docs/contracts/tui-plane.md](contracts/tui-plane.md) — Контракт C3 пакета internal/plane — типы, интерфейс API, клиент REST API Plane v1 (wire), фейк planetest [closed]
  - [docs/contracts/tui-services.md](contracts/tui-services.md) — Стыки сервисного слоя tui — пакеты text, forge, cache, search и app (сценарии поверх plane и gitbranch для TUI и CLI) [closed]
  - [docs/contracts/tui-ui.md](contracts/tui-ui.md) — Контракт TUI (C4 v3): Deps, раскладка (табы-состояния, список+превью), клавиши без vim, поиск, архитектура internal/tui, разбиение PT-5 [closed]
  - [docs/research/api-failures.md](research/api-failures.md) — Аудит недоступности API Plane (PT-45): таблица «как есть / как надо» с тестами TestFailure*, пробелы GAP и список доработок
  - [docs/research/libraries.md](research/libraries.md) — Реестр библиотек вместо самописного кода tui: версии под go 1.24, критерии, замеры на примерах контрактов, рекомендации
  - [docs/research/plane-api.md](research/plane-api.md) — Публичный API Plane (v1, частично v2) под скоуп MVP tui - эндпоинты, пагинация, лимиты, ошибки; расхождения с C3

## Прочие документы

### Реестры (registry)

- [docs/registry/findings.md](registry/findings.md) — Реестр выводов: что мы знаем, по модулям (пишется вручную).
