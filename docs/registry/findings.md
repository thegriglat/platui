---
type: "registry"
status: "active"
module: ""
updated: "2026-10-20"
summary: "Реестр выводов: что мы знаем, по модулям (пишется вручную)."
related: []
---

# Реестр выводов

По модулю — раздел `## <модуль>`, вывод — пункт `- …` (одна строка: вывод с числами и ссылка `путь#раздел`). Поиск — `docs findings`.

## english

- Код, тексты UI/CLI, config.example.yml и README на английском; кириллица осталась только в данных транслитерации (gitbranch/slug.go) и тестах поиска/slug; main в корне, `platui version` из build info (E1); go test -race ./... — 457 тестов зелёные (2026-10-20) — `docs/contracts/english.md#E1 v1`
