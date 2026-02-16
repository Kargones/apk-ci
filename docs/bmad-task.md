Следующий шаг: document-project
  Агент: analyst
  Команда: /bmad:bmm:workflows:document-project

  💡 Начните новый чат с агентом analyst первым:
  /bmad:bmm:agents:analyst

  Для проверки прогресса: /bmad:bmm:workflows:workflow-status


  Следующие шаги

  1. Ознакомьтесь с bdocs/index.md для понимания структуры документации
  2. При создании brownfield PRD укажите путь: bdocs/index.md
  3. Команда для запуска PRD workflow:
  /bmad:bmm:workflows:prd

  Рекомендации перед PR

  - Тесты: make test
  - Линтинг: make lint
  - Все проверки: make check


  /bmad:bmm:workflows:prd /home/xor/r/benadis-runner/bdocs/index.md

   Следующие шаги

  Согласно вашему workflow path (BMad Method Brownfield):

  Рекомендуемый порядок:

  1. /bmad:bmm:workflows:architecture — Техническая архитектура
    - Определить структуру модулей
    - Спроектировать DI-контейнер
    - Детализировать strategy-интерфейсы
  2. /bmad:bmm:workflows:create-epics-and-stories — Epic breakdown
    - Разбить 4 фазы на эпики
    - Создать user stories для каждого FR

  Опционально:
  - /bmad:bmm:workflows:validate-prd — Валидация PRD



  /bmad:bmm:workflows:create-story

   Для запуска выполните:
  /bmad:bmm:agents:tea

  А затем:
  /bmad:bmm:workflows:test-design