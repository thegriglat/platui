---
type: "contract"
status: "closed"
module: "certs"
updated: "2026-10-20"
summary: "C1 v8 — http.ca_file (свой CA поверх системных), пакет internal/httpclient, app.IsCertError и подсказка в ошибках"
related: ["docs/plan/certs.md", "docs/contracts/tui.md", "docs/contracts/tui-features.md"]
contracts: [{"id": "C1", "version": 8}, {"id": "C13", "version": 1}, {"id": "C10", "version": 5}]
---

# Контракты модуля certs

## C1 v8. config — http.ca_file

Дополнение к C1 v7 (секция `http`); всё из v7 сохраняется, тесты v1…v7 не меняются.

```yaml
http:
  timeout: 30s
  retries: 3
  ca_file: ~/certs/k3s-ingress-ca.crt   # необязательно: PEM с доверенными CA (один или несколько), добавляются к системным
```

```go
// HTTP v7 + поле:
//   CAFile string `yaml:"ca_file"`
func (c *Config) HTTPCAFile() string // путь с раскрытым "~/" (os.UserHomeDir); пусто — не задан
```
- Validate: только синтаксис — пустая строка допустима; существование файла здесь не проверяется (чтобы `Parse` оставался чистым).
- `config.example.yml` — строка `ca_file` закомментирована, с пояснением; `TestContractConfigExample` проходит.

## C13 v1. internal/httpclient

Листовой пакет (импортирует только stdlib).

```go
type Options struct {
    Timeout time.Duration
    CAFile  string // путь (уже раскрытый); пусто — только системные CA
}
func New(o Options) (*http.Client, error)
```
- Пул: `x509.SystemCertPool()`; ошибка или nil → `x509.NewCertPool()`; затем `AppendCertsFromPEM(содержимое CAFile)`.
- Файл не читается → ошибка `http.ca_file: <путь>: <причина>`; нет ни одного сертификата PEM → `http.ca_file: <путь>: no PEM certificates found`.
- Транспорт — `http.DefaultTransport.(*http.Transport).Clone()`, `TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}`; при пустом CAFile `TLSClientConfig` не задаётся (поведение как сейчас).
- `cmd/platui/main.go` `build`: клиент Plane — `httpclient.New(httpclient.Options{Timeout: cfg.HTTPTimeout(), CAFile: cfg.HTTPCAFile()})`, ошибка возвращается из `build`.

## C10 v5 / C4 v9. Подсказка при ошибке сертификата

```go
// internal/app
func IsCertError(err error) bool // x509.UnknownAuthorityError, x509.CertificateInvalidError, x509.HostnameError,
                                 // *tls.CertificateVerificationError; по тексту — "x509:"
const CertHint = "сертификат сервера не доверен — укажите CA в http.ca_file"
```
- `IsNetworkError` не меняется (ошибка сертификата остаётся сетевой: exit 4, офлайн с кэша).
- CLI (`internal/cli`): при `IsCertError` — `platui: Plane недоступен: <err>` и следующей строкой `platui: <CertHint>`; код 4.
- TUI (`internal/tui` `errText`): при `IsCertError` к тексту статусной строки добавляется ` (<CertHint>)`.
- Контрактные тесты: `internal/httpclient` — `TestContractHTTPClient*`; `internal/app` — `TestContractCertError*`.
