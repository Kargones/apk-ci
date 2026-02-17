package convert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// ToDo: Обобщить функцию загрузки конфигурации конвертации для любого типа базы данных

// InitDb инициализирует базу данных для процесса конвертации.
// Создает новую базу данных если она не существует и добавляет в неё все источники конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка инициализации базы данных, nil при успехе
func (cc *Config) InitDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if cc.OneDB.DbExist {
		return nil
	}
	if cc.OneDB.ServerDb {
		return nil
	}
	err := cc.OneDB.Create(ctx, l, cfg)
	if err != nil {
		return err
	}
	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.Add(ctx, l, cfg, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// LoadDb загружает конфигурацию из файлов в базу данных.
// Последовательно загружает основные конфигурации и дополнительные расширения.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка загрузки конфигурации, nil при успехе
func (cc *Config) LoadDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cc.OneDB.Load(ctx, l, cfg, path.Join(cfg.RepPath, cp.Source.RelPath))
		if err != nil {
			return err
		}
	}
	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.LoadAdd(ctx, l, cfg, path.Join(cfg.RepPath, cp.Source.RelPath), cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// DumpDb выгружает конфигурацию из базы данных в файлы.
// Выполняет выгрузку основной конфигурации и всех дополнительных расширений.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка выгрузки конфигурации, nil при успехе
func (cc *Config) DumpDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	err := cc.OneDB.Dump(ctx, l, cfg)
	if err != nil {
		return err
	}
	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.DumpAdd(ctx, l, cfg, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreLock блокирует объекты в хранилище конфигурации для редактирования.
// Выполняет блокировку основных конфигураций и дополнительных расширений.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка блокировки объектов, nil при успехе
func (cc *Config) StoreLock(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.Lock(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.LockAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreBind привязывает хранилища конфигурации к базе данных.
// Устанавливает связь между файловыми хранилищами и базой данных для синхронизации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка привязки хранилищ, nil при успехе
func (cc *Config) StoreBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		l.Debug("Привязка хранилища к базе данных",
			slog.String("Имя хранилища", cp.Source.Name),
		)
		bindErr := cp.Store.Bind(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, true)
		if bindErr != nil {
			return bindErr
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		l.Debug("Привязка хранилища расширения к базе данных",
			slog.String("Имя хранилища", cp.Source.Name),
		)
		bindAddErr := cp.Store.BindAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, cp.Source.Name)
		if bindAddErr != nil {
			return bindAddErr
		}
	}
	return nil
}

// DbUpdate обновляет конфигурацию базы данных из привязанных хранилищ.
// Синхронизирует изменения из файловых хранилищ в базу данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка обновления конфигурации, nil при успехе
func (cc *Config) DbUpdate(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cc.OneDB.UpdateCfg(ctx, l, cfg, cc.OneDB.FullConnectString)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.UpdateAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreUnBind отвязывает хранилища конфигурации от базы данных.
// Разрывает связь между файловыми хранилищами и базой данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка отвязки хранилищ, nil при успехе
func (cc *Config) StoreUnBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.UnBind(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.UnBindAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreCommit фиксирует изменения в хранилище конфигурации.
// Создает новую версию в хранилище с автоматически сгенерированным комментарием.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка фиксации изменений, nil при успехе
func (cc *Config) StoreCommit(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	comment := "Версия создана автоматически\nИсходный репозиторий ххх\nДата создания ххх"
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.StoreCommit(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, comment)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.StoreCommitAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, comment, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// func (s *Store) StoreCommitAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, comment string, addName string) error {

// Merge выполняет слияние конфигурации с хранилищем.
// Объединяет изменения из рабочей директории с версией в хранилище конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка слияния конфигурации, nil при успехе
func (cc *Config) Merge(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	mergeSettingFileName, err := mergeSetting(cfg)
	if err != nil {
		l.Error("ошибка создания файла настроек слияния",
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("ошибка создания файла настроек слияния: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(mergeSettingFileName); removeErr != nil {
			l.Warn("Failed to remove merge settings file", slog.String("file", mergeSettingFileName), slog.String("error", removeErr.Error()))
		}
	}()

	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.Merge(ctx, l, cfg, cc.OneDB.FullConnectString, path.Join(cfg.WorkDir, "main.cf"), mergeSettingFileName, cc.StoreRoot)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.MergeAdd(ctx, l, cfg, cc.OneDB.FullConnectString, path.Join(cfg.WorkDir, cp.Source.Name+".cfe"), mergeSettingFileName, cc.StoreRoot, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// Save сохраняет текущую конфигурацию конвертации в указанный файл.
// Сериализует структуру Config в JSON формат и записывает в файл.
// Параметры:
//   - ctx: контекст выполнения операции (не используется)
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - configPath: путь к файлу для сохранения конфигурации
//
// Возвращает:
//   - error: ошибка сохранения файла, nil при успехе
func (cc *Config) Save(ctx context.Context, _ *slog.Logger, _ *config.Config, configPath string) error {
	ocJSON, err := json.MarshalIndent(cc, "", "\t")
	if err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}
	err = os.WriteFile(configPath, ocJSON, constants.FilePermPrivate)
	return err
}
