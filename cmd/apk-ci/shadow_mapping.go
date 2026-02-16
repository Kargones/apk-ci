package main

import (
	"context"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/app"
	"github.com/Kargones/apk-ci/internal/command/shadowrun"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// buildLegacyMapping создаёт маппинг NR-команд на legacy-функции из internal/app/.
// Для функций с дополнительными параметрами создаются обёртки с унифицированной сигнатурой.
// Команды без legacy-аналога (nr-version, nr-help, nr-force-disconnect-sessions)
// не регистрируются — shadow-run выведет warning (AC8).
func buildLegacyMapping() *shadowrun.LegacyMapping {
	m := shadowrun.NewLegacyMapping()

	// Команды с сигнатурой (ctx, l, cfg) error — прямой маппинг
	m.Register(constants.ActNRStore2db, app.Store2DbWithConfig)
	m.Register(constants.ActNRConvert, app.Convert)
	m.Register(constants.ActNRGit2store, app.Git2Store)
	m.Register(constants.ActNRDbupdate, app.DbUpdateWithConfig)
	m.Register(constants.ActNRExecuteEpf, app.ExecuteEpf)
	m.Register(constants.ActNRStorebind, app.StoreBind)
	m.Register(constants.ActNRCreateStores, app.CreateStoresWrapper)
	m.Register(constants.ActNRSQProjectUpdate, app.SQProjectUpdate)
	m.Register(constants.ActNRSQReportBranch, app.SQReportBranch)
	m.Register(constants.ActNRSQScanPR, app.SQScanPR)
	m.Register(constants.ActNRTestMerge, app.TestMerge)
	m.Register(constants.ActNRActionMenuBuild, app.ActionMenuBuildWrapper)

	// Команды с дополнительными параметрами — обёртки
	m.Register(constants.ActNRServiceModeStatus,
		func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
			return app.ServiceModeStatus(ctx, l, cfg, cfg.InfobaseName)
		})

	m.Register(constants.ActNRServiceModeEnable,
		func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
			return app.ServiceModeEnable(ctx, l, cfg, cfg.InfobaseName, cfg.TerminateSessions)
		})

	m.Register(constants.ActNRServiceModeDisable,
		func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
			return app.ServiceModeDisable(ctx, l, cfg, cfg.InfobaseName)
		})

	m.Register(constants.ActNRDbrestore,
		func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
			return app.DbRestoreWithConfig(ctx, l, cfg, cfg.InfobaseName)
		})

	m.Register(constants.ActNRCreateTempDb,
		func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
			_, err := app.CreateTempDbWrapper(ctx, l, cfg)
			return err
		})

	m.Register(constants.ActNRSQScanBranch,
		func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
			// Legacy SQScanBranch принимает commitHash как параметр, NR handler получает коммиты из Gitea API.
			// Передаём пустой commitHash — legacy сканирует все коммиты ветки.
			// Ограничение: результаты сравнения могут отличаться из-за разных механизмов выбора коммитов.
			return app.SQScanBranch(ctx, l, cfg, "")
		})

	// Команды БЕЗ legacy-аналога (AC8 — shadow-run выведет warning):
	// 5 команд не зарегистрированы: nr-version, help, nr-force-disconnect-sessions,
	// nr-migrate, nr-deprecated-audit. Всего зарегистрировано 18 из 23 NR-команд.

	// Команды, изменяющие состояние: shadow-run выведет предупреждение,
	// т.к. legacy-версия может вызвать побочные эффекты повторно.
	m.MarkStateChanging(
		constants.ActNRServiceModeEnable,
		constants.ActNRServiceModeDisable,
		constants.ActNRDbrestore,
		constants.ActNRDbupdate,
		constants.ActNRCreateTempDb,
		constants.ActNRCreateStores,
		constants.ActNRGit2store,
		constants.ActNRStore2db,
		constants.ActNRStorebind,
		constants.ActNRExecuteEpf,
	)

	return m
}
