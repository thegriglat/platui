---
type: "registry"
status: "active"
module: ""
updated: "2026-10-20"
summary: "Исследования: вывод, данные, где применено."
related: []
generated: true
---

# Реестр исследований

Файл собран `docs index` — руками не править.

- [docs/research/api-failures.md](../research/api-failures.md) — 9 доработок - офлайн-старт с кэша, таймауты в конфиге, коды выхода CLI, потеря текста из редактора, ложное время кэша, валидация ответов, Retry-After-дата, ключ в теле ошибки, ошибка после восстановл… (data: Тесты TestFailure* в internal/{plane,app,tui,cli}/failure_test.go - httptest (з…; applied_in: [])
- [docs/research/libraries.md](../research/libraries.md) — Заменить - renameio (атомарная запись), cli/browser (открытие URL), cobra (разбор подкоманд, условно); из TUI брать bubbletea v1, textinput, viewport, lipgloss, glamour, но не bubbles/table; остально… (data: Версии, даты, звёзды, коммиты - GitHub API (gh api) и pkg.go.dev на 2026-10-20;…; applied_in: [])
- [docs/research/plane-api.md](../research/plane-api.md) — Для MVP брать /api/v1 (покрывает всё, включая projects); лимит 60 запросов/мин на ключ; пагинация cursor; ряд полей и URL веб-интерфейса в документации не описан. (data: Только публичная документация developers.plane.so, прочитана 2026-10-20; живой …; applied_in: ['internal/plane'])
