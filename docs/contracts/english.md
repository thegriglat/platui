---
type: "contract"
status: "closed"
module: "english"
updated: "2026-10-20"
summary: "E1 v1 — platui version из build info (ldflags перекрывает); E2 v1 — текст комментария таймера app→tui на английском"
related: ["docs/plan/english.md", "docs/contracts/tui.md", "docs/contracts/certs.md"]
contracts: [{"id": "E1", "version": 1}, {"id": "E2", "version": 1}]
---

# Контракты модуля english

## E1 v1 — версия `platui version`

Владелец: EN-1. Потребители: пользователи CLI, сборка (`-ldflags -X`), EN-5 (не трогает версию).

- Пакет `internal/cli`, новый файл `internal/cli/version.go` (+ `version_test.go`).
- `var Version = "dev"` остаётся в `internal/cli/cli.go` (путь для `-ldflags "-X github.com/thegriglat/platui/internal/cli.Version=…"` не меняется).
- `func resolveVersion(ldflags string, bi *debug.BuildInfo, ok bool) string` — чистая функция, порядок:
  1. `ldflags != "dev" && ldflags != ""` → `ldflags` (перекрывает всё);
  2. `ok && bi.Main.Version != "" && bi.Main.Version != "(devel)"` → `bi.Main.Version` (так работает `go install …@v1.2.3` / `@latest`);
  3. `ok` и в `bi.Settings` есть `vcs.revision` → `"devel " + revision[:min(12,len)]`, плюс суффикс `"-dirty"`, если `vcs.modified == "true"`;
  4. иначе → `"dev"`.
- `func versionString() string { bi, ok := debug.ReadBuildInfo(); return resolveVersion(Version, bi, ok) }`.
- `platui version` печатает в stdout ровно `platui <versionString()>\n`, exit 0; формат строки не меняется.
- Контрактный тест: `TestContractE1Version` в `internal/cli/version_test.go` — таблица по пунктам 1–4 (включая `-dirty` и короткий revision).

## E2 v1 — HTML комментария «затраченное время» (app → tui)

Владелец: EN-4 (`internal/app/timer.go`). Потребитель: EN-2 (`internal/tui/item_test.go`).

- Было: `<p>Затрачено: %s</p>`. Стало: `<p>Time spent: %s</p>`, `%s` — `FormatDuration(d)` без изменений.
- Тест потребителя ищет подстроку `Time spent`.
- Контрактный тест: `TestContractE2TimerComment` в `internal/app` — HTML комментария начинается с `<p>Time spent: `.

## Общее для всех задач перевода

- Другие строки между пакетами идут через идентификаторы (`app.CertHint`, ошибки через `errors.Is`) — тексты потребителей не сверяют; найдётся сверка литерала чужого пакета — `contract:` координатору.
- Строки UI, закреплённые в контрактных тестах (`contract_*_test.go`), переводятся вместе с кодом; исполнитель перечисляет их в отчёте — координатор обновляет цитаты в `docs/contracts/tui*.md`, `certs.md`.
