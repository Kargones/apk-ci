package gitea

import (
	"context"
	"fmt"
	"strings"
)

// Функции TestMerge и AnalyzeProject перенесены в сервисный слой (service/gitea_service.go)

// AnalyzeProjectStructure анализирует структуру проекта 1С.
// Определяет название основного проекта и список расширений конфигурации
// на основе структуры каталогов в репозитории.
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив строк где первый элемент - название проекта, остальные - имена расширений
//   - error: ошибка анализа или nil при успехе
//
// analyzeProjectStructure анализирует структуру проекта 1С.
// Определяет название основного проекта и список расширений конфигурации
// на основе структуры каталогов в репозитории.
// Параметры:
//   - directories: список каталогов в репозитории
//
// Возвращает:
//   - []string: массив строк где первый элемент - название проекта, остальные - имена расширений
//   - error: ошибка анализа или nil при успехе
func analyzeProjectStructure(directories []string) ([]string, error) {
	if len(directories) == 0 {
		return []string{}, nil
	}

	// Находим каталоги без точки в середине
	var dirsWithoutDot []string
	for _, dir := range directories {
		if !strings.Contains(dir, ".") {
			dirsWithoutDot = append(dirsWithoutDot, dir)
		}
	}

	// Если нет каталогов без точки, возвращаем пустой результат
	if len(dirsWithoutDot) == 0 {
		return []string{}, nil
	}

	// Единственный каталог без точки - это название проекта
	projectName := dirsWithoutDot[0]

	// Ищем каталоги вида <projectName>.<расширение> (расширения конфигурации 1С)
	var extensions []string
	for _, dir := range directories {
		if strings.HasPrefix(dir, projectName+".") {
			ext := strings.TrimPrefix(dir, projectName+".")
			if ext != "" {
				extensions = append(extensions, ext)
			}
		}
	}

	// Если не найдены расширения и каталогов без точки больше одного - ошибка
	// (этот случай уже обработан выше, но добавляем для ясности логики)
	if len(extensions) == 0 && len(dirsWithoutDot) > 1 {
		return []string{}, fmt.Errorf("не найдены расширения конфигурации, а каталогов без точки больше одного: %v", dirsWithoutDot)
	}

	// Формируем результат: первый элемент - название проекта, остальные - расширения
	result := make([]string, 0, 1+len(extensions))
	result = append(result, projectName)
	result = append(result, extensions...)

	return result, nil
}

// AnalyzeProjectStructure анализирует структуру проекта 1С.
// Определяет название основного проекта и список расширений конфигурации
// на основе структуры каталогов в репозитории.
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив строк где первый элемент - название проекта, остальные - имена расширений
//   - error: ошибка анализа или nil при успехе
func (g *API) AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error) {
	files, err := g.GetRepositoryContents(ctx, ".", branch)
	if err != nil {
		return []string{}, err
	}

	// Получаем только каталоги, не начинающиеся с точки
	var directories []string
	for _, file := range files {
		if file.Type == "dir" && !strings.HasPrefix(file.Name, ".") {
			directories = append(directories, file.Name)
		}
	}

	return analyzeProjectStructure(directories)
}

// AnalyzeProject анализирует структуру проекта 1С в репозитории.
// Определяет название основного проекта и список расширений конфигурации
// на основе анализа структуры каталогов в корне репозитория.
// Логика анализа:
// 1. Получает список каталогов в корне репозитория (исключая скрытые)
// 2. Ищет расширения конфигурации 1С (каталоги вида <projectName>.<расширение>)
// 3. Если каталогов без точки больше одного - возвращает ошибку
// 4. Если только один каталог без точки - возвращает массив с названием проекта и расширениями
// 5. Если не найдены расширения, но каталогов без точки больше одного - возвращает ошибку
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив где первый элемент - название проекта, остальные - расширения
//   - error: ошибка анализа или nil при успехе
//
// AnalyzeProject анализирует структуру проекта 1С в репозитории.
// Определяет название основного проекта и список расширений конфигурации
// на основе анализа структуры каталогов в корне репозитория.
// Логика анализа:
// 1. Получает список каталогов в корне репозитория (исключая скрытые)
// 2. Ищем расширения конфигурации 1С (каталоги вида <projectName>.<расширение>)
// 3. Если каталогов без точки больше одного - возвращает ошибку
// 4. Если только один каталог без точки - возвращает массив с названием проекта и расширениями
// 5. Если не найдены расширения, но каталогов без точки больше одного - возвращает ошибку
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - []string: массив где первый элемент - название проекта, остальные - расширения
//   - error: ошибка анализа или nil при успехе
func (g *API) AnalyzeProject(ctx context.Context, branch string) ([]string, error) {
	files, err := g.GetRepositoryContents(ctx, "", branch)
	if err != nil {
		return []string{}, err
	}

	// Получаем только каталоги, не начинающиеся с точки
	var directories []string
	for _, file := range files {
		if file.Type == "dir" && !strings.HasPrefix(file.Name, ".") {
			directories = append(directories, file.Name)
		}
	}

	return analyzeProjectStructure(directories)
}
