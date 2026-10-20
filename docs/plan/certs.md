---
type: plan
status: closed
module: certs
updated: 2026-10-20
summary: Свой CA для HTTPS к Plane — http.ca_file в конфиге (добавляется к системным, как NODE_EXTRA_CA_CERTS), подсказка при ошибке сертификата
related: [docs/contracts/certs.md]
---

# План модуля certs — свои сертификаты для Plane

## 1. Цель
Подключение к self-hosted Plane за внутренним CA (пример: `https://plane.k3s.ashoo.local`, CA `pve5-k3s-ingress-ca`):
доверенный CA задаётся в конфиге и добавляется к системным, как `NODE_EXTRA_CA_CERTS` у Plane MCP.

## 2. Решения пользователя
- Нужен только свой CA в конфиге; клиентский сертификат (mTLS) и отключение проверки — не делать (2026-10-20).
- Поведение — как `NODE_EXTRA_CA_CERTS`: дополнительные сертификаты поверх системного хранилища, не замена.

## 3. Решения главной сессии
- Поле `http.ca_file` (C1 v8, контракт `docs/contracts/certs.md`): путь к PEM, в файле может быть несколько сертификатов; `~` раскрывается.
- Пул: `x509.SystemCertPool()` (ошибка/nil → новый пул) + сертификаты из файла; транспорт — клон `http.DefaultTransport` с `TLSClientConfig.RootCAs`, `MinVersion` TLS 1.2.
- Ошибки чтения файла и «в файле нет сертификатов» — при сборке клиента, до первого запроса, с путём в тексте (exit 1 в CLI).
- Ошибка проверки сертификата сервера по-прежнему сетевая (exit 4, офлайн с кэша), но текст дополняется подсказкой про `http.ca_file`.
- `glab` и `git push` используют свои настройки CA — вне скоупа, в README — одна строка-подсказка (`glab config set ca_cert … --host …`, `git config http.sslCAInfo`).

## 4. Задачи
- **CA-1** (dp-engineer) — `http.ca_file`: конфиг, сборка HTTP-клиента, подсказка в ошибках CLI/TUI, пример конфига и README; тесты на `httptest.NewTLSServer`.

## 5. Условия остановки
- Нужна правка контрактов других пакетов, кроме перечисленных в `docs/contracts/certs.md`.
