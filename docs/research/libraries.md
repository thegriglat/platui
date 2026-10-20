---
type: research
status: active
module: tui
updated: 2026-10-20
summary: Реестр библиотек вместо самописного кода tui: версии под go 1.24, критерии, замеры на примерах контрактов, рекомендации
related: [docs/contracts/tui-services.md, docs/contracts/tui-ui.md, docs/plan/tui.md]
conclusion: Заменить - renameio (атомарная запись), cli/browser (открытие URL), cobra (разбор подкоманд, условно); из TUI брать bubbletea v1, textinput, viewport, lipgloss, glamour, но не bubbles/table; остальное (транслитерация, HTML, OSC 52, повторы 429, editor, URL remote, формат времени) оставить своим - библиотеки либо не совпадают с контрактом, либо не проходят критерии выбора.
data: Версии, даты, звёзды, коммиты - GitHub API (gh api) и pkg.go.dev на 2026-10-20; поведение - запуск кандидатов в песочнице вне репозитория (go 1.24.2, GOTOOLCHAIN=local); исходники своего кода - копия internal/ ветки tui/PT-18.
applied_in: []
---

# Реестр библиотек вместо своего кода (модуль tui)

Решение пользователя 2026-10-20: брать зрелую библиотеку вместо своего кода, если она совместима с go 1.24 и проходит критерии C0. Документ - вход для решений координатора и пользователя; код и контракты не менялись.

## 1. Критерии и способ проверки

Критерии (решение пользователя 2026-10-20): много пользователей (звёзды, импортёры на pkg.go.dev); живая поддержка (релиз и коммиты за последние 12 месяцев); больше одного активного мейнтейнера или организация; стабильный API (v1+ или известный проект); без тяжёлых транзитивных зависимостей. Проект одного человека или заброшенный - не брать, рекомендация «оставить своё».

Как собрано (все числа на 2026-10-20, перепроверяемы):

- Версия с go <= 1.24: `go mod download -json <модуль>@<версия>` -> поле `GoMod` -> директива `go`; перебор версий от последней вниз до первой с директивой <= 1.24 (локальный тулчейн go 1.24.2, `GOTOOLCHAIN=local`, чтобы go не скачивал новый). Затем `go get <модуль>@<версия>` в пустом модуле `go 1.24` - проверка, что итоговая директива `go` не выросла (транзитивные требования).
- Звёзды, владелец (Organization/User), лицензия, дата последнего коммита, релизы, авторы коммитов за 365 дней (до 100 коммитов в выборке): `gh api repos/<o>/<r>`, `.../releases/latest`, `.../commits?since=...`.
- Импортёры: `https://pkg.go.dev/<пакет>?tab=importedby`, строка «Known importers» (для TUI - по конкретному пакету, например `bubbles/table`).
- Транзитивные зависимости: число модулей из `go list -deps -f '{{.Module.Path}}'` для программы, импортирующей только нужный пакет (включая сам модуль).
- Поведение: песочница с одним `go.mod` и маленькими программами; свой код скопирован из `internal/text`, `internal/plane/html.go`, `internal/gitbranch/slug.go` без правок.

Ограничения: «число мейнтейнеров» - только по авторам коммитов за 12 месяцев (без ботов), это оценка, а не данные об организации; звёзды и импортёры меняются; примеры в таблицах поведения - выборка, не полный тест. То, что не проверялось, помечено «не проверено».

## 2. Реестр по местам

Колонки по решению координатора. «go» - директива `go` в go.mod выбранной версии; «Транз.» - число модулей с учётом самого модуля.

| Место | Сейчас | Кандидат | Версия, go | Лицензия | Зрелость | Мейнтейнеры (12 мес) | Релиз / коммит | Транз. | Соответствие контракту, отличия | Рекомендация |
|---|---|---|---|---|---|---|---|---|---|---|
| 1. text.HTMLToMarkdown | свой токенайзер и рендер, 547 строк (htmltree 164, html2md* 383) | [html-to-markdown/v2](https://github.com/JohannesKaufmann/html-to-markdown) | v2.5.0, go 1.24.0 (v2.5.1 и v2.5.2 требуют go 1.25.0) | MIT | 3835 звёзд, 113 импортёров | 1 (JohannesKaufmann: 11 из 11 коммитов) | v2.5.2 2026-06-07 / 2026-06-07 | 3 (dom, x/net) | Нужны плагины base, commonmark, strikethrough, table (без них `<s>` и таблицы теряют разметку). Совпало побайтно 4 из 12 примеров (§3.2), остальные отличаются оформлением: `<br>` - два пробела вместо `\`, вложенный список с пустой строкой, таблица `\|---\|`, `<b>` экранируется как `&lt;b&gt;`; скрипт отбрасывается (у нас текст скрипта протекает). Равенство после `MarkdownToHTML` не проверялось | Оставить: проект одного человека, две последние версии требуют go 1.25 (привязка к v2.5.0 без обновлений). Единственный живой кандидат - замена возможна только решением пользователя в обход критерия мейнтейнеров |
| 2. plane.HTMLToText | regexp, 30 строк | [k3a/html2text](https://github.com/k3a/html2text) | v1.5.1, go 1.16 | MIT | 191 звезда, 154 импортёра | 1 (k3a: 21 из 24) | v1.5.1 2026-10-12 / 2026-10-12 | 1 | Подставляет URL вместо текста ссылки; вложенные списки без маркеров, таблица слипается (`AB12`), лишние пустые строки (§3.2). Не совпадает с контрактом «абзацы/`<br>`/`<li>` -> строки, теги убраны» | Оставить (один мейнтейнер, поведение другое) |
| 2. plane.HTMLToText | то же | [jaytaylor/html2text](https://github.com/jaytaylor/html2text) | только псевдоверсия v0.0.0-20260303211410, go.mod без `go` | MIT | 589 звёзд, 605 импортёров | 1 (Necoro, 3 коммита) | релизов нет / 2026-03-03 | сам + x/net | `go get` тянет x/net v0.61.0 с требованием go 1.26.0 - в go 1.24 не собирается без ручной подгонки версии x/net (не проверено) | Не брать (нет релизов, один мейнтейнер, тянет go 1.26) |
| 3. gitbranch.Slug | таблица рун, 50 строк | [gosimple/slug](https://github.com/gosimple/slug) | v1.15.0, go 1.11 | MPL-2.0 | 1335 звёзд, 1920 импортёров | org gosimple, коммитов за 12 мес нет | v1.15.0 2024-12-23 / 2024-12-23 | 2 (unidecode) | `MakeLang(s, "ru")`: щ -> shch, ж -> zh, х -> kh, ц -> ts, но ю -> iu, я -> ia, й -> i, ы -> y, ъ/ь дают дефис (`s-ezd`); обрезка по границе слова, а не по символу; верхний регистр - только с `Lowercase`. Совпало 3 из 10 строк (§3.1) | Оставить (не совпадает с таблицей C2, нет активности 12 мес) |
| 3. gitbranch.Slug | то же | [mozillazg/go-unidecode](https://github.com/mozillazg/go-unidecode) | v0.2.0, go 1.12 | MIT | 146 звёзд, 88 импортёров | 1, коммитов нет | v0.2.0 2022-08-20 / 2023-05-14 | 1 | ю -> Iu, я -> ia, й -> i, ъ/ь -> `'` (§3.1) | Не брать (один мейнтейнер, заброшен, не совпадает) |
| 3. gitbranch.Slug | то же | [mehanizm/iuliia-go](https://github.com/mehanizm/iuliia-go) (схемы ГОСТ 7.79, Wikipedia и др.) | v1.0.2, go 1.13 | MIT | 56 звёзд, 8 импортёров | 1, коммитов нет | v1.0.2 2021-06-15 / 2025-01-08 | 1 | `Gost_779_alt` (ГОСТ 7.79 Б): ц -> c, щ -> shh, х -> x, й -> j, ъ -> два обратных апострофа (§3.1); `Wikipedia`: ближе, но е после ъ -> ye (`Syezd`), конечное -ий -> y | Не брать (один мейнтейнер, 8 импортёров, схемы не совпадают) |
| 3. gitbranch.Slug | то же | [essentialkaos/translit/v2](https://github.com/essentialkaos/translit) | v2.1.3, go 1.18 | Apache-2.0 | 38 звёзд, 11 импортёров | org essentialkaos, коммитов нет | v2.1.3 2024-04-02 / 2025-04-29 | 1 | Схемы ISO9, BGN, PCGN, ALALC и др.; ГОСТ нет. BGN выдаёт `′`, `″`, `ё` -> `ë` (§3.1) | Не брать (нет ГОСТ, мало пользователей) |
| 4. text.OSC52 | 13 строк | [aymanbagabas/go-osc52/v2](https://github.com/aymanbagabas/go-osc52) | v2.0.1, go 1.16 | MIT | 48 звёзд, 32 импортёра | 1, коммитов нет | v2.0.1 2023-03-06 / 2023-06-21 | 1 | `New(s).String()` и `.Tmux().String()` дают те же байты, что наш `OSC52(text, tmux)` (проверено на `hello`); есть ещё Screen, Query, Clear и лимит длины. Это зависимость lipgloss и bubbletea - при TUI в графе уже есть | Оставить (13 строк, один мейнтейнер, нет активности). Пересмотреть, если после выбора TUI-стека модуль уже в go.mod |
| 5. text.EditSession | временный файл, строки-подсказки, `sh -c`, 78 строк | [charmbracelet/x/editor](https://github.com/charmbracelet/x/tree/main/editor) | v0.2.0, go 1.24.0 | MIT | 323 звезды (монорепо x), 13 импортёров | org Charm, 4+ человек | v0.2.0 2026-01-09 / 2026-10-18 | 1 | Заменяет только сборку команды (`Cmd(app, path, LineNumber(n))`): `$EDITOR` делится по пробелам без учёта кавычек (`sh -c 'x'` сохраняет кавычки как есть), `$VISUAL` не читает, без редактора - `nano`. Временный файл, подсказки, `Result`, `Close` остаются нашими. Наш `sh -c '<editor> "$1"'` корректнее для редакторов с кавычками и путей с пробелами | Оставить (покрывает 3 строки из 78 и ведёт себя хуже) |
| 6. cache.FileStore.Save | CreateTemp, Write, Close, Chmod, Rename; 15 строк из 87 | [google/renameio/v2](https://github.com/google/renameio) | v2.0.1, go 1.13 (v2.0.2 требует go 1.25) | Apache-2.0 | 681 звезда, 122 импортёра | org google, фактически 1 (stapelberg: 9 из 10) | v2.0.2 2026-01-10 / 2026-01-10 | 1 | `WriteFile(path, data, 0o600)`: временный файл в том же каталоге, fsync, rename, очистка при ошибке; наш код fsync не делает. `MkdirAll` остаётся. Сборка только `!windows` (платформы Windows в плане нет - grep по `docs/plan`, `docs/contracts`; не проверено у пользователя) | Заменить (Google, 9 месяцев без коммитов - граница критерия; выигрыш невелик, но добавляет fsync) |
| 6. cache.FileStore.Save | то же | [natefinch/atomic](https://github.com/natefinch/atomic) | v1.0.1, go 1.12 | MIT | 211 звёзд, 130 импортёров | 1, коммитов нет | v1.0.1 2021-06-28 / 2021-06-28 | 1 | `WriteFile(path, reader)`, права берёт от старого файла | Не брать (заброшен, один человек) |
| 7. plane.do: повторы 429/503 | цикл, `retryDelay` с Retry-After и X-RateLimit-Reset, `sleepCtx`; 126 строк с transport | [hashicorp/go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) | v0.7.8, go 1.23 | MPL-2.0 | 2352 звезды, 2571 импортёр | org hashicorp, коммиты в основном ботом | v0.7.8 2025-06-18 / 2026-04-14 | 2 (go-cleanhttp) | Замер (§3.3): `Retry-After` в секундах и датой уважается без верхней границы (`120` - ждёт до отмены ctx, 60 с нет); `X-RateLimit-Reset` игнорируется (пауза 100 мс); по умолчанию повторяет любые 5xx и сетевые ошибки (у нас только 429/503); ctx прерывает ожидание. Для контракта нужны свои `CheckRetry` и `Backoff` - по объёму это наш `retryDelay` | Оставить (после настройки своего кода столько же, плюс MPL-2.0 и ещё один модуль) |
| 7. plane.do: повторы 429/503 | то же | [cenkalti/backoff/v5](https://github.com/cenkalti/backoff) | v5.0.3, go 1.23 | MIT | 4086 звёзд, 524 импортёра | 3+ (jakealti 13, cenkalti 4, ещё 3) | v5.0.3 2025-07-23 / 2026-10-19 | 1 | `Retry(ctx, op, WithMaxTries)`, `RetryAfter(seconds int)` для паузы от сервера (только целые секунды); разбор заголовков и X-RateLimit-Reset остаётся нашим. Не запускался | Оставить (заменяет только цикл и `sleepCtx`, заголовки всё равно свои) |
| 8. forge.New: определение gitlab/github | `strings.Contains(remoteURL, "gitlab")`; разбора URL в коде нет | [whilp/git-urls](https://github.com/whilp/git-urls) | v1.0.0, go 1.13 | MIT | 34 звезды, 122 импортёра | 1, коммитов нет | v1.0.0 2020-08-27 / 2020-09-17 | 1 | Разбирает scp-форму и ssh/https: `git@github.com:org/repo.git` -> host `github.com`, path `org/repo.git`; `ssh://git@host:2222/...` -> host `host:2222` (§3.3) | Не брать (заброшен) |
| 8. forge.New | то же | [chainguard-dev/git-urls](https://github.com/chainguard-dev/git-urls) (форк whilp) | v1.0.2, go 1.19 | MIT | 7 звёзд, 56 импортёров | org chainguard-dev, 1 человек (2 коммита) | v1.0.2 2023-12-21 / 2026-06-25 | 1 | Тот же API; не запускался отдельно | Оставить (потребности в разборе нет: проверка подстроки достаточна; не заводить зависимость) |
| 8. forge.New | то же | [kubescape/go-git-url](https://github.com/kubescape/go-git-url) | v0.0.33, go 1.19 | Apache-2.0 | 16 звёзд, 25 импортёров | org kubescape, 1 человек (5 из 6) | v0.0.33 2026-08-20 / 2026-08-20 | 3 (git-urls, k8s.io/utils) | Владелец/репо/провайдер для github, gitlab, azure, bitbucket; неизвестный хост (`git.example.com`) определяет как gitlab; версия 0.x | Не брать (мало пользователей, k8s.io/utils, 0.x) |
| 9. app.FormatDuration | ~10 строк, формат `1h 25m` | [hako/durafmt](https://github.com/hako/durafmt) | только псевдоверсия 2021-06-08, go 1.11 | MIT | 511 звёзд, 496 импортёров | 1, коммитов нет | релизов нет / 2021-06-08 | 1 | Формат по умолчанию «1 hour 25 minutes» - не контрактный; не запускался | Оставить (заброшен, формат другой) |
| 9. app.Diff (New/Updated по ID) | сравнение списков задач | [r3labs/diff/v3](https://github.com/r3labs/diff) | v3.0.2, go 1.23.5 | MPL-2.0 | 981 звезда, 97 импортёров | 1 | v3.0.2 2025-08-10 / 2025-08-10 | 3 | Структурный diff значений; семантику «новая/изменённая задача по ID, порядок не важен» не даёт - не запускался | Оставить (доменная логика; код `internal/app` в этой ветке ещё не написан) |
| 10. TUI: каркас | - | [bubbletea](https://github.com/charmbracelet/bubbletea) | v1.3.10, go 1.24.0 (ветка v2: модуль `charm.land/bubbletea/v2`, v2.1.0 требует go 1.26.0, ранняя v2.0.2 - 1.24.2) | MIT | 45465 звёзд, 14989 импортёров | org Charm, 4+ человек (aymanbagabas 45, meowgorithm 13, andrinoff 7) | v1.3.10 2025-09-17, v2.1.0 2026-10-08 / 2026-10-08 | 21 вместе с lipgloss, textinput, viewport, key, help | Контракт C4 и go 1.24 - только ветка v1; активная разработка ушла в v2 (последний релиз v1 - сентябрь 2025), v2 по go не подходит | Брать v1.3.10; риск: v1 без новых релизов (вывод из дат релизов) |
| 10. TUI: таблица задач | свой `cursorList` и `layout` (контракт) | [bubbles/table](https://github.com/charmbracelet/bubbles) | v1.0.0, go 1.24.2 | MIT | 8999 звёзд (bubbles), 693 импортёра пакета | org Charm | v1.0.0 2026-02-09 (v2.2.1 2026-08-24 - go 1.25) / 2026-10-01 | 21 | Колонки только с фиксированной `Width` (нет «название забирает остаток»); ячейка - строка, обрезка `runewidth.Truncate` не учитывает ANSI-последовательности (исходник `table.go`, строки 422, 435); подсветки отдельных рун и ячеек нет, стиль только строки и выбранной строки | Не брать: контракт требует гибких колонок и подсветки совпавших рун (`search.Highlight`) - таблицу рисовать на lipgloss самим |
| 10. TUI: строка поиска, ввод | - | [bubbles/textinput](https://github.com/charmbracelet/bubbles) | v1.0.0, go 1.24.2 | MIT | 2804 импортёра | org Charm | то же | + atotto/clipboard | `CharLimit` (нужные 32 руны), `Placeholder`, курсор, вставка; подходит для строки поиска, диалога создания и пикеров | Брать |
| 10. TUI: превью | - | [bubbles/viewport](https://github.com/charmbracelet/bubbles) | v1.0.0, go 1.24.2 | MIT | 1256 импортёров | org Charm | то же | 21 | Прокрутка текста `↑↓`, `PgUp/PgDn`; подсветку внутри не проверял | Брать для текста карточки и превью |
| 10. TUI: подсказки и справка | свой реестр `action`, строка подсказок и оверлей `?` (контракт C4) | bubbles/key + bubbles/help | v1.0.0, go 1.24.2 | MIT | org Charm | то же | то же | 21 | `key.Binding` дублирует `action.Keys` и `Hint`; `help.Model` рисует короткую и полную справку с усечением `…`. Не запускался | Оставить свой реестр (он источник для палитры и справки), `help` - по усмотрению PT-5a |
| 10. TUI: нечёткий список | - | bubbles/list | v1.0.0, go 1.24.2 | MIT | org Charm | то же | то же | + sahilm/fuzzy | Свой фильтр на sahilm/fuzzy и свой интерфейс; контракт использует fzf и свой индекс | Не брать |
| 10. TUI: стили | `config.Color`, `styles.go` | [lipgloss](https://github.com/charmbracelet/lipgloss) | v1.1.0, go 1.18 (v2.0.6 - go 1.25.0) | MIT | 11937 звёзд, 9630 импортёров | org Charm, 4+ человек | v1.1.0 2025-03-12 / 2026-10-09 | 13 | Стили, рамки, ширина, склейка панелей; поддержка `NO_COLOR`/профиля цвета - не проверено | Брать (зависимость bubbletea) |
| 10. TUI: Markdown в терминале | - | [glamour](https://github.com/charmbracelet/glamour) | v1.0.0, go 1.24.0 (v2.0.1 - `charm.land/glamour/v2`, go 1.25.8) | MIT | 3738 звёзд, 839 импортёров | org Charm (aymanbagabas 12, taciturnaxolotl 7) | v1.0.0 2025-11-24 / 2026-10-15 | 29 (chroma, bluemonday, x/net, x/text, goldmark-emoji и др.) | Тяжёлая зависимость, но выбор уже сделан в C4 (описание с подсветкой Markdown); go.mod требует lipgloss v1.1.1 (псевдоверсия) | Брать v1.0.0 |
| 10. TUI: открытие в браузере | `Deps.OpenURL` nil -> xdg-open/open (контракт C4) | [cli/browser](https://github.com/cli/browser) | v1.3.0, go 1.21 | BSD-2-Clause | 99 звёзд, 97 импортёров | org cli (GitHub CLI): williammartin 25, ещё 4 | v1.3.0 2023-10-05 / 2026-10-12 | 2 (x/sys) | `OpenURL(url)`: Linux - первый найденный из xdg-open, x-www-browser, www-browser, wslview; macOS, Windows, BSD. Выводит stdout/stderr команды в `os.Stdout/Stderr` - в TUI обязательно `browser.Stdout = io.Discard`, `browser.Stderr = io.Discard`. Реальное открытие не запускалось | Заменить (код активно поддерживается, форк pkg/browser) |
| 10. TUI: открытие в браузере | то же | [pkg/browser](https://github.com/pkg/browser) | только псевдоверсия 2024-01-02, go 1.14 | BSD-2-Clause | 618 звёзд, 1790 импортёров | org pkg, коммитов за 12 мес нет | релизов нет / 2024-01-02 | 2 | Тот же API; основа cli/browser | Не брать (нет активности, нет релизов) |
| 11. cli: разбор подкоманд | stdlib `flag`, `FlagSet` на подкоманду (контракт C5) | [spf13/cobra](https://github.com/spf13/cobra) | v1.10.2, go 1.15 | Apache-2.0 | 44709 звёзд, 195884 импортёра | 6 разных авторов за год (marckhouzam и др.) | v1.10.2 2025-12-03 / 2026-07-11 | 3 (pflag, mousetrap) | Флаги после позиционных аргументов работают: `show PROJ-42 --json`; в stdlib `flag` - нет (§3.4), а контракт пишет `show <ref> [--json]`. Размер бинарника: +1.0 МБ (3.38 против 2.34 МБ для hello). Коды выхода 0/1/2, тихий `prompt`, справка в stdout - настраиваются (`SilenceUsage`, `SilenceErrors`, свой тип ошибки), не запускалось | Заменить (условно, решение шлюза: C5 переписать под cobra) либо оставить `flag` и поправить контракт (флаги до позиционных) / 15 строк перестановки |
| 11. cli | то же | [alecthomas/kong](https://github.com/alecthomas/kong) | v1.16.1, go 1.20 | MIT | 3190 звёзд, 3079 импортёров | 5+ разовых авторов и бот renovate | релиз на GH не помечен; тег v1.16.1 2026-08-09 / 2026-08-28 | 1 | Подкоманды через теги структур; 0 зависимостей; не запускался | Не брать (cobra мощнее по пользователям; kong - запасной вариант) |
| 11. cli | то же | [urfave/cli/v3](https://github.com/urfave/cli) | v3.14.0, go 1.22 | MIT | 24294 звёзды, 4439 импортёров | org urfave, 6+ | v3.14.0 2026-10-02 / 2026-10-20 | 1 | API v3 свежий; не запускался | Не брать (cobra распространённее) |
| 12. config: Duration в YAML | `Refresh.UnmarshalYAML`, 44 строки | yaml.v3 (уже в go.mod) | v3.0.1 | MIT/Apache-2.0 | - | - | - | 0 новых | Замер (§3.5): `interval: 90s` декодируется в `*time.Duration` сам; голый `0` и ошибка в понятной форме - нет; контракт требует `0` = выключено и проверку неизвестных ключей | Оставить (библиотеки не нужны; свой код тонкий) |
| 13. search.Fuzzy | обёртка над fzf (решение пользователя) | [junegunn/fzf](https://github.com/junegunn/fzf) пакеты `src/algo`, `src/util` | v0.74.5, go 1.23.0 | MIT | 83689 звёзд, 36 импортёров `src/algo` | 1 основной (junegunn 77 из 100) + разовые | v0.74.5 2026-10-11 / 2026-10-17 | 5 (go-shellwords, x/sys, go-isatty, uniseg) | Собирается под go 1.24. Замер (§3.6): позиции возвращаются по убыванию (`[14 13 12 11 10]`), контракт C9 пишет «по возрастанию» - нужна сортировка; перед `FuzzyMatchV2` нужен `algo.Init("default")` (без него оценка 96 вместо 140); кириллица совпала | Оставить выбор (решение пользователя); критерий мейнтейнеров формально не выполняется - один основной автор, но проект самый распространённый; стабильность API `src/algo` не проверялась |
| 13. search.Fuzzy | то же | [sahilm/fuzzy](https://github.com/sahilm/fuzzy) | v0.1.1 (без директивы go; v0.1.2 и v0.1.3 требуют go 1.24.5) | MIT | 1449 звёзд, 419 импортёров | 1 (sahilm 5 из 7) | v0.1.3 2026-06-11 / 2026-06-24 | 1 | Запасной вариант; используется в `bubbles/list`; не запускался | Не брать (один мейнтейнер; fzf выбран) |
| 13. search.Fuzzy | то же | [lithammer/fuzzysearch](https://github.com/lithammer/fuzzysearch) | v1.1.8, go 1.15 | MIT | 1323 звезды, 368 импортёров | 1 (lithammer 15, остальное - dependabot) | v1.1.8 2023-05-09 / 2026-10-01 | 1 | Подпоследовательность без оценки по границам слов; не запускался | Не брать |

## 3. Замеры и примеры

### 3.1. Slug и транслитерация: вход -> наш -> библиотека

Контракт C2: ГОСТ 7.79-2000 схема Б без диакритики, щ -> shch, ж -> zh, х -> kh, ц -> ts, ч -> ch, ш -> sh, ю -> yu, я -> ya, ё -> yo, й -> y, ъ/ь -> пусто, ы -> y, э -> e. Наш код реализует именно эту таблицу. Остальные колонки - вывод библиотек на тех же входах (slug-режим: `gosimple/slug.MakeLang(s, "ru")`; `go-unidecode`, `iuliia` и `translit` - как есть, без чистки).

| Вход | Наш | gosimple/slug (ru) | go-unidecode | iuliia ГОСТ 7.79 Б | iuliia Wikipedia | translit BGN |
|---|---|---|---|---|---|---|
| Щука и жираф | shchuka-i-zhiraf | shchuka-i-zhiraf | Shchuka i zhiraf | Shhuka i zhiraf | Shchuka i zhiraf | Shchuka i zhiraf |
| Хлеб цельный | khleb-tselnyy | khleb-tsel-nyi | Khleb tsel'nyi | Xleb cel`ny`j | Khleb tselny | Khleb tsel′nyy |
| Юла яблоко | yula-yabloko | iula-iabloko | Iula iabloko | Yula yabloko | Yula yabloko | Yula yabloko |
| Ёлка и йод | yolka-i-yod | iolka-i-iod | Iolka i iod | Yolka i jod | Yolka i yod | Yëlka i yod |
| Объект: ъ ь ы э | obekt-y-e | ob-ekt-y-e | Ob'ekt: ' ' y e | Ob``ekt: `` ` y` е` | Obyekt:   y e | Ob″ekt: ″ ′ y e |
| Исправить логин на странице | ispravit-login-na-stranitse | ispravit-login-na-stranitse | Ispravit' login na stranitse | Ispravit` login na stranice | Ispravit login na stranitse | Ispravit′ login na stranitse |
| Съезд, объезд, подъём | sezd-obezd-podyom | s-ezd-ob-ezd-pod-iom | S'ezd, ob'ezd, pod'iom | S``ezd, ob``ezd, pod``yom | Syezd, obyezd, podyom | S″ezd, ob″ezd, pod″ëm |
| Чай шашлык | chay-shashlyk | chai-shashlyk | Chai shashlyk | Chaj shashly`k | Chay shashlyk | Chay shashlyk |
| Fix login page (v2)! | fix-login-page-v2 | fix-login-page-v2 | Fix login page (v2)! | то же | то же | то же |
| Мягкий знак: соль | myagkiy-znak-sol | miagkii-znak-sol | Miagkii znak: sol' | Myagkij znak: sol` | Myagky znak: sol | Myagkiy znak: sol′ |

Выводы:

- Ни одна библиотека не даёт нашу таблицу: слаг у gosimple/slug совпал в 3 строках из 10 (латиница и слова без ю, я, й, ъ, ь, ё), у остальных - сырой текст с апострофами, который всё равно пришлось бы чистить своим кодом.
- Расхождение с контрактом: таблица C2 названа «ГОСТ 7.79-2000 схема Б», но настоящая схема Б (iuliia `gost_779_alt`) даёт ц -> c, щ -> shh, х -> x, й -> j, ъ -> два обратных апострофа. Наша таблица ближе к BGN/PCGN без диакритики. Поведение кода не меняется, но формулировку в C2 стоит исправить («собственная таблица, близкая к BGN/PCGN»).
- Обрезка: `gosimple/slug` с `MaxLength = 40` режет по границе слова (`EnableSmartTruncate`), а контракт - по символу; на строке из 77 символов получилось `ochen-dlinnoe-nazvanie-zadachi-kotoroe` против нашего `ochen-dlinnoe-nazvanie-zadachi-kotoroe-n`.

### 3.2. HTML -> Markdown и текст на 12 примерах

Примеры: жирный/курсив/зачёркнутый/код; заголовок и `<br>`; ссылки; вложенные списки и нумерованный; цитата и `<hr>`; `<pre><code>` с тройным обратным апострофом; таблица; спецсимволы Markdown в тексте; `<div>`, `<script>`, `<span>`, `<img>`; список задач; `&nbsp;` и пустой абзац; неправильно вложенные теги. Сравнение - свой `HTMLToMarkdown` и html-to-markdown/v2 v2.5.0 с плагинами base, commonmark (`WithHorizontalRule("---")`, `WithListEndComment(false)`), strikethrough, table.

| Пример | html-to-markdown v2 относительно нашего |
|---|---|
| 1 (инлайн), 3 (ссылки), 6 (pre), 11 (nbsp) | побайтно то же |
| 2 (`<br>`) | жёсткий перенос - два пробела вместо `\` (оба валидны в CommonMark) |
| 4 (вложенный список) | после «one» пустая строка перед вложенным списком (рыхлый список) |
| 5 (цитата, hr) | `>` без пробела на пустой строке у нас, у них `> `; hr - как настроено |
| 7 (таблица) | разделитель `\|---\|---\|` вместо `\| --- \| --- \|`; без плагина table таблица превращается в `AB12` |
| 8 (спецсимволы) | `<b>` экранируется как `&lt;b&gt;`, `*not bold*` - `\*not bold*`; у нас `\<b\>` |
| 9 (script, img) | текст скрипта отбрасывается (у нас `alert(1)` протекает в вывод) |
| 10 (task list) | пункты без пустых строк между ними (у нас рыхлый список) |
| 12 (кривая вложенность) | `**broken bold *both* tail**` против нашего `**bold *both*** tail` |

Побайтно совпало 4 из 12 (без плагинов strikethrough и table - 3 из 12). Равенство после повторного `MarkdownToHTML` (критерий обратимости контракта) на этих примерах не проверялось.

HTMLToText (наш regexp и k3a/html2text v1.5.1): на `<ul><li>one<ul><li>nested</li></ul></li><li>two</li></ul>` наш вывод `- one- nested` (склейка вложенного пункта - дефект своего кода, не следствие контракта), у k3a - `one`, `nested`, `two` без маркеров; ссылки у k3a раскрываются в URL; таблица `AB12` у k3a, у нас `AB` и `12` на разных строках; пример 10 (список задач) у k3a даёт много пустых строк.

### 3.3. Повторы запросов, URL remote, редактор

go-retryablehttp v0.7.8 против httptest-сервера (RetryWaitMin 100 мс, RetryMax 3, ctx):

| Сценарий сервера | Результат |
|---|---|
| 429, `Retry-After: 1`, затем 200 | 200 за 1.01 с, 2 попытки |
| 429, только `X-RateLimit-Reset` = +2 с, затем 200 | 200 за 0.10 с, 2 попытки: заголовок игнорируется |
| 503, `Retry-After: 120`, ctx 3 с | `context deadline exceeded` на 3 с, 1 попытка: ожидание 120 с без верхней границы |
| 500 | 4 попытки за 0.7 с, затем ошибка (повтор любых 5xx) |
| 404 | 1 попытка, ответ 404 |

Разбор URL remote: whilp/git-urls v1.0.0 - `git@github.com:org/repo.git` -> ssh, `github.com`, `org/repo.git`; `https://gitlab.com/group/sub/repo.git` -> https, `gitlab.com`; `ssh://git@host:2222/org/repo` -> `host:2222`; локальный путь -> `file`. kubescape/go-git-url v0.0.33: владелец `group/sub`, репо `repo`, провайдер gitlab; неизвестный хост `git.example.com` - провайдер gitlab.

x/editor v0.2.0 при `EDITOR=vim -u NONE`: `[vim -u NONE /tmp/x.md]`, с `LineNumber(7)` - `+7`; при `EDITOR="sh -c 'x'"` - `[sh -c 'x' /tmp/x.md]` (кавычки остались внутри аргумента); без `EDITOR` - `nano`.

### 3.4. Разбор флагов подкоманд

| Команда | stdlib `flag` | cobra |
|---|---|---|
| `show --json PROJ-42` | `json=true`, args `[PROJ-42]` | то же |
| `show PROJ-42 --json` | `json=false`, args `[PROJ-42 --json]` | `json=true`, args `[PROJ-42]` |

Контракт C5 описывает `show <ref> [--json]` и `branch [<ref>] [--print] [--worktree]`; со стандартным `flag` флаги после `ref` не разбираются - либо менять порядок в контракте и справке, либо переставлять аргументы перед `Parse`, либо брать cobra/pflag.

### 3.5. Duration в YAML

yaml.v3 v3.0.1 в `*time.Duration`: `90s` -> 1m30s, `2m` -> 2m0s; `0` и `5` - ошибка «cannot unmarshal !!int into time.Duration»; `abc` - ошибка без подсказки формата. Свой `UnmarshalYAML` нужен ради `0` и понятного сообщения.

### 3.6. fzf/src/algo

`algo.FuzzyMatchV2(false, true, true, &chars, []rune("логин"), true, slab)` на «Исправить логин на странице»: начало 10, конец 15, оценка 140, позиции `[14 13 12 11 10]` (по убыванию). Без `algo.Init("default")` оценка 96. Кириллица и нормализация работают.

## 4. Итоги для координатора

Заменить:

1. `cache.FileStore.Save` - `github.com/google/renameio/v2 v2.0.1` (добавляет fsync; v2.0.2 требует go 1.25; только не-Windows).
2. Открытие URL - `github.com/cli/browser v1.3.0`, с `Stdout`/`Stderr` = `io.Discard`.
3. `internal/cli` - `github.com/spf13/cobra v1.10.2` (условно: либо cobra, либо правка C5 под порядок флагов; решение шлюза).

TUI-стек (контракт C4): `bubbletea v1.3.10`, `bubbles v1.0.0` (textinput, viewport; help по желанию), `lipgloss v1.1.0`, `glamour v1.0.0`. Не брать `bubbles/table` и `bubbles/list`. Все версии v1; v2-ветки требуют go 1.25 или выше. `bubbles v1.0.0` объявляет `go 1.24.2`, поэтому `go get` поднял директиву песочницы с `1.24` до `1.24.2` - это выше строки `go 1.24` в go.mod проекта (тулчейн 1.24.2 в среде есть, но директиву придётся поднять вместе с добавлением).

Оставить своим: HTMLToMarkdown (html-to-markdown - единственный кандидат, один мейнтейнер, v2.5.1+ требуют go 1.25), HTMLToText, Slug и транслитерация, OSC52, EditSession, повторы 429/503, разбор URL remote, `FormatDuration`, `Diff`, `config.Duration`.

Решения за пользователем: (а) html-to-markdown v2.5.0 вместо 547 строк, если принять один мейнтейнер; (б) cobra или перестановка флагов; (в) fzf - критерий мейнтейнеров формально не выполняется, но выбор пользователя.

## 5. Воспроизведение

Каталог-песочница вне репозитория (пример рабочего каталога - временная папка):

```
export GOTOOLCHAIN=local
mkdir lab && cd lab && printf 'module lab\n\ngo 1.24\n' > go.mod
go list -m -versions <модуль>@latest
go mod download -json <модуль>@<версия>      # поле GoMod -> директива go
go get <модуль>@<версия>                      # итоговая директива go в go.mod песочницы
go list -deps -f '{{with .Module}}{{.Path}}{{end}}' ./<программа> | sort -u
gh api repos/<владелец>/<репо>                # звёзды, владелец, лицензия
gh api "repos/<владелец>/<репо>/commits?since=<дата>&per_page=100"
curl -s "https://pkg.go.dev/<пакет>?tab=importedby" | grep -o 'Known importers:</strong> [0-9,]*'
```

Программы замеров: §3.1 - вывод `slug.MakeLang`, `unidecode.Unidecode`, `iuliia.Gost_779_alt.Translate`, `iuliia.Wikipedia.Translate`, `translit.BGN` рядом с копией `internal/gitbranch/slug.go`; §3.2 - `converter.NewConverter` с плагинами и копия `internal/text/html2md*.go`; §3.3 - `httptest` с `retryablehttp.NewClient`; §3.4 - `flag.NewFlagSet` и `cobra.Command`; §3.5 - `yaml.Unmarshal` в структуру с `*time.Duration`; §3.6 - `algo.FuzzyMatchV2`. Исходники в репозиторий не добавлялись (скоуп задачи - один документ); при необходимости - отдельная задача.

Лицензии кандидатов: см. таблицу; MPL-2.0 (gosimple/slug, go-retryablehttp, r3labs/diff) - слабый копилефт на уровне файлов, для вариантов «оставить» не актуально.
