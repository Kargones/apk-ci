package designer

import (
	"context"
	"log/slog"
	"path"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// Load загружает конфигурацию из файлов в информационную базу.
// Может загружать как основную конфигурацию, так и расширения при указании имени расширения.
// Параметры:
//   - ctx: контекст выполнения операции (не используется)
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - sourcePath: путь к исходным файлам конфигурации
//   - extensionName: опциональное имя расширения для загрузки
//
// Возвращает:
//   - error: ошибка загрузки конфигурации, nil при успехе
func (odb *OneDb) Load(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, extensionName ...string) error {
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, odb.FullConnectString)
	r.Params = append(r.Params, "/LoadConfigFromFiles")
	r.Params = append(r.Params, sourcePath)

	// Если указано имя расширения, добавляем параметры для работы с расширением
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, extensionName[0])
	}

	// r.Params = append(r.Params, "/UpdateDBCfg")

	addDisableParam(&r)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем соответствующее сообщение в зависимости от типа загрузки
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "/c Загрузка из файлов расширения")
	} else {
		r.Params = append(r.Params, "/c Загрузка из файлов основной конфигурации")
	}

	_, err := r.RunCommand(ctx, l)
	if err != nil {
		if len(extensionName) > 0 && extensionName[0] != "" {
			l.Error("Ошибка загрузки файлов расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Источник файлов", sourcePath),
				slog.String("Error", err.Error()),
			)
		} else {
			l.Error("Ошибка загрузки файлов конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Источник файлов", sourcePath),
				slog.String("Error", err.Error()),
			)
		}
		return err
	}

	// Проверяем результат выполнения
	if len(extensionName) > 0 && extensionName[0] != "" {
		if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseLoadOk) &&
			!strings.Contains(string(r.FileOut), constants.SearchMsgEmptyFile) &&
			!strings.Contains(string(r.FileOut), constants.InvalidLink) {
			l.Error("Неопознанная ошибка загрузки файлов расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Источник файлов", sourcePath),
			)
			// ToDo: дополнить игнорирование предупреждений
			// return err
		}
	} else {
		if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseLoadOk) &&
			!strings.Contains(string(r.FileOut), constants.SearchMsgEmptyFile) &&
			!strings.Contains(string(r.FileOut), constants.InvalidLink) {
			l.Error("Неопознанная ошибка загрузки файлов конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("FileOut", string(r.FileOut)),
			)
			// ToDo: дополнить игнорирование предупреждений
			// return err
		}
	}
	return err
}

// LoadAdd загружает расширение в информационную базу.
// Обертка для функции Load с указанием имени расширения для обратной совместимости.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - sourcePath: путь к исходным файлам расширения
//   - name: имя расширения
//
// Возвращает:
//   - error: ошибка загрузки расширения, nil при успехе
func (odb *OneDb) LoadAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, name string) error {
	return odb.Load(ctx, l, cfg, sourcePath, name)
}

// UpdateCfg обновляет конфигурацию информационной базы.
// Применяет изменения конфигурации к базе данных, может работать с расширениями.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - sourcePath: путь к источнику обновлений
//   - extensionName: опциональное имя расширения для обновления
//
// Возвращает:
//   - error: ошибка обновления конфигурации, nil при успехе
func (odb *OneDb) UpdateCfg(ctx context.Context, l *slog.Logger, cfg *config.Config, _ string, extensionName ...string) error {
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, odb.FullConnectString)
	r.Params = append(r.Params, "/UpdateDBCfg")

	// Если указано имя расширения, добавляем параметры для работы с расширением
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, extensionName[0])
	}

	addDisableParam(&r)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем соответствующее сообщение в зависимости от типа обновления
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "/c Обновление конфигурации расширения")
	} else {
		r.Params = append(r.Params, "/c Обновление основной конфигурации")
	}

	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка обновления конфигурации расширения",
			slog.String("Путь", odb.DbConnectString),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseLoadOk) && !strings.Contains(string(r.FileOut), constants.SearchMsgEmptyFile) {
		l.Error("Неопознанная ошибка обновления конфигурации расширения",
			slog.String("Путь", odb.DbConnectString),
			slog.String("Вывод файла", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// UpdateAdd обертка для обратной совместимости.
// Вызывает метод UpdateCfg для обновления конфигурации расширения.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - sourcePath: путь к исходным файлам конфигурации
//   - name: имя расширения для обновления
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (odb *OneDb) UpdateAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, name string) error {
	return odb.UpdateCfg(ctx, l, cfg, sourcePath, name)
}

// Dump выгружает конфигурацию базы данных в файл.
// Если указано имя расширения, выгружает конфигурацию расширения в файл .cfe,
// иначе выгружает основную конфигурацию в файл main.cf.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - extensionName: опциональное имя расширения для выгрузки
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (odb *OneDb) Dump(ctx context.Context, l *slog.Logger, cfg *config.Config, extensionName ...string) error {
	var fileName string
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, odb.FullConnectString)
	r.Params = append(r.Params, "/DumpDBCfg")

	// Если указано имя расширения, выгружаем расширение
	if len(extensionName) > 0 && extensionName[0] != "" {
		fileName = path.Join(cfg.WorkDir, extensionName[0]+".cfe")
		r.Params = append(r.Params, fileName)
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, extensionName[0])
	} else {
		fileName = path.Join(cfg.WorkDir, "main.cf")
		r.Params = append(r.Params, fileName)
	}

	addDisableParam(&r)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем соответствующее сообщение в зависимости от типа выгрузки
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "/c Выгрузка расширения")
	} else {
		r.Params = append(r.Params, "/c Выгрузка основной конфигурации")
	}

	_, err := r.RunCommand(ctx, l)
	if err != nil {
		if len(extensionName) > 0 && extensionName[0] != "" {
			l.Error("Ошибка выгрузки расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя расширения", extensionName[0]),
				slog.String("Имя файла", fileName),
				slog.String("Error", err.Error()),
			)
		} else {
			l.Error("Ошибка выгрузки основной конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя файла", fileName),
				slog.String("Error", err.Error()),
			)
		}
		return err
	}

	if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseDumpOk) {
		if len(extensionName) > 0 && extensionName[0] != "" {
			l.Error("Неопознанная ошибка выгрузки расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя расширения", extensionName[0]),
				slog.String("Имя файла", fileName),
			)
		} else {
			l.Error("Неопознанная ошибка выгрузки основной конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя файла", fileName),
			)
		}
		return err
	}
	return err
}

// DumpAdd обертка для обратной совместимости.
// Вызывает метод Dump для выгрузки конфигурации расширения.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - name: имя расширения для выгрузки
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (odb *OneDb) DumpAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, name string) error {
	return odb.Dump(ctx, l, cfg, name)
}
