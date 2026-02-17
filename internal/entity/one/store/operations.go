package store

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// Lock блокирует хранилище конфигурации для эксклюзивного доступа.
// Функция захватывает объекты основной конфигурации в хранилище,
// предотвращая их изменение другими пользователями.
// Параметры:
//   - _: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) Lock(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryLock")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Захват объектов основной конфигурации")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка захвата основной конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreLockOk) {
		l.Error("Неопознанная ошибка захвата основной конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// LockAdd блокирует объекты в хранилище с дополнительным именем.
// Функция захватывает объекты указанного расширения в хранилище конфигурации.
// Параметры:
//   - ctx: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) LockAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, addName string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryLock")
	r.Params = append(r.Params, "-Extension")
	r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Захват объектов расширения")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка захвата объектов расширения",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreLockOk) {
		l.Error("Неопознанная ошибка захвата расширения",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// Create создает новое хранилище для основной конфигурации.
// Функция создает хранилище конфигурации с настройками по умолчанию,
// разрешающими изменения объектов с поддержкой редактирования.
// Параметры:
//   - _: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - isMain: флаг основной конфигурации
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) Create(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, _ bool) error {
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryCreate")
	r.Params = append(r.Params, "-AllowConfigurationChanges")
	r.Params = append(r.Params, "-ChangesAllowedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "-ChangesNotRecommendedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Создание хранилища конфигурации")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка создания хранилища конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreCreateOk) {
		l.Error("Неопознанная ошибка создания хранилища конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}
	return err
}

// CreateAdd создает новое хранилище конфигурации для расширения.
// Функция создает хранилище для указанного расширения конфигурации
// с настройками, разрешающими изменения объектов.
// Параметры:
//   - _: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) CreateAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, addName string) error {
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryCreate")
	r.Params = append(r.Params, "-Extension")
	r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "-AllowConfigurationChanges")
	r.Params = append(r.Params, "-ChangesAllowedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "-ChangesNotRecommendedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Создание хранилища конфигурации")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка создания хранилища конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreCreateOk) {
		l.Error("Неопознанная ошибка создания хранилища конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}
	return err
}

// Check проверяет существование хранилища конфигурации и создает его при необходимости.
// Функция проверяет наличие файла хранилища или доступность TCP-соединения,
// и создает новое хранилище, если оно не существует.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - isMain: флаг основной конфигурации
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) Check(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, isMain bool) error {
	// ToDo: заменить проверку на получение списка пользователей
	// ToDo: Добавить проверку существования хранилища конфигурации для tcp://
	storeExists := false
	if s.Path[0:6] != TCPProtocol {
		if exists(path.Join(s.Path, "1cv8ddb.1CD")) {
			storeExists = true
		}
	} else {
		// // Проверка наличия хранилища конфигурации для пути с TCPProtocol
		// _, _, _, err := s.ReadReport(l, dbConnectString, cfg, -1)
		// if err == nil {
		storeExists = true
		// }
	}
	if !storeExists {
		if err := s.Create(context.Background(), l, cfg, dbConnectString, isMain); err != nil {
			l.Error("Ошибка создания хранилища конфигурации",
				slog.String("Путь", s.Path),
				slog.String("Строка подключения", dbConnectString),
				slog.String("Error", err.Error()),
			)
			return fmt.Errorf("ошибка создания хранилища конфигурации")
		}
	}

	return nil
}

// CheckAdd проверяет существование хранилища конфигурации для расширения и создает его при необходимости.
// Функция проверяет наличие файла хранилища для указанного расширения
// и создает новое хранилище, если оно не существует.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) CheckAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, addName string) error {
	if s.Path[0:6] != TCPProtocol {
		if !exists(path.Join(s.Path, "1cv8ddb.1CD")) {
			if err := s.CreateAdd(ctx, l, cfg, dbConnectString, addName); err != nil {
				l.Error("Ошибка создания хранилища конфигурации",
					slog.String("Путь", s.Path),
					slog.String("Строка подключения", dbConnectString),
					slog.String("Error", err.Error()),
				)
				return fmt.Errorf("ошибка создания хранилища конфигурации")
			}
		}
	}
	return nil
}

// Bind подключает хранилище конфигурации к базе данных.
// Функция привязывает основную конфигурацию к хранилищу с принудительной
// заменой существующей конфигурации и привязкой уже подключенных пользователей.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//   - isMain: флаг основной конфигурации
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) Bind(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, isMain bool) error {
	var tPath string
	s.Name = s.Path
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	err := s.Check(ctx, l, cfg, dbConnectString, isMain)
	if err != nil {
		l.Error("Невозможно подключиться к хранилищу конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("не удалось подключиться к хранилищу конфигурации")
	}
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryBindCfg")
	r.Params = append(r.Params, "-forceBindAlreadyBindedUser")
	r.Params = append(r.Params, "-forceReplaceCfg")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Подключение основной конфигурации к хранилищу")
	_, err = r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка подключения основной конфигурации к хранилищу",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreBindOk) {
		l.Error("Неопознанная ошибка подключения основной конфигурации к хранилищу",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}

	// Обновление конфигурации из хранилища
	r = s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryUpdateCfg")
	r.Params = append(r.Params, "-revised")
	r.Params = append(r.Params, "-force")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Обновление конфигурации из хранилища")
	_, err = r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка обновления конфигурации из хранилища",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreUpdateCfgOk) {
		l.Error("Неопознанная ошибка обновления конфигурации из хранилища",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}

	return err
}

// BindAdd подключает хранилище конфигурации расширения к базе данных.
// Функция привязывает указанное расширение к хранилищу с принудительной
// заменой существующей конфигурации и привязкой уже подключенных пользователей.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) BindAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, addName string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	err := s.CheckAdd(context.Background(), l, cfg, dbConnectString, addName)
	if err != nil {
		l.Error("Невозможно подключиться к хранилищу конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("не удалось подключиться к хранилищу конфигурации")
	}

	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryBindCfg")
	r.Params = append(r.Params, "-Extension")
	r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "-forceBindAlreadyBindedUser")
	r.Params = append(r.Params, "-forceReplaceCfg")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Подключение расширения к хранилищу")
	_, err = r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка подключения расширения к хранилищу",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreBindOk) {
		l.Error("Неопознанная ошибка подключения расширения к хранилищу",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}

	// Обновление конфигурации расширения из хранилища
	r = s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryUpdateCfg")
	r.Params = append(r.Params, "-Extension")
	r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "-revised")
	r.Params = append(r.Params, "-force")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Обновление расширения из хранилища")
	_, err = r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка обновления расширения из хранилища",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreUpdateCfgOk) {
		l.Error("Неопознанная ошибка обновления расширения из хранилища",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}

	return err
}

// UnBind отвязывает хранилище конфигурации от базы данных.
// Функция принудительно отключает основную конфигурацию от хранилища.
// Параметры:
//   - ctx: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) UnBind(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryUnbindCfg")
	r.Params = append(r.Params, "-force")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Отключение конфигурации от хранилища")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка отключения конфигурации от хранилища",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreUnBindOk) {
		l.Error("Неопознанная ошибка отключения конфигурации от хранилища",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// UnBindAdd отвязывает хранилище конфигурации с дополнительным именем от базы данных.
// Функция принудительно отключает указанное расширение от хранилища.
// Параметры:
//   - ctx: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) UnBindAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, addName string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryUnbindCfg")
	r.Params = append(r.Params, "-Extension")
	r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "-force")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Отключение конфигурации от хранилища")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка отключения расширения от хранилища",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreUnBindOk) {
		l.Error("Неопознанная ошибка отключения расширения от хранилища",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// Merge выполняет слияние конфигураций в хранилище.
// Функция принудительно загружает изменения в основную конфигурацию
// с использованием указанных настроек слияния и обновляет конфигурацию базы данных.
// Параметры:
//   - ctx: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - pathCf: путь к файлу конфигурации
//   - pathMergeSettings: путь к настройкам слияния
//   - storeRoot: корневой путь к хранилищу
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) Merge(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, pathCf string, pathMergeSettings string, storeRoot string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()

	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/MergeCfg")
	r.Params = append(r.Params, pathCf)
	r.Params = append(r.Params, "-Settings")
	r.Params = append(r.Params, pathMergeSettings)
	r.Params = append(r.Params, "-force")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/UpdateDBCfg")
	r.Params = append(r.Params, "/c Загрузка изменений в основную конфигурацию")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка загрузки изменений основной конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreMergeOk) {
		l.Error("Неопознанная ошибка загрузки изменений основной конфигурации",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// MergeAdd выполняет слияние конфигураций с дополнительным именем.
// Функция принудительно загружает изменения в указанное расширение
// с использованием указанных настроек слияния и обновляет конфигурацию базы данных.
// Параметры:
//   - ctx: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - pathCf: путь к файлу конфигурации
//   - pathMergeSettings: путь к настройкам слияния
//   - storeRoot: корневой путь к хранилищу
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) MergeAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, pathCf string, pathMergeSettings string, storeRoot string, addName string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/MergeCfg")
	r.Params = append(r.Params, pathCf)
	r.Params = append(r.Params, "-Settings")
	r.Params = append(r.Params, pathMergeSettings)
	r.Params = append(r.Params, "-Extension")
	r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "-force")
	r.Params = append(r.Params, "/UpdateDBCfg")
	//	Так не работает
	// r.Params = append(r.Params, "-Extension")
	// r.Params = append(r.Params, addName)
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Загрузка изменений в расширение")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка загрузки изменений  в расширение",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreMergeOk) {
		l.Error("Неопознанная ошибка загрузки изменений в расширение",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// StoreCommit создает версию хранилища.
// Функция принудительно создает новую версию в хранилище конфигурации.
// Если указано имя расширения, создает версию для расширения.
// Параметры:
//   - ctx: контекст выполнения (не используется)
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//   - comment: комментарий к версии
//   - addName: опциональное имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) StoreCommit(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, comment string, addName ...string) error {
	var tPath string
	tPath, s.Path = fullPathStore(storeRoot, s.Path)
	defer func() {
		s.Path = tPath
	}()
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryCommit")
	CommentAdd(&r, comment)
	r.Params = append(r.Params, "-force")

	// Если указано имя расширения, добавляем параметры для работы с расширением
	if len(addName) > 0 && addName[0] != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, addName[0])
	}

	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Создание версии хранилища")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка создания версии хранилища",
			slog.String("Путь", s.Path),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreCommitOk) {
		l.Error("Неопознанная ошибка создания версии хранилища",
			slog.String("Путь", s.Path),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// StoreCommitAdd создает версию хранилища для расширения.
// Функция является оберткой для StoreCommit для обратной совместимости.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - dbConnectString: строка подключения к базе данных
//   - storeRoot: корневой путь к хранилищу
//   - comment: комментарий к версии
//   - addName: имя расширения
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func (s *Store) StoreCommitAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, comment string, addName string) error {
	return s.StoreCommit(ctx, l, cfg, dbConnectString, storeRoot, comment, addName)
}
