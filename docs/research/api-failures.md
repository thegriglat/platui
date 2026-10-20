---
type: research
status: active
module: tui
updated: 2026-10-20
summary: Аудит недоступности API Plane (PT-45): таблица «как есть / как надо» с тестами TestFailure*, пробелы GAP и список доработок
conclusion: 9 доработок - офлайн-старт с кэша, таймауты в конфиге, коды выхода CLI, потеря текста из редактора, ложное время кэша, валидация ответов, Retry-After-дата, ключ в теле ошибки, ошибка после восстановления; остальное ведёт себя как надо
data: Тесты TestFailure* в internal/{plane,app,tui,cli}/failure_test.go - httptest (закрытый порт, .invalid, Hijack, 5xx, 429, 401/403/404, TLS) и planetest.Fake.Err; прогон go test -race 2026-10-20
applied_in: []
---

# Недоступность API Plane: аудит

Задача PT-45, решение 2026-10-20 (`.dpcli/dpcli plan tui «2. Решения»`, пункт о недоступности API). Код не менялся: добавлены тесты `internal/{plane,app,tui,cli}/failure_test.go` (имена `TestFailure<Слой><Сценарий>`). Где поведение не соответствует ожидаемому, тест проверяет «как надо» и пропускается через `t.Skip("GAP: …")`; пока GAP не закрыт, его причина видна в `go test -v -run TestFailure`. Закрытие GAP превращает SKIP в PASS без правки теста.

Как проверялось: слой plane — настоящий клиент против `httptest` (закрытый порт, адрес `.invalid`, сон дольше `Client.Timeout` в 50–100 мс, `Hijack` с обрывом, 5xx, 429, 401/403/404, `NewTLSServer` без доверия, HTML и обрезанный JSON). app/tui/cli — фейк `planetest.Fake.Err` и сквозные прогоны с настоящим клиентом.

Обозначения: тест — функция `TestFailure…` в `failure_test.go` своего слоя; `ок` — как надо и так и есть; `GAP-n` — пробел, номер доработки в списке ниже.

## Таблица сценариев

| сценарий | слой | как есть (наблюдаемо, тест) | как надо | GAP/ок |
|---|---|---|---|---|
| Нет сети / DNS | plane | ошибка `plane: GET request failed: … no such host`; не `APIError`, не 401; ключа и URL в тексте нет; повторов нет (`TestFailurePlaneNoNetworkDNS`) | то же; повторы — уровнем опроса | ок |
| Нет сети / DNS | app | ошибка пробрасывается, память и кэш не тронуты; `Me`, `Projects`, `Meta` кэш на диске не читают, `Find` без сети не работает даже для задач из кэша; читается только `CachedItems` (`TestFailureAppOfflineStartFromDisk`) | старт с кэша: проекты, справочники, «я» и задачи с диска | GAP-1 |
| Нет сети / DNS | tui | экран жив, красная ошибка в статусной строке, действия без проекта не падают (`TestFailureTUIBootNoNetwork`); при кэше на диске всё равно пусто: `bootstrap` падает на `Me`; пометки «офлайн, данные от ЧЧ:ММ» нет (`TestFailureTUIOfflineStartFromCache`, `TestFailureTUIItemsDownShowsCache`) | старт с кэша, пометка «офлайн, данные от ЧЧ:ММ» | GAP-1 |
| Нет сети / DNS | cli | `platui: <текст>` в stderr, одна строка, stdout пуст, код 1 (`TestFailureCLIRealScenarios/dns`, `TestFailureCLIFakeErrors`) | stderr и ненулевой код; код отличим от 401 | GAP-3 |
| Отказ соединения | plane, app, tui, cli | `request failed`, без повторов, без ключа и адреса; в TUI/CLI то же, что для DNS (`TestFailurePlaneConnectionRefused`, `TestFailureAppE2EClosedPortAndTLS`, `TestFailureTUIE2EErrorsReadable/refused`, `TestFailureCLIRealScenarios/refused`) | то же | ок |
| Таймаут | plane | `Client.Timeout` 30 с по умолчанию; ошибка — `net.Error` с `Timeout()`; отмена по `ctx` работает (`TestFailurePlaneTimeout`, `TestFailurePlaneCtxCancelDuringCall`) | то же, значение настраивается | GAP-2 |
| Таймаут | tui, cli | не зависают сверх таймаута клиента, статус `…Client.Timeout exceeded…`; Ctrl+C (`ctx`) прерывает запрос в CLI (`TestFailureTUIE2EErrorsReadable/timeout`, `TestFailureCLIRealScenarios/timeout`, `TestFailureCLICtrlC`) | то же | ок |
| Таймаут и повторы в конфиге | config, cmd | в конфиге нет полей таймаута и числа повторов; 30 с и 3 повтора зашиты в `plane.New`, `main.go` не передаёт `HTTPClient` (`TestFailureCLITimeoutsConfigurable`) | `http.timeout`, `http.retries` в YAML, передаются в `plane.Options` | GAP-2 |
| Обрыв посреди ответа | plane | `plane: reading response: unexpected EOF`, данных нет, повтора нет (`TestFailurePlaneMidBodyCut`) | то же | ок |
| Обрыв посреди пагинации | plane, app, cli | 2-я страница: обрыв, 500 или 429 — ошибка и `nil` вместо частичного списка; кэш в памяти и на диске остаётся полным; после возврата связи обновление проходит (`TestFailurePlaneMidPagination`, `TestFailureAppE2EPartialPaginationKeepsFullList`, `TestFailureCLIRealScenarios/cut-page2`) | частичная пагинация не подменяет полный список | ок |
| 5xx | plane | 500 и 502 — одна попытка, `APIError{Status}`; 503 — 3 повтора (1 с, 2 с, 4 с), затем ошибка; HTML-страница прокси в текст не попадает (`TestFailurePlane5xx`, `TestFailureAppE2EErrorsTextAndSentinels`) | ошибка понятным текстом, повтор — опросом с backoff | ок |
| 429 и Retry-After | plane | Retry-After в секундах соблюдается, потолок 60 с, до 3 повторов, затем `ErrRateLimited`; отмена `ctx` во время ожидания прерывает его; после 429 запрос восстанавливается (`TestFailurePlane429RetryAfter`, `TestFailurePlane429RecoversAndCaps`, `TestFailurePlane429CancelDuringWait`) | то же | ок |
| 429 и Retry-After | plane | Retry-After в формате HTTP-date не разбирается: ждём 1 с, 2 с, 4 с (`TestFailurePlane429RetryAfterHTTPDate`) | уважать дату по RFC 9110 | GAP-5 |
| Backoff опроса | app, tui | `NextPoll`: удвоение до 5 мин (429 — до 10 мин), возврат к базе после успеха; опрос в полёте не плодит запросов; данные при ошибке не теряются (`TestFailureAppNextPollBackoff`, `TestFailureTUIPollBackoffAndRecovery`, `TestFailureTUIPollDoesNotPileUp`) | с backoff, без засыпания сообщениями, восстановление само | ок |
| Восстановление после сбоя | tui | интервал возвращается к базе, данные обновляются, но красная ошибка остаётся в статусной строке до следующего сообщения (`TestFailureTUIPollBackoffAndRecovery`) | ошибка снимается или «связь восстановлена» | GAP-8 |
| R при ошибке | tui | старые данные, табы и кэш остаются, ошибка в статусе без ключа, `loading` снят; следующий R — «обновлено: N задач» (`TestFailureTUIReloadKeepsDataAndRecovers`, `TestFailureAppReloadKeepsData`) | то же | ок |
| 401 / 403 | plane, app | `errors.Is(ErrUnauthorized)`, одна попытка, текст сервера в ошибке; `Project` и `Find` не принимают 401 за «нет проекта/задачи» (`TestFailurePlaneUnauthorized`, `TestFailureAppMeProjectsRetry`, `TestFailureAppFindErrors`) | то же | ок |
| 401 / 403 | tui | статус `plane: HTTP 401: …`, без ключа (`TestFailureTUIBootUnauthorized`) | то же | ок |
| 401 / 403 | cli | код 1 — такой же, как у сети; подсказки про `api_key` нет: `platui: plane: HTTP 401: Unauthorized` (`TestFailureCLIExitCodesDistinguishAuthFromNetwork`, `TestFailureCLIUnauthorizedHint`) | различимый код, подсказка проверить ключ | GAP-3 |
| 404 | plane, app, cli | `ErrNotFound`; `Find` — `ErrItemNotFound`; CLI: `work item not found`, код 1 (`TestFailurePlaneNotFound`, `TestFailureAppFindErrors`, `TestFailureCLIShowNotFound`) | то же | ок |
| TLS-ошибка | plane, app, tui, cli | `x509: certificate …`; запрос с ключом на сервер не уходит (`TestFailurePlaneTLS`, `TestFailureAppE2EClosedPortAndTLS`, `TestFailureTUIE2EErrorsReadable/tls`, `TestFailureCLIRealScenarios/tls`) | то же | ок |
| Неразборчивый ответ | plane | HTML и обрезанный JSON на 200 — ошибка `plane: bad …` во всех методах, тело ответа в текст не попадает; пустое тело списка — пустой список (`TestFailurePlaneGarbageResponse`) | ошибка | ок |
| Неразборчивый ответ | plane | `Me`: ошибка разбора без префикса `plane:` и контекста (`TestFailurePlaneMeGarbageHasContext`) | ошибка с контекстом | GAP-6 |
| Неразборчивый ответ | plane, app | 200 без `results` (или пустое тело) принимается как пустой список и затирает кэш задач; 200 `{}` на правку — «задача» без ID (`TestFailurePlaneListWithoutResults`, `TestFailurePlaneEmptyObjectAsItem`) | ошибка, кэш не тронут | GAP-6 |
| Ключ API в ошибках | plane, app, tui, cli | ключ уходит только в заголовке; ни на одном сценарии его нет в тексте ошибок, статусе и stderr (`TestFailurePlaneKeyNotInErrors`, `TestFailureAppE2EErrorsTextAndSentinels`, `TestFailureTUIE2EErrorsReadable`, `TestFailureCLIRealScenarios`) | то же | ок |
| Ключ API в ошибках | plane | текст ошибки берётся из тела ответа как есть: сервер, эхом возвращающий ключ, раскроет его (`TestFailurePlaneServerEchoesKey`) | ключ вырезается из текста | GAP-7 |
| Откат правок | tui | состояние (через `patchWith`), приоритет, «на себя», `patchItem` — задача в UI не меняется, статус `HTTP 503…`; канбан откатывается на исходную колонку, выделение сохраняется; после возврата связи правка проходит (`TestFailureTUIEditsDoNotChangeItems`, `TestFailureTUIKanbanMoveRollback`) | откат в UI и сообщение | ок |
| Откат правок | app | неудачные `Update`, `SetDescription`, `CreateQuick`, `Comment`, `Item`, `Comments` возвращают ошибку с сохранением `errors.Is`, кэш не меняется; пакетные операции дают ошибку по каждой задаче (`TestFailureAppEditsFail`, `TestFailureAppBulkPartial`) | то же | ок |
| Создание задачи | tui | при ошибке экран ввода остаётся с введённым текстом, `busy` снят; повторный Enter создаёт (`TestFailureTUICreateKeepsInput`) | текст не теряется | ок |
| Комментарий и описание из $EDITOR | tui | при ошибке отправки временный файл удаляется (`finishEdit`: `defer sess.Close()`), введённый текст теряется (`TestFailureTUIEditorTextSurvivesAPIError`) | текст не теряется: файл сохраняется или черновик | GAP-9 |
| «Начать работу» при недоступном Plane | app | ветка создана и отражена в результате, ошибка Plane возвращена, состояние задачи не менялось (`TestFailureAppStartPlaneDown`) | то же | ок |
| Время данных в кэше | app | правка одной задачи (`replaceItem`) обновляет время всего списка; «данные от ЧЧ:ММ» будет врать (`TestFailureAppItemsTimestampTracksFullList`) | время — только полной загрузки | GAP-4 |

## Доработки

По одной строке на доработку; координатор заводит задачи. Номер — из таблицы.

1. GAP-1: app + tui — офлайн-старт: `Me`, `Projects`, `Meta` читают кэш на диске при ошибке сети (и пишут его), `bootstrap` не падает на `Me`, статус «офлайн, данные от ЧЧ:ММ», `Find` по кэшу; файлы `internal/app/app.go`, `internal/app/items.go`, `internal/tui/msgs.go`, `internal/tui/update.go`, `internal/tui/status.go`; контракты C5, C4.
2. GAP-2: config + plane + cmd — таймаут запроса и число повторов в YAML (`http.timeout`, `http.retries`), передача в `plane.Options`; файлы `internal/config/*`, `cmd/platui/main.go`; контракты C1, C0.
3. GAP-3: cli — разные коды выхода для 401/403 и для сети/таймаута, подсказка проверить `api_key`/`api_key_env`; файлы `internal/cli/cli.go`; контракт C10.
4. GAP-4: app — время списка задач только у `RefreshItems` и `Reload`, не у `replaceItem` (отдельное время или без обновления); файлы `internal/app/items.go`, `internal/app/item.go`; контракт C5.
5. GAP-5: plane — Retry-After в формате HTTP-date; файл `internal/plane/transport.go` (`retryDelay`).
6. GAP-6: plane — проверка ответов: `Me` с префиксом `plane:`, ошибка для 200 без `results` и для `{}` вместо задачи; файлы `internal/plane/endpoints.go`, `internal/plane/paginate.go`.
7. GAP-7: plane — вырезать ключ клиента из текста ошибки из тела ответа; файл `internal/plane/transport.go` (`errorMessage`).
8. GAP-8: tui — снимать ошибку или показывать «связь восстановлена» при успешном опросе после ошибки; файл `internal/tui/update.go` (`onItems`).
9. GAP-9: tui — не терять текст из $EDITOR при ошибке отправки (оставлять файл и показывать путь, либо черновик в кэше); файл `internal/tui/edit.go`.

## Границы аудита

- TUI проверен на уровне модели Bubble Tea (`Update`, команды), не в терминале; фоновые тики обрезаются по таймауту тестовой обвязки.
- Пакетные операции TUI (`bulk.go`) проверены на уровне app (`BulkPatch`); отдельный сквозной тест на экране не писали.
- Повтор на сетевых ошибках в plane отсутствует намеренно (повторяются только 429 и 503); это не отмечено как GAP, backoff выполняет опрос.
- Подмена времени и таймаутов: сон 50–100 мс для таймаута, `Sleep` клиента подменён (пауза повторов не ждётся).
