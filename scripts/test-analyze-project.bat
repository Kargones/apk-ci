@echo off
echo Тестирование команды analyze-project...

REM Устанавливаем переменные окружения
set BR_COMMAND=analyze-project
set BR_REPO_URL=https://gitea.example.com/owner/repo
set BR_ACCESS_TOKEN=test_token

REM Запускаем команду
benadis-runner.exe

echo Тест завершен.
pause