package edt

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Kargones/apk-ci/internal/config"
)

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
