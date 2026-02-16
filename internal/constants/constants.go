// Package constants содержит все константы, используемые в проекте apk-ci.
// Константы сгруппированы по их функциональному назначению для удобства использования и поддержки.
package constants

// Константы сообщений приложения
const (
	// MsgAppExit - сообщение о завершении работы программы
	MsgAppExit = "Завершение работы програмы"
	// MsgErrProcessing - сообщение об обработке ошибки
	MsgErrProcessing = "Обработка ошибки"
	// MsgSource - сообщение об исходном объекте
	MsgSource = "Исходный"
	// MsgDistination - сообщение о конечном объекте
	MsgDistination = "Конечный"
	// ПроверкаПеременных - сообщение о проверке переменных
	ПроверкаПеременных = "Проверка переменных"
	// ПеременныеСреды - сообщение о переменных среды
	ПеременныеСреды = "Переменные среды"
)

// Константы версии приложения теперь находятся в автоматически генерируемом файле version.go
// Этот файл создается при каждой сборке с помощью скрипта generate-version.sh

// Константы веток Git
const (
	// EdtBranch - основная ветка для EDT
	EdtBranch = "main"
	// OneCBranch - ветка для 1C конфигураций
	OneCBranch = "xml"
	// BaseBranch - базовая ветка
	BaseBranch = "main"
	// TestBranch - ветка для тестирования
	TestBranch = "testMerge"
)

// Константы корневых путей
const (
	// StoreRoot - корневой путь хранилища
	// ToDo: перенести значение в конфигурацию
	StoreRoot = "tcp://prod-1c-repo.apkholding.ru/gitops/"
	// GiteaWorkflowsPath - путь к рабочим процессам Gitea
	GiteaWorkflowsPath = ".gitea/workflows"
)

// LocalBase - локальное расположение базы сборки.
const LocalBase = "local"

// Константы действий (команд)
const (
	// ActConvert - действие конвертации
	ActConvert = "convert"
	// // ActConvertxml - действие конвертации в XML
	// ActConvertxml = "convertxml"
	// // ActConvertedt - действие конвертации в EDT
	// ActConvertedt = "convertedt"
	// // ActStore2git - действие переноса из хранилища в Git
	// ActStore2git = "store2git"
	// ActGit2store - действие переноса из Git в хранилище
	ActGit2store = "git2store"
	// ActDbrestore - действие восстановления базы данных
	ActDbrestore = "dbrestore"
	// ActIssuetask - действие создания задачи
	ActIssuetask = "issuetask"
	// ActServiceModeEnable - действие включения сервисного режима
	ActServiceModeEnable = "service-mode-enable"
	// ActServiceModeDisable - действие отключения сервисного режима
	ActServiceModeDisable = "service-mode-disable"
	// ActServiceModeStatus - действие получения статуса сервисного режима
	ActServiceModeStatus = "service-mode-status"
	// // ActServiceModeEnableDb - действие включения сервисного режима для БД
	// ActServiceModeEnableDb = "service-mode-enable-db"
	// // ActServiceModeDisableDb - действие отключения сервисного режима для БД
	// ActServiceModeDisableDb = "service-mode-disable-db"
	// ActStore2db - действие переноса из хранилища в БД
	ActStore2db = "store2db"
	// ActStoreBind - действие привязки хранилища к базе данных
	ActStoreBind = "storebind"
	// ActDbupdate - действие обновления базы данных
	ActDbupdate = "dbupdate"
	// ActAnalyzeProject - действие анализа проекта
	ActAnalyzeProject = "analyze-project"
	// ActionMenuBuildName - действие построения меню действий
	ActionMenuBuildName = "action-menu-build"
	// ActCreateTempDb - действие создания временной базы данных
	ActCreateTempDb = "create-temp-db"
	// ActCreateStores - действие создания хранилищ
	ActCreateStores = "create-stores"
	// ActExecuteEpf - действие выполнения внешней обработки
	ActExecuteEpf = "execute-epf"
	// ActSQScanBranch - действие сканирования ветки SonarQube
	ActSQScanBranch = "sq-scan-branch"
	// ActSQScanPR - действие сканирования pull request SonarQube
	ActSQScanPR = "sq-scan-pr"
	// ActSQProjectUpdate - действие обновления проекта SonarQube
	ActSQProjectUpdate = "sq-project-update"
	// ActSQReportBranch - действие генерации отчета по ветке SonarQube
	ActSQReportBranch = "sq-report-branch"
	// ActTestMerge - действие проверки конфликтов слияния
	ActTestMerge = "test-merge"
	// ActExtensionPublish - действие публикации расширения 1C
	ActExtensionPublish = "extension-publish"
)

// Константы заголовков задач
const (
	// TaskDbRestore - заголовок задачи восстановления БД
	TaskDbRestore = "Восстановление базы из бекапа продуктивного контура [TEST]"
	// TaskStore2DbTest - заголовок задачи загрузки конфигурации в тестовую БД
	TaskStore2DbTest = "Загрузка конфигурации из хранилища [TEST]"
	// TaskUpdateDbTest - заголовок задачи обновления тестовой БД
	TaskUpdateDbTest = "Применение обновленной конфигурации [TEST]"
	// TaskLoadAndUpdateDbTest - заголовок задачи загрузки и обновления тестовой БД
	TaskLoadAndUpdateDbTest = "Загрузка конфигурации из хранилища и ее применение [TEST]"
	// TaskStore2DbProd - заголовок задачи загрузки конфигурации в продуктивную БД
	TaskStore2DbProd = "Загрузка конфигурации из хранилища [PROD]"
	// TaskUpdateDbProd - заголовок задачи обновления продуктивной БД
	TaskUpdateDbProd = "Применение обновленной конфигурации [PROD]"
	// TaskLoadAndUpdateDbProd - заголовок задачи загрузки и обновления продуктивной БД
	TaskLoadAndUpdateDbProd = "Загрузка конфигурации из хранилища и ее применение [PROD]"
)

// Константы API и групп
const (


	// APIVersion - версия API
	APIVersion = "v1"
	// GroupName - имя группы в Gitea
	GroupName = "qa"
	// GitOpsSystemUser - системный пользователь GitOps
	GitOpsSystemUser = "gitops"
	// DebugUser - пользователь для отладки
	DebugUser = "xor"
)

// Константы для автора коммитов Gitea
const (
	// DefaultCommitAuthorName - имя автора коммитов по умолчанию
	DefaultCommitAuthorName = "GitOps Bot"
	// DefaultCommitAuthorEmail - email автора коммитов по умолчанию
	DefaultCommitAuthorEmail = "gitops@apkholding.ru"
)

// Константы уровней логирования
const (
	// LogLevelDebug - уровень отладки
	LogLevelDebug = "Debug"
	// LogLevelInfo - информационный уровень
	LogLevelInfo = "Info"
	// LogLevelWarn - уровень предупреждений
	LogLevelWarn = "Warn"
	// LogLevelError - уровень ошибок
	LogLevelError = "Error"
	// LogLevelDefault - уровень по умолчанию
	LogLevelDefault = LogLevelInfo
)

// Константы для работы с конвертацией
const (
	// MergeSettingsString - строка настроек слияния конфигураций
	MergeSettingsString = `<?xml version="1.0" encoding="UTF-8"?>
<Settings xmlns="http://v8.1c.ru/8.3/config/merge/settings" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.2" platformVersion="8.3.11">
	<Parameters>
		<AllowMainConfigurationObjectDeletion>true</AllowMainConfigurationObjectDeletion>
	</Parameters>
</Settings>`
	// DefaultUser - пользователь по умолчанию для конвертации
	DefaultUser = "gitops"
	// DefaultPass - пароль по умолчанию для конвертации
	DefaultPass = "gitops"
)

// Константы сообщений хранилища
const (
	// SearchMsgStoreLockOk - сообщение об успешном захвате объектов в хранилище
	SearchMsgStoreLockOk = "Захват объектов в хранилище успешно завершен"
	// SearchMsgStoreMergeOk - сообщение об успешном объединении конфигураций
	SearchMsgStoreMergeOk = "Объединение конфигураций успешно завершено"
	// SearchMsgStoreBindOk - сообщение об успешном подключении к хранилищу
	SearchMsgStoreBindOk = "Подключение информационной базы к хранилищу успешно завершено"
	// SearchMsgStoreUnBindOk - сообщение об успешном отключении от хранилища
	SearchMsgStoreUnBindOk = "Отключение от хранилища конфигурации успешно завершено"
	// SearchMsgStoreCommitOk - сообщение об успешном помещении изменений в хранилище
	SearchMsgStoreCommitOk = "Помещение изменений объектов в хранилище успешно завершено"
	// SearchMsgStoreCreateOk - сообщение об успешном создании хранилища
	SearchMsgStoreCreateOk = "Создание хранилища конфигурации успешно завершено"
	// SearchMsgStoreUpdateCfgOk - сообщение об успешном обновлении конфигурации из хранилища
	SearchMsgStoreUpdateCfgOk = "Обновление конфигурации из хранилища успешно завершено"
)

// Константы сообщений дизайнера
const (
	// SearchMsgBaseCreateOk - сообщение об успешном создании информационной базы (русский вариант)
	SearchMsgBaseCreateOk = "Создание информационной базы успешно завершено"
	// SearchMsgBaseCreateOkEn - сообщение об успешном создании информационной базы (английский вариант для ibcmd)
	SearchMsgBaseCreateOkEn = "Infobase created"
	// SearchMsgBaseAddOk - сообщение об успешном обновлении конфигурации БД
	SearchMsgBaseAddOk = "Обновление конфигурации базы данных успешно завершено"
	// SearchMsgBaseLoadOk - сообщение об успешном обновлении конфигурации
	SearchMsgBaseLoadOk = "Обновление конфигурации успешно завершено"
	// SearchMsgBaseDumpOk - сообщение об успешном сохранении конфигурации
	SearchMsgBaseDumpOk = "Сохранение конфигурации успешно завершено"
	// SearchMsgEmptyFile - маркер пустого файла
	SearchMsgEmptyFile = "\ufeff"
	// InvalidLink - сообщение об ошибке ссылки
	InvalidLink = "неверная ссылка"
)

// Константы сервисного режима
const (
	// DefaultServiceModeMessage - сообщение сервисного режима по умолчанию
	DefaultServiceModeMessage = "Система находится в режиме обслуживания"
)

// Константы Git
const (
	// LastCommit - константа для обозначения последнего коммита
	LastCommit = "last"
)

const (
	// WorkDir - рабочая директория.
	WorkDir = "/tmp/4del"
	// TempDir - временная директория.
	TempDir = "/tmp/4del/temp"
)
