// Package main содержит точку входа для приложения benadis-runner.
// Приложение предназначено для автоматизации процессов разработки и развертывания.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Kargones/apk-ci/internal/app"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

func main() {
	var err error
	ctx := context.Background()
	cfg, err := config.MustLoad()
	if err != nil || cfg == nil {
		fmt.Fprintf(os.Stderr, "Не удалось загрузить конфигурацию приложения: %v\n", err)
		os.Exit(5)
	}
	l := cfg.Logger
	// Логирование информации о версии и коммите на уровне Debug
	l.Debug("Информация о сборке",
		slog.String("version", constants.Version),
		slog.String("commit_hash", constants.PreCommitHash),
	)
	switch cfg.Command {
	case constants.ActStore2db:
		err = app.Store2DbWithConfig(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка обновления хранилища",
				slog.String("Сообщение об ошибке", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(7)
		}
	case constants.ActConvert:
		err = app.Convert(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка конвертации",
				slog.String("Сообщение об ошибке", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(6)
		}
		l.Info("Конвертация успешно завершена")
	case constants.ActGit2store:
		err = app.Git2Store(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка обновления хранилища",
				slog.String("Сообщение об ошибке", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(7)
		}
		l.Info("Обновление хранилища успешно завершено")

	case constants.ActServiceModeEnable:
		// Получаем имя информационной базы из переменной окружения
		infobaseName := cfg.InfobaseName
		if infobaseName == "" {
			l.Error("Не указано имя информационной базы",
				slog.String("env_var", "BR_INFOBASE_NAME"),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		terminateSessions := cfg.TerminateSessions
		err = app.ServiceModeEnable(&ctx, l, cfg, infobaseName, terminateSessions)
		if err != nil {
			l.Error("Ошибка включения сервисного режима",
				slog.String("infobase", infobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Сервисный режим успешно включен", "infobase", infobaseName)
	case constants.ActServiceModeDisable:
		// Получаем имя информационной базы из конфигурации
		infobaseName := cfg.InfobaseName
		if infobaseName == "" {
			l.Error("Не указано имя информационной базы",
				slog.String("env_var", "BR_INFOBASE_NAME"),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		err = app.ServiceModeDisable(&ctx, l, cfg, infobaseName)
		if err != nil {
			l.Error("Ошибка отключения сервисного режима",
				slog.String("infobase", infobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Сервисный режим успешно отключен", "infobase", infobaseName)
	case constants.ActServiceModeStatus:
		// Получаем имя информационной базы из конфигурации
		infobaseName := cfg.InfobaseName
		if infobaseName == "" {
			l.Error("Не указано имя информационной базы",
				slog.String("env_var", "BR_INFOBASE_NAME"),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		err = app.ServiceModeStatus(&ctx, l, cfg, infobaseName)
		if err != nil {
			l.Error("Ошибка получения статуса сервисного режима",
				slog.String("infobase", infobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Статус сервисного режима", "infobase", infobaseName, "status", err)
	case constants.ActDbupdate:
		err = app.DbUpdateWithConfig(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка выполнения DbUpdate",
				slog.String("dbName", cfg.InfobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("DbUpdate успешно выполнен", "dbName", cfg.InfobaseName)
	case constants.ActDbrestore:
		err = app.DbRestoreWithConfig(&ctx, l, cfg, cfg.InfobaseName)
		if err != nil {
			l.Error("Ошибка выполнения DbRestore",
				slog.String("dbName", cfg.InfobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("DbRestore успешно выполнен", "dbName", cfg.InfobaseName)
	case constants.ActionMenuBuildName:
		err = app.ActionMenuBuildWrapper(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка выполнения ActionMenuBuild",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("ActionMenuBuild успешно выполнен")
	case constants.ActStoreBind:
		err = app.StoreBind(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка выполнения StoreBind",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("StoreBind успешно выполнен")
	case constants.ActCreateTempDb:
		connectString, errCreateDB := app.CreateTempDbWrapper(&ctx, l, cfg)
		if errCreateDB != nil {
			l.Error("Ошибка создания временной базы данных",
				slog.String("error", errCreateDB.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Временная база данных успешно создана", "connectString", connectString)
	case constants.ActCreateStores:
		err = app.CreateStoresWrapper(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка создания хранилищ конфигурации",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Хранилища конфигурации успешно созданы")
	case constants.ActExecuteEpf:
		err = app.ExecuteEpf(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка выполнения внешней обработки",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Внешняя обработка успешно выполнена")
	case constants.ActSQScanBranch:
		err = app.SQScanBranch(&ctx, l, cfg, cfg.CommitHash) // Пустой commitHash означает последний коммит
		if err != nil {
			l.Error("Ошибка сканирования ветки SonarQube",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Сканирование ветки SonarQube успешно завершено")
	case constants.ActSQScanPR:
		err = app.SQScanPR(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка сканирования pull request SonarQube",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Сканирование pull request SonarQube успешно завершено")
	case constants.ActSQProjectUpdate:
		err = app.SQProjectUpdate(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка обновления проекта SonarQube",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Обновление проекта SonarQube успешно завершено")
	case constants.ActSQReportBranch:
		err = app.SQReportBranch(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка генерации отчета по ветке SonarQube",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Генерация отчета по ветке SonarQube успешно завершена")
	case constants.ActTestMerge:
		err = app.TestMerge(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка проверки конфликтов слияния",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Проверка конфликтов слияния успешно завершена")
	case constants.ActExtensionPublish:
		err = app.ExtensionPublish(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка публикации расширения",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			os.Exit(8)
		}
		l.Info("Публикация расширения успешно завершена")
	default:
		l.Error("неизвестная команда",
			slog.String("BR_ACTION", cfg.Command),
			slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
		)
		os.Exit(2)
	}
}
