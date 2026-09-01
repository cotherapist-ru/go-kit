# Публикация go-kit

Модуль публикуется тегом Git с GitHub. `proxy.golang.org` подхватывает публичный репозиторий.

| Что | Где |
|-----|-----|
| GitHub-репозиторий | https://github.com/cotherapist-ru/go-kit |
| Module path | `github.com/cotherapist-ru/go-kit` |
| CI | `.github/workflows/ci.yml` |

Публикация для Go = git-тег `vX.Y.Z`. Отдельного npm workflow нет.

## 1. Создать репозиторий

1. https://github.com/organizations/cotherapist-ru/repositories/new
2. **Repository name:** `go-kit`
3. **Visibility:** Public
4. Без README / .gitignore / license — они уже в пакете

```bash
cd packages/go-kit
git init
git add .
git commit -m "feat: initial release of go-kit"
git branch -M main
git remote add origin git@github.com:cotherapist-ru/go-kit.git
git push -u origin main
```

## 2. Проверка CI

После push в `main`: Actions → workflow **CI** (`go test ./...` на Go 1.23 и 1.25).

Локально:

```bash
go test ./...
```

## 3. Первый релиз (v0.1.0)

Тег **обязан** иметь префикс `v` (Go modules).

```bash
git tag v0.1.0
git push origin v0.1.0
```

Либо GitHub Release: tag `v0.1.0` на `main`.

После индексации proxy:

```bash
go get github.com/cotherapist-ru/go-kit@v0.1.0
```

`GOPRIVATE` не нужен для публичного репозитория.

## 4. Потребители

В `go.mod` сервиса:

```
require github.com/cotherapist-ru/go-kit v0.1.0
```

Без `replace`. Для разработки до первого тега:

```
replace github.com/cotherapist-ru/go-kit => ../packages/go-kit
```

## 5. Следующие релизы

1. Изменения в `packages/go-kit`
2. `go test ./...`
3. Conventional commit, push в `main`
4. Тег `v0.1.1` / `v0.2.0` и push тега
5. В сервисах: `go get github.com/cotherapist-ru/go-kit@v0.1.1`

## Checklist первого релиза

- [ ] Репозиторий `cotherapist-ru/go-kit` на GitHub создан
- [ ] `main` запушен
- [ ] CI зелёный
- [ ] Тег `v0.1.0` запушен
- [ ] `go get github.com/cotherapist-ru/go-kit@v0.1.0` резолвится
- [ ] `replace` убран из потребителей
