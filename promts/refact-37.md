Напиши функцию которая на основе константы:
```go
TemplateAction1 string = `on:
  workflow_dispatch:
    inputs:
      restore_DB:
        description: 'Восстановить базу перед загрузкой конфигурации'
        required: true
        type: boolean
        default: false 
      service_mode_enable:
        description: 'Включить сервисный режим (отключать только для загрузки конфигузации без применения)'
        required: true
        type: boolean
        default: true 
      load_cfg:
        description: 'Загрузить конфигурацию из хранилища'
        required: true
        type: boolean
        default: true 
      DbName:
        description: 'Выберите базу для загрузки конфигурации (Test)'
        required: true
        default: 'TestBaseReplace1'
        type: choice
        options:
          TestBaseReplaceAll
      update_conf:
        description: 'Применить конфигурацию после загрузки'
        required: true
        type: boolean
        default: true 
jobs:               
  db-update-test:
    runs-on: edt
    steps:
      - name: Восстановление базы данных (Test)
        id: br-dbrestore
        if: ${{ inputs.restore_DB == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbrestore"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Включение сервисного режима (Test)
        id: br-service-mode-enable
        if: ${{ inputs.service_mode_enable == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-enable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Загрузка конфигурации из хранилища (Test)
        id: br-store2db
        if: ${{ inputs.load_cfg == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "store2db"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Применение конфигурации (Test)
        id: br-dbupdate
        if: ${{ inputs.update_conf == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbupdate"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Отключение сервисного режима (Test)
        id: br-service-mode-disable
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-disable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
`
```
и массива 
```go
testDatabases := []string{
    "TestBase1",
    "TestBase2",
}
```
создаст строку:
`
on:
  workflow_dispatch:
    inputs:
      restore_DB:
        description: 'Восстановить базу перед загрузкой конфигурации'
        required: true
        type: boolean
        default: false 
      service_mode_enable:
        description: 'Включить сервисный режим (отключать только для загрузки конфигузации без применения)'
        required: true
        type: boolean
        default: true 
      load_cfg:
        description: 'Загрузить конфигурацию из хранилища'
        required: true
        type: boolean
        default: true 
      DbName:
        description: 'Выберите базу для загрузки конфигурации (Test)'
        required: true
        default: 'TestBase1'
        type: choice
        options:
          - TestBase1
          - TestBase2
        description: 'Применить конфигурацию после загрузки'
        required: true
        type: boolean
        default: true 
jobs:               
  db-update-test:
    runs-on: edt
    steps:
      - name: Восстановление базы данных (Test)
        id: br-dbrestore
        if: ${{ inputs.restore_DB == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbrestore"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Включение сервисного режима (Test)
        id: br-service-mode-enable
        if: ${{ inputs.service_mode_enable == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-enable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Загрузка конфигурации из хранилища (Test)
        id: br-store2db
        if: ${{ inputs.load_cfg == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "store2db"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Применение конфигурации (Test)
        id: br-dbupdate
        if: ${{ inputs.update_conf == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbupdate"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Отключение сервисного режима (Test)
        id: br-service-mode-disable
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-disable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

`