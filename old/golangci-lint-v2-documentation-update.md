# Обновление документации по установке golangci-lint версии 2

## Текущая документация

В файле [docs/wiki/06-Разработка-и-тестирование.md](file:///root/r/github.com/Kargones/apk-ci/docs/wiki/06-Разработка-и-тестирование.md) в разделе "Установка инструментов разработки" есть следующий блок:

```markdown
#### Установка golangci-lint
```bash
# Используйте скрипт из проекта
./scripts/inst-golangci-lint.sh

# Или установка вручную
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
```
```

## Предлагаемое обновление

Необходимо обновить этот раздел, чтобы отразить использование golangci-lint версии 2:

```markdown
#### Установка golangci-lint
```bash
# Используйте скрипт из проекта для установки версии 2
./scripts/inst-golangci-lint-v2.sh

# Или установка вручную версии 2
go install github.com/golangci-lint/cmd/golangci-lint@v2

# Для установки конкретной версии v2.x.x
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.1.0
```
```

## Дополнительная информация

Конфигурационный файл .golangci.yml в проекте уже настроен для работы с golangci-lint версии 2, что подтверждается первой строкой файла:

```yaml
# .golangci.yml (for golangci-lint v2)
version: "2"
```

## Рекомендации

1. Рекомендуется использовать скрипт `./scripts/inst-golangci-lint-v2.sh` для установки golangci-lint версии 2
2. Убедитесь, что у вас установлена поддерживаемая версия Go (1.24.3 или выше)
3. После установки проверьте версию установленного golangci-lint командой:
   ```bash
   golangci-lint --version