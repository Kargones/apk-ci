// Package store предоставляет функциональность для работы с хранилищем конфигураций 1С
package store

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// TCPProtocol определяет префикс для TCP соединений
const TCPProtocol = "tcp://"

// Константы перенесены в internal/constants/constants.go

// Store представляет хранилище конфигурации 1С.
// Содержит информацию о подключении к хранилищу конфигураций,
// включая имя, путь, учетные данные и команды для работы с хранилищем.
type Store struct {
	Name    string `json:"Имя хранилища,omitempty"`
	Path    string `json:"Относительный путь"`
	User    string `json:"Пользователь"`
	Pass    string `json:"Пароль"`
	Command string `json:"-"`
}

// Record представляет запись в хранилище конфигураций.
// Содержит информацию о версии конфигурации, пользователе,
// дате создания и комментарии к изменениям в хранилище.
// Record представляет запись в хранилище конфигураций.
// Содержит информацию о версии конфигурации, пользователе,
// дате создания и комментарии к изменениям в хранилище.
// Record представляет запись в хранилище конфигураций.
// Содержит информацию о версии конфигурации, пользователе,
// дате создания и комментарии к изменениям в хранилище.
type Record struct {
	Version     int       `json:"Версия хранилища"`
	ConfVersion string    `json:"Версия конфигурации"`
	User        string    `json:"Пользователь"`
	Date        time.Time `json:"Дата время"`
	Comment     string    `json:"Комментарий,omitempty"`
}

// User представляет пользователя системы.
// Содержит информацию о пользователе хранилища конфигураций,
// включая имя в хранилище, доменное имя и электронную почту.
type User struct {
	StoreUserName string `json:"Имя пользователя в хранилище"`
	AccountName   string `json:"Доменное имя"`
	Email         string `json:"e-mail"`
}

// ReadReport читает отчет из хранилища конфигурации.
// Функция формирует отчет о версиях конфигурации в хранилище,
// начиная с указанной версии, и возвращает записи, пользователей и максимальную версию.
// Параметры:
//   - l: логгер для записи сообщений
//   - dbConnectString: строка подключения к базе данных
//   - cfg: конфигурация приложения
//   - startVersion: начальная версия для формирования отчета
//
// Возвращает:
//   - []Record: список записей хранилища
//   - []User: список пользователей
//   - int: максимальная версия в хранилище
//   - error: ошибку, если операция не удалась
func (s *Store) ReadReport(ctx context.Context, l *slog.Logger, dbConnectString string, cfg *config.Config, startVersion int) ([]Record, []User, int, error) {
	r := s.GetStoreParam(dbConnectString, cfg)
	r.Params = append(r.Params, "/ConfigurationRepositoryReport")
	repPath := filepath.Join(r.TmpDir, s.Name+".txt")
	r.Params = append(r.Params, repPath)
	r.Params = append(r.Params, "-NBegin")
	r.Params = append(r.Params, fmt.Sprint(startVersion))
	r.Params = append(r.Params, "-ReportFormat")
	r.Params = append(r.Params, "txt")
	if s.Name != "Main" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, s.Name)
	}
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Отчет хранилища "+s.Name)
	bOut, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка выполнения команды формирования отчета", slog.String("storeName", s.Name), slog.String("error", err.Error()))
		return nil, nil, 0, fmt.Errorf("ошибка формирования отчета хранилища %s: %w", s.Name, err)
	}
	if len(bOut) > 44 && string(bOut[3:45]) != "Отчет успешно построен" {
		l.Error("Ошибка формирования отчета", slog.String("storeName", s.Name), slog.String("output", string(bOut)))
	}
	records, userList, maxVersion, err := ParseReport(repPath)
	return records, userList, maxVersion, err
}

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
func (s *Store) UnBind(_ context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string) error {
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
	_, err := r.RunCommand(context.Background(), l)
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
func fullPathStore(storeRoot, relativePath string) (string, string) {
	var fullPathStore string
	if storeRoot[0:6] != TCPProtocol {
		fullPathStore = path.Join(storeRoot, relativePath)
	} else {
		fullPathStore = storeRoot + "/" + relativePath
	}
	return relativePath, fullPathStore
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
func (s *Store) UnBindAdd(_ context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, addName string) error {
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
	_, err := r.RunCommand(context.Background(), l)
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
func (s *Store) Merge(_ context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, pathCf string, pathMergeSettings string, storeRoot string) error {
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
	_, err := r.RunCommand(context.Background(), l)
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
func (s *Store) MergeAdd(_ context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, pathCf string, pathMergeSettings string, storeRoot string, addName string) error {
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
	_, err := r.RunCommand(context.Background(), l)
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
func (s *Store) LockAdd(_ context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, addName string) error {
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
	_, err := r.RunCommand(context.Background(), l)
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

// CommentAdd добавляет комментарий к команде runner.
// Функция разбивает многострочный комментарий на отдельные строки
// и добавляет каждую строку как отдельный параметр -comment.
// Параметры:
//   - r: указатель на runner для добавления параметров
//   - comment: текст комментария (может быть многострочным)
func CommentAdd(r *runner.Runner, comment string) {
	as := strings.Split(comment, "\n")
	for _, s := range as {
		r.Params = append(r.Params, "-comment")
		r.Params = append(r.Params, "\""+s+"\"")
	}
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
func (s *Store) StoreCommit(_ context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, comment string, addName ...string) error {
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
	_, err := r.RunCommand(context.Background(), l)
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

// ParseReport парсит отчет хранилища конфигурации из файла.
// Функция читает файл отчета и извлекает информацию о версиях хранилища и пользователях.
// TODO: переделать на чтение из буфера байт.
// Параметры:
//   - path: путь к файлу отчета
//
// Возвращает:
//   - []Record: список записей версий хранилища
//   - []User: список пользователей
//   - int: максимальный номер версии
//   - error: ошибку, если операция не удалась
func ParseReport(path string) ([]Record, []User, int, error) {
	maxVersion := 0
	records := []Record{}
	userList := []User{}
	// Валидация пути к файлу
	if strings.Contains(path, "..") || path == "" {
		return nil, nil, 0, fmt.Errorf("небезопасный путь к файлу: %s", path)
	}
	// #nosec G304 - path is validated above
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}()
	sr := Record{}
	dc := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := scanner.Text()
		if t == "Комментарий:" {
			scanner.Scan()
			comment := scanner.Text()
			sr.Comment = comment
			continue
		}
		as := strings.Split(t, "\t")
		if len(as) < 2 {
			continue
		}
		switch as[0] {
		case "Версия:":
			if sr.Version != 0 {
				records = append(records, sr)
				sr = Record{}
			}
			i, parseErr := strconv.Atoi(as[1])
			if parseErr != nil {
				continue
			}
			sr.Version = i
			maxVersion = i
		case "Версия конфигурации:":
			sr.ConfVersion = as[1]
		case "Пользователь:":
			sr.User = as[1]
			exist := false
			for _, v := range userList {
				if v.StoreUserName == as[1] {
					exist = true
					break
				}
			}
			if !exist {
				cu := User{}
				cu.StoreUserName = as[1]
				userList = append(userList, cu)
			}
		case "Дата создания:":
			dc = as[1]
		case "Время создания:":
			dc += " " + as[1]
			sr.Date, err = time.Parse("02.01.2006 15:04:05", dc)
			if err != nil {
				dc = ""
				continue
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, 0, fmt.Errorf("scanner error: %w", scanErr)
	}
	return records, userList, maxVersion, err
}

// GetStoreParam возвращает параметры для работы с хранилищем конфигурации
// GetStoreParam создает и настраивает runner для работы с хранилищем конфигурации.
// Функция формирует базовые параметры подключения к хранилищу и базе данных.
// Параметры:
//   - dbConnectString: строка подключения к базе данных
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - runner.Runner: настроенный runner с базовыми параметрами
func (s *Store) GetStoreParam(dbConnectString string, cfg *config.Config) runner.Runner {
	r := runner.Runner{}
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.TmpDir = cfg.WorkDir

	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, dbConnectString)
	r.Params = append(r.Params, "/ConfigurationRepositoryF")
	r.Params = append(r.Params, s.Path)
	r.Params = append(r.Params, "/ConfigurationRepositoryN")
	r.Params = append(r.Params, s.User)
	r.Params = append(r.Params, "/ConfigurationRepositoryP")
	r.Params = append(r.Params, s.Pass)
	addDisableParam(&r)

	return r
}

// exists проверяет существование файла или директории
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// addDisableParam добавляет параметры отключения диалогов
func addDisableParam(r *runner.Runner) {
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
}

// CreateStores создает основное хранилище конфигурации и хранилища расширений
// CreateStores создает хранилища конфигурации для основной конфигурации и расширений.
// Функция создает основное хранилище и хранилища для всех указанных расширений.
// Параметры:
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - storeRoot: корневой путь к хранилищу
//   - dbConnectString: строка подключения к базе данных
//   - arrayAdd: массив имен расширений для создания хранилищ
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func CreateStores(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error {
	l.Debug("Начало создания хранилищ",
		slog.String("storeRoot", storeRoot),
		slog.String("dbConnectString", dbConnectString),
		slog.Any("arrayAdd", arrayAdd),
	)

	// Создаем основное хранилище конфигурации
	mainStore := &Store{
		Path: filepath.Join(storeRoot, "main"),
	}

	l.Debug("Создание основного хранилища конфигурации",
		slog.String("path", mainStore.Path),
	)

	// Встроенная логика создания основного хранилища (из метода Create)
	r := runner.Runner{}
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.TmpDir = cfg.WorkDir

	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, dbConnectString)
	r.Params = append(r.Params, "/ConfigurationRepositoryF")
	r.Params = append(r.Params, mainStore.Path)
	r.Params = append(r.Params, "/ConfigurationRepositoryN")
	r.Params = append(r.Params, cfg.AppConfig.Users.StoreAdmin)
	r.Params = append(r.Params, "/ConfigurationRepositoryP")
	r.Params = append(r.Params, cfg.SecretConfig.Passwords.StoreAdminPassword)
	// Добавляем параметры отключения диалогов
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
	// Параметры создания хранилища
	r.Params = append(r.Params, "/ConfigurationRepositoryCreate")
	r.Params = append(r.Params, "-AllowConfigurationChanges")
	r.Params = append(r.Params, "-ChangesAllowedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "-ChangesNotRecommendedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Создание хранилища конфигурации")

	_, err := r.RunCommand(context.Background(), l)
	if err != nil {
		l.Error("Ошибка создания хранилища конфигурации",
			slog.String("Путь", mainStore.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreCreateOk) {
		l.Error("Неопознанная ошибка создания хранилища конфигурации",
			slog.String("Путь", mainStore.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}

	l.Debug("Основное хранилище конфигурации успешно создано",
		slog.String("path", mainStore.Path),
	)

	// Создаем хранилища расширений для каждого элемента в arrayAdd
	for i, addName := range arrayAdd {
		l.Debug("Создание хранилища расширения",
			slog.Int("index", i),
			slog.String("extensionName", addName),
		)

		addStore := &Store{
			Path: filepath.Join(storeRoot, "add", addName),
		}

		l.Debug("Создание хранилища расширения",
			slog.String("path", addStore.Path),
			slog.String("extensionName", addName),
		)

		// Встроенная логика создания хранилища расширения (из метода CreateAdd)
		rAdd := runner.Runner{}
		rAdd.RunString = cfg.AppConfig.Paths.Bin1cv8
		rAdd.WorkDir = cfg.WorkDir
		rAdd.TmpDir = cfg.WorkDir

		rAdd.Params = append(rAdd.Params, "@")
		rAdd.Params = append(rAdd.Params, "DESIGNER")
		rAdd.Params = append(rAdd.Params, dbConnectString)
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryF")
		rAdd.Params = append(rAdd.Params, addStore.Path)
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryN")
		rAdd.Params = append(rAdd.Params, cfg.AppConfig.Users.StoreAdmin)
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryP")
		rAdd.Params = append(rAdd.Params, cfg.SecretConfig.Passwords.StoreAdminPassword)
		// Добавляем параметры отключения диалогов
		rAdd.Params = append(rAdd.Params, "/DisableStartupDialogs")
		rAdd.Params = append(rAdd.Params, "/DisableStartupMessages")
		rAdd.Params = append(rAdd.Params, "/DisableUnrecoverableErrorMessage")
		rAdd.Params = append(rAdd.Params, "/UC ServiceMode")
		// Параметры создания хранилища расширения
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryCreate")
		rAdd.Params = append(rAdd.Params, "-Extension")
		rAdd.Params = append(rAdd.Params, addName)
		rAdd.Params = append(rAdd.Params, "-AllowConfigurationChanges")
		rAdd.Params = append(rAdd.Params, "-ChangesAllowedRule")
		rAdd.Params = append(rAdd.Params, "ObjectIsEditableSupportEnabled")
		rAdd.Params = append(rAdd.Params, "-ChangesNotRecommendedRule")
		rAdd.Params = append(rAdd.Params, "ObjectIsEditableSupportEnabled")
		rAdd.Params = append(rAdd.Params, "/Out")
		rAdd.Params = append(rAdd.Params, "/c Создание хранилища конфигурации")

		_, err := rAdd.RunCommand(context.Background(), l)
		if err != nil {
			l.Error("Ошибка создания хранилища конфигурации",
				slog.String("Путь", addStore.Path),
				slog.String("Строка подключения", dbConnectString),
				slog.String("Error", err.Error()),
			)
			return fmt.Errorf("ошибка создания хранилища конфигурации")
		}
		if !strings.Contains(string(rAdd.FileOut), constants.SearchMsgStoreCreateOk) {
			l.Error("Неопознанная ошибка создания хранилища конфигурации",
				slog.String("Путь", addStore.Path),
				slog.String("Строка подключения", dbConnectString),
				slog.String("Вывод в файл", runner.TrimOut(rAdd.FileOut)),
			)
			return fmt.Errorf("ошибка создания хранилища конфигурации")
		}

		l.Debug("Хранилище расширения успешно создано",
			slog.String("path", addStore.Path),
			slog.String("extensionName", addName),
		)
	}

	l.Debug("Все хранилища успешно созданы",
		slog.String("storeRoot", storeRoot),
		slog.Int("extensionsCount", len(arrayAdd)),
	)

	return nil
}
