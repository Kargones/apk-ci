Шаг 1: Удали из action.yaml все intuts добавленные после строки  # Test All Actions Parameters которые используются в конструкциях вида:

        if: ${{ inputs.action_7 == true }}
Шаг 2: Удали из парсера входных параметров все параметры которые отсутствуют после выполнения первого шага.

Добавь во все шаги пераметры:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          actor: ${{ gitea.actor }}
          # Всегда передавай 2 параметра расположенные ниже
          configSystem: "app.yaml"
          configProject: "project.yaml"
          configSecret: "secret.yaml"
