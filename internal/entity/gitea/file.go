package gitea

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Kargones/apk-ci/internal/constants"
)

// GetFileContent получает содержимое файла из репозитория Gitea или по прямому URL.
// Извлекает содержимое указанного файла из корня репозитория для анализа
// или обработки. Возвращает декодированное содержимое файла.
// Если fileName содержит полный URL (начинающийся с http:// или https://),
// то используется этот URL напрямую без построения пути через API Gitea.
// Параметры:
//   - fileName: имя файла в корне репозитория или полный URL для загрузки
//
// Возвращает:
//   - []byte: содержимое файла в виде массива байт
//   - error: ошибка получения файла или nil при успехе
func (g *API) GetFileContent(fileName string) ([]byte, error) {
	var urlString string
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		urlString = fileName
	} else {
		urlString = fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, fileName)
	}

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении файла %s: статус %d", fileName, statusCode)
	}

	r := strings.NewReader(body)
	var fileData struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(r).Decode(&fileData); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании ответа: %w", err)
	}

	// Декодируем base64 содержимое
	if fileData.Encoding == "base64" {
		content := strings.ReplaceAll(fileData.Content, "\n", "")
		decodedBytes, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("ошибка при декодировании base64: %w", err)
		}
		return decodedBytes, nil
	}

	return nil, nil
}

// GetRepositoryContents получает содержимое директории репозитория.
// Возвращает список файлов и каталогов из указанного пути в репозитории
// для анализа структуры проекта или навигации по файлам.
// Параметры:
//   - filepath: путь к директории в репозитории
//   - branch: имя ветки для получения содержимого
//
// Возвращает:
//   - []FileInfo: список файлов и каталогов с метаданными
//   - error: ошибка получения содержимого или nil при успехе
func (g *API) GetRepositoryContents(filepath, branch string) ([]FileInfo, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s?ref=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, filepath, branch)

	statusCode, body, err := g.sendReq(urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении содержимого репозитория: статус %d", statusCode)
	}

	var files []FileInfo
	err = json.Unmarshal([]byte(body), &files)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return files, nil
}

// SetRepositoryState устанавливает состояние файлов в репозитории.
// Выполняет множественные операции с файлами (создание, изменение, удаление)
// в рамках одного коммита для атомарного обновления состояния репозитория.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - operations: массив операций для выполнения
//   - branch: имя ветки для применения изменений
//   - commitMessage: сообщение коммита
//
// Возвращает:
//   - error: ошибка выполнения операций или nil при успехе
func (g *API) SetRepositoryState(l *slog.Logger, operations []ChangeFileOperation, branch, commitMessage string) error {
	if len(operations) == 0 {
		return fmt.Errorf("список операций не может быть пустым")
	}

	// Создаем запрос для batch коммита
	request := ChangeFilesOptions{
		Branch: branch,
		Author: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Committer: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Message: commitMessage,
		Files:   operations,
	}

	// Сериализуем запрос в JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("ошибка при сериализации запроса: %w", err)
	}

	// Формируем URL для batch API
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	l.Debug("Выполнение batch запроса", "url", urlString, "body", string(requestBody))
	// Выполняем POST запрос
	statusCode, responseBody, err := g.sendReq(urlString, string(requestBody), "POST")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении batch запроса: %w", err)
	}

	// Проверяем статус ответа
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return fmt.Errorf("ошибка при выполнении batch операций: статус %d, ответ: %s", statusCode, responseBody)
	}

	return nil
}

// SetRepositoryStateWithNewBranch выполняет множественные операции с файлами и создаёт новую ветку.
// Аналогичен SetRepositoryState, но дополнительно создаёт новую ветку от указанной базовой ветки.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - operations: массив операций для выполнения
//   - baseBranch: базовая ветка (откуда создаётся новая)
//   - newBranch: имя новой ветки для создания
//   - commitMessage: сообщение коммита
//
// Возвращает:
//   - string: SHA созданного коммита
//   - error: ошибка выполнения операций или nil при успехе
func (g *API) SetRepositoryStateWithNewBranch(l *slog.Logger, operations []ChangeFileOperation, baseBranch, newBranch, commitMessage string) (string, error) {
	if len(operations) == 0 {
		return "", fmt.Errorf("список операций не может быть пустым")
	}

	// Валидация операций - проверяем на пустые пути
	for i, op := range operations {
		if op.Path == "" {
			l.Error("SetRepositoryStateWithNewBranch: обнаружена операция с пустым путём",
				slog.Int("index", i),
				slog.String("operation", op.Operation),
				slog.String("sha", op.SHA),
			)
			return "", fmt.Errorf("операция %d имеет пустой путь (operation=%s)", i, op.Operation)
		}
	}

	// Логируем информацию о запросе
	l.Debug("SetRepositoryStateWithNewBranch: подготовка batch запроса",
		slog.Int("operations_count", len(operations)),
		slog.String("baseBranch", baseBranch),
		slog.String("newBranch", newBranch),
		slog.String("repo", fmt.Sprintf("%s/%s", g.Owner, g.Repo)),
	)

	// Создаем запрос для batch коммита с новой веткой
	request := ChangeFilesOptions{
		Branch:    baseBranch,
		NewBranch: newBranch,
		Author: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Committer: &Identity{
			Name:  constants.DefaultCommitAuthorName,
			Email: constants.DefaultCommitAuthorEmail,
		},
		Message: commitMessage,
		Files:   operations,
	}

	// Сериализуем запрос в JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("ошибка при сериализации запроса: %w", err)
	}

	// Формируем URL для batch API
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	// Логируем пути операций для диагностики (без содержимого файлов)
	var operationPaths []string
	for _, op := range operations {
		operationPaths = append(operationPaths, fmt.Sprintf("%s:%s", op.Operation, op.Path))
	}
	l.Debug("SetRepositoryStateWithNewBranch: пути операций",
		slog.Any("operations", operationPaths),
	)

	l.Debug("Выполнение batch запроса с новой веткой", "url", urlString, "newBranch", newBranch)
	// Выполняем POST запрос
	statusCode, responseBody, err := g.sendReq(urlString, string(requestBody), "POST")
	if err != nil {
		return "", fmt.Errorf("ошибка при выполнении batch запроса: %w", err)
	}

	// Проверяем статус ответа
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		// Логируем детали ошибки для диагностики
		l.Error("SetRepositoryStateWithNewBranch: ошибка batch операций",
			slog.Int("status_code", statusCode),
			slog.String("response", responseBody),
			slog.String("repo", fmt.Sprintf("%s/%s", g.Owner, g.Repo)),
			slog.String("newBranch", newBranch),
			slog.Int("operations_count", len(operations)),
		)
		return "", fmt.Errorf("ошибка при выполнении batch операций: статус %d, ответ: %s", statusCode, responseBody)
	}

	// Парсим ответ для получения SHA коммита
	var response struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		// Если не удалось распарсить, возвращаем пустой SHA, но без ошибки
		l.Warn("Не удалось получить SHA коммита из ответа", "error", err)
		return "", nil
	}

	return response.Commit.SHA, nil
}

// GetConfigData получает данные конфигурации из файла в репозитории.
// Загружает и декодирует содержимое конфигурационного файла для
// использования в настройках приложения или процессах развертывания.
// Параметры:
//   - l: логгер для записи отладочной информации
//   - filename: имя файла конфигурации или URL для загрузки
//
// Возвращает:
//   - []byte: содержимое файла в виде массива байт
//   - error: ошибка получения данных или nil при успехе
func (g *API) GetConfigData(l *slog.Logger, filename string) ([]byte, error) {
	var fileURL string

	l.Debug("GetConfigData started",
		"filename", filename,
		"giteaURL", g.GiteaURL,
		"owner", g.Owner,
		"repo", g.Repo,
		"baseBranch", g.BaseBranch,
		"hasAccessToken", g.AccessToken != "")

	// Determine source based on filename prefix
	if strings.HasPrefix(filename, "https://") {
		l.Debug("Using direct URL", "url", filename)
		fileURL = filename
	} else {
		// Repository files - use current owner/repo
		// Build URL for file content API
		fileURL = fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s?ref=%s", g.GiteaURL, g.Owner, g.Repo, filename, g.BaseBranch)
		l.Debug("Built repository file URL",
			"owner", g.Owner,
			"repo", g.Repo,
			"filename", filename,
			"baseBranch", g.BaseBranch,
			"fileURL", fileURL)
	}

	// Make HTTP request
	l.Debug("Creating HTTP request", "fileURL", fileURL, "filename", filename)
	statusCode, respBody, err := g.sendReq(fileURL, "", "GET")
	if err != nil {
		l.Error("Failed to create HTTP request",
			"filename", filename,
			"fileURL", fileURL,
			"error", err)
		return nil, fmt.Errorf("failed to create request for file %s: %w", filename, err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP ошибка %d при загрузке %s", statusCode, filename)
	}

	l.Debug("Reading response body")
	body, err := io.ReadAll(strings.NewReader(respBody))
	if err != nil {
		l.Error("Failed to read response body",
			"filename", filename,
			"error", err)
		return nil, fmt.Errorf("failed to read response for file %s: %w", filename, err)
	}

	// Parse JSON response to get content
	var fileData struct {
		Content string `json:"content"`
	}

	l.Debug("Parsing JSON response")
	if unmarshalErr := json.Unmarshal(body, &fileData); unmarshalErr != nil {
		l.Error("Failed to parse JSON response",
			"filename", filename,
			"error", unmarshalErr,
			"responseBody", string(body))
		return nil, fmt.Errorf("failed to parse response for file %s: %w", filename, unmarshalErr)
	}

	l.Debug("JSON response parsed",
		"contentLength", len(fileData.Content),
		"filename", filename)

	// Decode base64 content
	l.Debug("Decoding base64 content")
	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileData.Content, "\n", ""))
	if err != nil {
		l.Error("Failed to decode base64 content",
			"filename", filename,
			"error", err)
		return nil, fmt.Errorf("failed to decode content for file %s: %w", filename, err)
	}

	l.Debug("Base64 content decoded",
		"decodedLength", len(content),
		"filename", filename)

	// Check if data is empty
	if len(content) == 0 {
		l.Error("Decoded content is empty", "filename", filename)
		return nil, fmt.Errorf("data for file %s is empty", filename)
	}

	l.Debug("GetConfigData completed successfully",
		"filename", filename,
		"finalContentLength", len(content))

	return content, nil
}

// GetConfigDataBad получает данные конфигурации по префиксу имени файла.
// Устаревший метод для поиска и загрузки конфигурационных файлов
// по префиксу имени в корневой директории репозитория.
// Параметры:
//   - filenamePrefix: префикс имени файла для поиска
//
// Возвращает:
//   - []byte: содержимое найденного файла
//   - error: ошибка поиска или загрузки файла или nil при успехе
func (g *API) GetConfigDataBad(filenamePrefix string) ([]byte, error) {
	// Если это прямая ссылка, загружаем напрямую
	if strings.HasPrefix(filenamePrefix, "http://") || strings.HasPrefix(filenamePrefix, "https://") {
		statusCode, body, err := g.sendReq(filenamePrefix, "", "GET")

		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки по URL %s: %w", filenamePrefix, err)
		}

		if statusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP ошибка %d при загрузке %s", statusCode, filenamePrefix)
		}

		return []byte(body), nil
	}

	// Получаем содержимое корневой директории репозитория
	contents, err := g.GetRepositoryContents("", g.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения содержимого репозитория: %w", err)
	}

	// Ищем файл с нужным префиксом
	for _, file := range contents {
		if file.Type == "file" && strings.HasPrefix(file.Name, filenamePrefix) {
			// Получаем содержимое файла
			urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s?ref=%s",
				g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, file.Path, g.BaseBranch)

			statusCode, body, err := g.sendReq(urlString, "", "GET")
			if err != nil {
				return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
			}

			if statusCode != http.StatusOK {
				return nil, fmt.Errorf("ошибка API: %d - %s", statusCode, body)
			}

			r := strings.NewReader(body)
			var fileInfo FileInfo
			if err := json.NewDecoder(r).Decode(&fileInfo); err != nil {
				return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
			}

			// Декодируем содержимое из base64
			if fileInfo.Encoding == "base64" {
				content := strings.ReplaceAll(fileInfo.Content, "\n", "")
				decodedBytes, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return nil, fmt.Errorf("ошибка декодирования base64: %w", err)
				}
				return decodedBytes, nil
			}

			return []byte(fileInfo.Content), nil
		}
	}

	return nil, fmt.Errorf("файл с префиксом '%s' не найден", filenamePrefix)
}
