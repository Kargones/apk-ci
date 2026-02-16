// Package onec содержит адаптеры для работы с 1C:Предприятие.
//
// Архитектурное отклонение от Architecture.md:
// Реализации ConfigExporter и DatabaseCreator (1cv8, ibcmd) размещены в этом пакете,
// а не в подпакетах onecv8/ и ibcmd/ как указано в Architecture.md:55-58.
// Причина: избежание import cycle между factory.go и реализациями.
// Это соответствует паттерну существующих реализаций (Updater, TempDbCreator).
// ADR Exception документирована в Story 4.1 Dev Notes.
package onec

import (
	"fmt"

	"github.com/Kargones/apk-ci/internal/config"
)

// Константы для значений реализаций (goconst fix).
const (
	// Impl1cv8 — реализация через 1cv8 DESIGNER.
	Impl1cv8 = "1cv8"
	// ImplIbcmd — реализация через ibcmd.
	ImplIbcmd = "ibcmd"
	// ImplNative — нативная реализация (не реализовано).
	ImplNative = "native"
)

// Factory создаёт реализации операций на основе конфигурации.
// Реализует Strategy pattern (ADR-004) для выбора инструмента (1cv8/ibcmd/native).
//
// Использование:
//
//	factory := onec.NewFactory(cfg)
//	exporter, err := factory.NewConfigExporter()
//	creator, err := factory.NewDatabaseCreator()
type Factory struct {
	cfg *config.Config
}

// NewFactory создаёт новую фабрику операций.
func NewFactory(cfg *config.Config) *Factory {
	return &Factory{cfg: cfg}
}

// NewConfigExporter возвращает реализацию ConfigExporter на основе конфигурации.
// Выбор определяется config.AppConfig.Implementations.ConfigExport.
//
// Допустимые значения:
//   - "1cv8" (default) — использует 1cv8 DESIGNER /DumpCfg
//   - "ibcmd" — использует ibcmd infobase config export
//   - "native" — не реализовано, возвращает ошибку
func (f *Factory) NewConfigExporter() (ConfigExporter, error) {
	impl := f.cfg.AppConfig.Implementations.ConfigExport
	if impl == "" {
		impl = Impl1cv8 // default
	}

	switch impl {
	case Impl1cv8:
		return NewExporter1cv8(
			f.cfg.AppConfig.Paths.Bin1cv8,
			f.cfg.AppConfig.WorkDir,
			f.cfg.AppConfig.TmpDir,
		), nil
	case ImplIbcmd:
		return NewExporterIbcmd(
			f.cfg.AppConfig.Paths.BinIbcmd,
		), nil
	case ImplNative:
		// TODO: реализовать native exporter в будущем (Epic 4 Story 4.5 nr-convert)
		return nil, fmt.Errorf("%w: native config_export not implemented yet", ErrInvalidImplementation)
	default:
		return nil, fmt.Errorf("%w: unknown config_export implementation '%s', valid: 1cv8, ibcmd, native",
			ErrInvalidImplementation, impl)
	}
}

// NewDatabaseCreator возвращает реализацию DatabaseCreator на основе конфигурации.
// Выбор определяется config.AppConfig.Implementations.DBCreate.
//
// Допустимые значения:
//   - "1cv8" (default) — использует 1cv8 CREATEINFOBASE
//   - "ibcmd" — использует ibcmd infobase create
func (f *Factory) NewDatabaseCreator() (DatabaseCreator, error) {
	impl := f.cfg.AppConfig.Implementations.DBCreate
	if impl == "" {
		impl = Impl1cv8 // default
	}

	switch impl {
	case Impl1cv8:
		return NewCreator1cv8(
			f.cfg.AppConfig.Paths.Bin1cv8,
			f.cfg.AppConfig.WorkDir,
			f.cfg.AppConfig.TmpDir,
		), nil
	case ImplIbcmd:
		return NewCreatorIbcmd(
			f.cfg.AppConfig.Paths.BinIbcmd,
		), nil
	default:
		return nil, fmt.Errorf("%w: unknown db_create implementation '%s', valid: 1cv8, ibcmd",
			ErrInvalidImplementation, impl)
	}
}
