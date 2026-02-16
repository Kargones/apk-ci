// Package edt предоставляет функциональность для работы с EDT (Enterprise Development Tools)
package edt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/git"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

const (
	// XML2edt - путь к утилите xml2edt.
	XML2edt string = "xml2edt"
	// Edt2xml - путь к утилите edt2xml.
	Edt2xml   string = "edt2xml"
	formatXML string = "xml"
	formatEDT string = "edt"
)

// Convert представляет конфигурацию для конвертации между форматами EDT и XML.
// Содержит информацию о коммите, источнике, приемнике и сопоставлениях путей
// для выполнения процесса конвертации проекта.
type Convert struct {
	CommitSha1  string    `json:"Хеш коммита"`
	Source      Data      `json:"Источник"`
	Distination Data      `json:"Приемник"`
	Mappings    []Mapping `json:"Сопоставление путей"`
}

// Data представляет данные источника или приемника конвертации.
// Содержит информацию о формате данных и ветке репозитория
// для процесса конвертации.
type Data struct {
	Format string `json:"Формат"`
	Branch string `json:"Ветка"`
}

// Mapping представляет сопоставление путей для конвертации.
// Определяет соответствие между путями источника и приемника
// в процессе конвертации файлов проекта.
type Mapping struct {
	SourcePath      string `json:"Путь источника"`
	DistinationPath string `json:"Путь приемника"`
}

// Cli представляет интерфейс командной строки для работы с EDT.
// Содержит настройки путей, направления конвертации, рабочей области
// и информацию о последней ошибке выполнения операций.
// EdtCli представляет клиент командной строки для работы с EDT.
// Содержит настройки путей, направления конвертации, рабочей области
// и информацию о последней ошибке выполнения операций.
// Cli представляет клиент командной строки для работы с EDT.
// Содержит настройки путей, направления конвертации, рабочей области
// и информацию о последней ошибке выполнения операций.
type Cli struct {
	CliPath   string
	Direction string
	PathIn    string
	PathOut   string
	WorkSpace string
	Operation string
	LastErr   error
}

func cleanDirectoryPreservingHidden(targetDir string, l *slog.Logger) error {
	// 1. Проверяем, существует ли указанный путь и является ли он каталогом
	dirInfo, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("каталог '%s' не существует", targetDir)
		}
		return fmt.Errorf("не удалось получить информацию о '%s': %w", targetDir, err)
	}

	if !dirInfo.IsDir() {
		return fmt.Errorf("путь '%s' не является каталогом", targetDir)
	}

	// 2. Читаем содержимое каталога
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("не удалось прочитать содержимое каталога '%s': %w", targetDir, err)
	}

	var encounteredErrors []string

	// 3. Итерируемся по элементам
	for _, entry := range entries {
		entryName := entry.Name()

		// 4. Пропускаем файлы и каталоги, начинающиеся с точки
		if strings.HasPrefix(entryName, ".") {
			l.Debug("Пропуск скрытого файла/каталога", slog.String("path", filepath.Join(targetDir, entryName)))
			continue
		}

		fullPath := filepath.Join(targetDir, entryName)

		// 5. Удаляем
		if entry.IsDir() {
			l.Debug("Удаление каталога рекурсивно", slog.String("path", fullPath))
			// os.RemoveAll удаляет каталог и все его содержимое
			if err := os.RemoveAll(fullPath); err != nil {
				errorMsg := fmt.Sprintf("ошибка при удалении каталога '%s': %v", fullPath, err)
				fmt.Fprintln(os.Stderr, errorMsg)
				encounteredErrors = append(encounteredErrors, errorMsg)
			}
		} else {
			l.Debug("Удаление файла", slog.String("path", fullPath))
			if err := os.Remove(fullPath); err != nil {
				errorMsg := fmt.Sprintf("ошибка при удалении файла '%s': %v", fullPath, err)
				fmt.Fprintln(os.Stderr, errorMsg)
				encounteredErrors = append(encounteredErrors, errorMsg)
			}
		}
	}

	if len(encounteredErrors) > 0 {
		return fmt.Errorf("возникло %d ошибок во время очистки:\n%s",
			len(encounteredErrors), strings.Join(encounteredErrors, "\n"))
	}

	return nil
}

// Convert выполняет конвертацию между форматами EDT и XML
func (c *Convert) Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	r := Cli{}
	r.Init(cfg)
	var ok bool
	switch c.Source.Format {
	default:
		l.Error("Неизвестный формат",
			slog.String("Указанный формат", c.Source.Format),
		)
		return fmt.Errorf("неизвестный формат")
	case formatXML:
		r.Direction = XML2edt
	case formatEDT:
		r.Direction = Edt2xml
	}
	WorkSpace, err := os.MkdirTemp(cfg.WorkDir, "ws")
	if err != nil {
		l.Error("Не удалось создать временный каталог",
			slog.String("Корень временных каталогов", cfg.TmpDir),
			slog.String("Текст ошибки", err.Error()),
		)
		return err
	}

	r.WorkSpace = WorkSpace
	g := git.Git{}
	g.WorkDir = cfg.WorkDir
	repSourcePath := cfg.RepPath
	repDistinationPath, err := os.MkdirTemp(cfg.WorkDir, "d")
	if err != nil {
		l.Error("Не удалось создать временный каталог",
			slog.String("Корень временных каталогов", cfg.TmpDir),
			slog.String("Текст ошибки", err.Error()),
		)
		return err
	}

	for i, m := range c.Mappings {
		r.PathIn = path.Join(repSourcePath, m.SourcePath)
		r.PathOut = path.Join(repDistinationPath, m.DistinationPath)
		
		l.Debug("Проверка существования исходного каталога",
			slog.Int("Итерация", i),
			slog.String("SourcePath", m.SourcePath),
			slog.String("repSourcePath", repSourcePath),
			slog.String("Полный путь", r.PathIn),
		)
		
		ok, err = exists(r.PathIn)
		if !ok {
			l.Error("Отсутствует исходный каталог",
				slog.String("Каталог", r.PathIn),
				slog.String("Error", fmt.Sprint(err)),
				slog.String("Обработка ошибки", "Завершение работы программы"),
			)
			return fmt.Errorf("отсутствует исходный каталог: %s", r.PathIn)
		}
		ok, _ = exists(r.PathOut)
		if ok {
			l.Debug("Начало очистки каталога",
				slog.String("Путь очистки", r.PathOut),
			)
			err = os.RemoveAll(r.PathOut)
			if err != nil {
				l.Error("Не удалось очистить каталог приемник",
					slog.String("Каталог", r.PathOut),
					slog.String("Error", err.Error()),
					slog.String("Обработка ошибки", "Завершение работы программы"),
				)
				return fmt.Errorf("не удалось очистить каталог приемник %s: %w", r.PathOut, err)
			}
			err = os.MkdirAll(r.PathOut, 0750)
			if err != nil {
				l.Error("Не удалось создать каталог после очистки",
					slog.String("Каталог", r.PathOut),
					slog.String("Ошибка", err.Error()),
				)
			}
			l.Debug("Окончание очистки каталога",
				slog.String("Путь очистки", r.PathOut),
			)
		}
		err = os.MkdirAll(r.PathOut, 0750)
		if err != nil {
			l.Error("Не удалось создать каталог",
				slog.String("Каталог", r.PathOut),
				slog.String("Error", err.Error()),
				slog.String("Обработка ошибки", "Завершение работы программы"),
			)
			return fmt.Errorf("не удалось создать каталог %s: %w", r.PathOut, err)
		}
		l.Debug("Конвертация каталога",
			slog.String("Номер итерации", strconv.Itoa(i)),
			slog.String("Каталог источника", m.SourcePath),
			slog.String("Каталог приемника", m.DistinationPath),
		)
		r.Convert(ctx, l, cfg)
		if r.LastErr != nil {
			l.Error("ошибка конвертации",
				slog.String("Направление конвертации", r.Direction),
				slog.String("Исходный", r.PathIn),
				slog.String("Конечный", r.PathOut),
				slog.String("Error", r.LastErr.Error()),
				slog.String("Обработка ошибки", "Завершение работы программы"),
			)
			return fmt.Errorf("ошибка конвертации %s -> %s: %w", r.PathIn, r.PathOut, r.LastErr)
		}
		l.Info("Каталог конвертирован",
			slog.String("Исходный", r.PathIn),
			slog.String("Конечный", r.PathOut),
		)
	}
	g.RepPath = repSourcePath
	g.Branch = c.Distination.Branch
	if err := g.Switch(*ctx, l); err != nil {
		l.Error("Ошибка переключения на целевую ветку",
			slog.String("Описание ошибки", err.Error()),
			slog.String("Ветка", c.Distination.Branch),
		)
		return err
	}
	err = cleanDirectoryPreservingHidden(repSourcePath, l)
	if err != nil {
		l.Warn("Failed to clean directory preserving hidden files",
			slog.String("path", repSourcePath),
			slog.String("error", err.Error()),
		)
	}
	if err := g.Config(*ctx, l); err != nil {
		l.Warn("Failed to configure git",
			slog.String("error", err.Error()),
		)
	}

	err = MoveDirContents(repDistinationPath, repSourcePath)
	if err != nil {
		l.Error("ошибка копирования",
			slog.String("Направление конвертации", r.Direction),
			slog.String("Исходный", repSourcePath),
			slog.String("Конечный", repDistinationPath),
			slog.String("Error", r.LastErr.Error()),
			slog.String("Обработка ошибки", "Завершение работы программы"),
		)
		return fmt.Errorf("ошибка копирования из %s в %s: %w", repDistinationPath, repSourcePath, err)
	}
	if err := g.Add(*ctx, l); err != nil {
		l.Warn("Failed to add files to git",
			slog.String("error", err.Error()),
		)
	}
	if err := g.SetUser(*ctx, l, "gitops", "gitops@apkholding.ru"); err != nil {
		l.Warn("Failed to set git user",
			slog.String("error", err.Error()),
		)
	}
	if err := g.Commit(*ctx, l, GetComment(c)); err != nil {
		l.Warn("Failed to commit changes",
			slog.String("error", err.Error()),
		)
	}
	if err := g.PushForce(*ctx, l); err != nil {
		l.Warn("Failed to push changes",
			slog.String("error", err.Error()),
		)
	}
	return nil
}

// MoveDirContents перемещает содержимое одного каталога в другой.
// Функция читает все элементы из исходного каталога и перемещает их
// в целевой каталог, используя операцию переименования.
// Параметры:
//   - src: путь к исходному каталогу
//   - dst: путь к целевому каталогу
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func MoveDirContents(src, dst string) error {
	// Получаем список элементов в исходном каталоге
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("не удалось прочитать исходный каталог: %v", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Просто переименовываем (перемещаем) каждый элемент
		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("не удалось переместить %s: %v", srcPath, err)
		}
	}

	return nil
}

// GetComment возвращает комментарий для коммита.
// Функция генерирует стандартный комментарий для автоматических коммитов
// в процессе конвертации проекта.
// Параметры:
//   - _: конфигурация конвертации (не используется)
//
// Возвращает:
//   - string: текст комментария для коммита
func GetComment(_ *Convert) string {
	return "Конвертирован автоматически"
}

// MustLoad загружает конфигурацию конвертации с настройками по умолчанию
func (c *Convert) MustLoad(_ *slog.Logger, cfg *config.Config) error {
	var err error

	c.CommitSha1 = ""
	c.Source = Data{
		Format: "edt",
		Branch: "main",
	}
	c.Distination = Data{
		Format: "xml",
		Branch: "xml",
	}

	c.Mappings = []Mapping{
		{
			SourcePath:      cfg.ProjectName,
			DistinationPath: "src/cfg",
		},
	}
	for _, v := range cfg.AddArray {
		// ToDo: Добавить проверку на вхождение расширения в список запрещенных для этой информационной базы
		c.Mappings = append(c.Mappings, Mapping{
			SourcePath:      cfg.ProjectName + "." + v,
			DistinationPath: "src/cfe/" + v,
		})
	}
	return err
}

// Init - устаревшая функция, оставлена для обратной совместимости
// Рекомендуется использовать InitWithEdtConfig
// Init инициализирует EDT CLI с настройками конфигурации
func (e *Cli) Init(cfg *config.Config) {
	e.CliPath = cfg.AppConfig.Paths.EdtCli
	e.Direction = ""
	e.PathIn = cfg.RepPath
	e.PathOut = cfg.WorkDir
	e.WorkSpace = cfg.WorkSpace
	e.Operation = ""
	e.LastErr = nil
}

// Convert выполняет конвертацию с использованием EDT CLI
func (e *Cli) Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) {
	if e.WorkSpace == "" {
		tDir, err := os.MkdirTemp(cfg.TmpDir, "ws")
		if err != nil {
			panic(err)
		}
		e.WorkSpace = tDir
	}
	var path1, path2 string

	switch e.Direction {
	default: // Неопознанная операция
		l.Error("Неопознанная операция",
			slog.String("Название операции", e.Direction),
		)
		e.LastErr = fmt.Errorf("неопознанная операция")
		return
	case XML2edt:
		e.Operation = "import"
		if e.PathOut == "" {
			tDir, err := os.MkdirTemp(cfg.TmpDir, "e")
			if err != nil {
				l.Error("Ошибка создания временного каталога",
					slog.String("Временный каталог", cfg.TmpDir),
					slog.String("Ошибка", err.Error()),
				)
				e.LastErr = fmt.Errorf("ошибка создания временного каталога")
				return
			}
			e.PathOut = tDir
		}
		path1 = e.PathIn
		path2 = e.PathOut
	case Edt2xml:
		if e.PathOut == "" {
			tDir, err := os.MkdirTemp(cfg.TmpDir, "x")
			if err != nil {
				l.Error("Ошибка создания временного каталога",
					slog.String("Временный каталог", cfg.TmpDir),
					slog.String("Ошибка", err.Error()),
				)
				e.LastErr = fmt.Errorf("ошибка создания временного каталога")
				return
			}
			e.PathOut = tDir
		}
		e.Operation = "export"
		path2 = e.PathIn
		path1 = e.PathOut
	}
	r := runner.Runner{}
	r.RunString = cfg.AppConfig.Paths.EdtCli
	r.TmpDir = cfg.WorkDir
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "-data")
	r.Params = append(r.Params, e.WorkSpace)
	r.Params = append(r.Params, "-command")
	r.Params = append(r.Params, e.Operation)
	r.Params = append(r.Params, "--configuration-files")
	r.Params = append(r.Params, path1)
	r.Params = append(r.Params, "--project")
	r.Params = append(r.Params, path2)
	l.Debug("Параметры запуска",
		slog.String("Операция", fmt.Sprintf("%v", r.Params)),
	)

	l.Debug("Запуск команды EDT", slog.String("Операция", e.Operation))
	_, e.LastErr = r.RunCommand(*ctx, l)
	if e.LastErr != nil {
		// Проверяем, не был ли превышен таймаут
		if errors.Is(e.LastErr, context.DeadlineExceeded) {
			l.Error("Превышен таймаут выполнения команды EDT",
				slog.String("Операция", e.Operation),
				slog.String("PathIn", e.PathIn),
				slog.String("PathOut", e.PathOut),
			)
			return
		}

		switch e.LastErr.Error() {
		case "exec: already started":
			e.LastErr = nil
		default:
			l.Error("Ошибка выполнения команды EDT",
				slog.String("Операция", e.Operation),
				slog.String("PathIn", e.PathIn),
				slog.String("PathOut", e.PathOut),
				slog.String("Ошибка", e.LastErr.Error()),
			)
			return
		}
	}

	l.Debug("Команда EDT выполнена успешно",
		slog.String("Операция", e.Operation))
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
