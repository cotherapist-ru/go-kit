# go-kit

[![CI](https://github.com/cotherapist-ru/go-kit/actions/workflows/ci.yml/badge.svg)](https://github.com/cotherapist-ru/go-kit/actions/workflows/ci.yml)

Общая Go-библиотека HTTP/инфраструктурного кода сервисов Cotherapist.

## Установка

```bash
go get github.com/cotherapist-ru/go-kit@v0.1.2
```

Module path совпадает с GitHub-репозиторием. Теги — semver с префиксом `v`.

## Подпакеты

| Импорт | Назначение |
|---|---|
| `github.com/cotherapist-ru/go-kit/logging` | slog Setup, request logger, context |
| `github.com/cotherapist-ru/go-kit/httpserver` | chi-роутер и graceful shutdown |
| `github.com/cotherapist-ru/go-kit/healthz` | `/healthz` plain и JSON |
| `github.com/cotherapist-ru/go-kit/httpjson` | `Write` JSON-ответов |
| `github.com/cotherapist-ru/go-kit/envconfig` | чтение env |
| `github.com/cotherapist-ru/go-kit/postgres` | pgx pool, bootstrap DB, SQL-миграции |
| `github.com/cotherapist-ru/go-kit/admintoken` | Bearer admin API |
| `github.com/cotherapist-ru/go-kit/servicetoken` | Bearer service-to-service |
| `github.com/cotherapist-ru/go-kit/captcha` | Yandex SmartCaptcha |
| `github.com/cotherapist-ru/go-kit/ratelimit` | in-memory limiter (ограничение числа ключей) |
| `github.com/cotherapist-ru/go-kit/clientip` | IP клиента за доверенными прокси (`TRUSTED_PROXIES`) |

## Пример

```go
log := logging.Setup("promo")
cfg, err := config.Load()
r := httpserver.NewRouter(httpserver.WithLogger(
    logging.RequestLogger(logging.WithNoisePaths("/styles.css")),
))
r.Get("/healthz", healthz.Plain)
httpserver.Run(httpserver.Options{Addr: ":" + cfg.Port, Handler: r})
```

## IP клиента и `TRUSTED_PROXIES`

`httpserver.NewRouter` подставляет в `r.RemoteAddr` реальный IP клиента через `clientip`
(вместо chi `middleware.RealIP`). Заголовки `X-Forwarded-For` / `X-Real-IP` учитываются только
если TCP-пир — доверенный прокси; `X-Forwarded-For` разбирается справа налево.

| `TRUSTED_PROXIES` | Поведение |
|---|---|
| не задан | loopback + частные сети (10/8, 172.16/12, 192.168/16, 100.64/10, fc00::/7) — ingress в кластере |
| `203.0.113.0/24,198.51.100.7` | только перечисленные CIDR/IP |
| `none` | заголовки игнорируются, используется TCP-пир |

Если перед ingress стоит внешний балансировщик/CDN, добавьте его диапазоны в `TRUSTED_PROXIES`.

`admintoken`: принимается только `Authorization: Bearer`, сравнение constant-time.
`WithQueryToken()` оставлен для совместимости и ничего не делает (токены в URL попадают в логи).

## Разработка

```bash
go test ./...
```

Требуется Go >= 1.23.

Локально в соседнем checkout сервиса:

```
replace github.com/cotherapist-ru/go-kit => ../packages/go-kit
```

После публикации тега `replace` не нужен.

## Публикация

См. [PUBLISHING.md](PUBLISHING.md).

## Лицензия

MIT — см. [LICENSE](LICENSE).
