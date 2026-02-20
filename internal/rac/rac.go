// Package rac предоставляет функциональность для работы с RAC (Remote Administration Console) 1C
package rac

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/Kargones/apk-ci/internal/util/runner"
)

// Client представляет клиент для работы с RAC (Remote Administration Console) 1C.
// Содержит все необходимые параметры для подключения и выполнения
// административных команд на сервере 1C.
type Client struct {
	// RacPath - путь к исполняемому файлу RAC
	RacPath string
	// Server - адрес сервера 1C
	Server string
	// Port - порт сервера 1C
	Port int
	// User - имя пользователя для подключения к серверу
	User string
	// Password - пароль пользователя для подключения к серверу
	Password string
	// DbUser - имя пользователя базы данных
	DbUser string
	// DbPassword - пароль пользователя базы данных
	DbPassword string
	// Timeout - таймаут выполнения команд
	Timeout time.Duration
	// Retries - количество повторных попыток при ошибке
	Retries int
	// Logger - логгер для записи сообщений
	Logger *slog.Logger
}

// NewClient создает новый экземпляр RAC клиента.
// Инициализирует клиент с указанными параметрами подключения
// и настройками для работы с сервером 1C.
// Параметры:
//   - racPath: путь к исполняемому файлу RAC
//   - server: адрес сервера 1C
//   - port: порт сервера 1C
//   - user: имя пользователя для подключения
//   - password: пароль пользователя
//   - dbUser: имя пользователя базы данных
//   - dbPassword: пароль пользователя базы данных
//   - timeout: таймаут выполнения команд
//   - retries: количество повторных попыток
//   - logger: логгер для записи сообщений
//
// Возвращает:
//   - *Client: новый экземпляр RAC клиента
func NewClient(racPath, server string, port int, user, password, dbUser, dbPassword string, timeout time.Duration, retries int, logger *slog.Logger) *Client {
	return &Client{
		RacPath:    racPath,
		Server:     server,
		Port:       port,
		User:       user,
		Password:   password,
		DbUser:     dbUser,
		DbPassword: dbPassword,
		Timeout:    timeout,
		Retries:    retries,
		Logger:     logger,
	}
}

// ExecuteCommand выполняет RAC команду с повторными попытками.
// Отправляет команду на сервер 1C через RAC и обрабатывает результат.
// При неудаче автоматически повторяет попытки согласно настройкам.
// Параметры:
//   - ctx: контекст выполнения операции
//   - args: аргументы команды RAC
//
// Возвращает:
//   - string: результат выполнения команды
//   - error: ошибка выполнения или nil при успехе
func (c *Client) ExecuteCommand(ctx context.Context, args ...string) (string, error) {
	var lastErr error
	for i := 0; i < c.Retries; i++ {
		output, err := c.executeCommandOnce(ctx, args...)
		if err == nil {
			return output, nil
		}
		lastErr = err
		// Логируем ошибку с уже сконвертированным текстом
		c.Logger.Warn("RAC command failed, retrying", "attempt", i+1, "error", err.Error())
		if i < c.Retries-1 {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}
	return "", fmt.Errorf("RAC command failed after %d attempts: %w", c.Retries, lastErr)
}

// convertOutputToUTF8 конвертирует текст из различных кодировок в UTF-8
func convertOutputToUTF8(input []byte) (string, error) {
	// Сначала пробуем интерпретировать как UTF-8
	if utf8String := string(input); isValidUTF8(utf8String) {
		return utf8String, nil
	}

	// Если не UTF-8, пробуем CP1251
	decoder := charmap.Windows1251.NewDecoder()
	reader := transform.NewReader(bytes.NewReader(input), decoder)
	result, err := io.ReadAll(reader)
	if err != nil {
		// Если конвертация не удалась, возвращаем исходный текст
		return string(input), err
	}
	return string(result), nil
}

// isValidUTF8 проверяет, является ли строка валидным UTF-8
func isValidUTF8(s string) bool {
	return utf8.ValidString(s)
}

// executeCommandOnce выполняет RAC команду один раз
func (c *Client) executeCommandOnce(ctx context.Context, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// Валидация пути к RAC
	if c.RacPath == "" {
		return "", fmt.Errorf("RAC path is empty")
	}

	// Формируем полную команду
	serverAddress := c.Server + ":" + strconv.Itoa(c.Port)
	fullArgs := make([]string, 0, 1+len(args))
	fullArgs = append(fullArgs, serverAddress)
	fullArgs = append(fullArgs, args...)

	// Валидация аргументов
	for _, arg := range fullArgs {
		if strings.Contains(arg, ";") || strings.Contains(arg, "&") || strings.Contains(arg, "|") {
			return "", fmt.Errorf("potentially unsafe argument detected: %s", arg)
		}
	}

	// Добавляем детальное логирование параметров подключения
	c.Logger.Debug("Executing RAC command",
		"racPath", c.RacPath,
		"serverAddress", serverAddress,
		"timeout", c.Timeout,
		"fullCommand", append([]string{c.RacPath}, fullArgs...),
		"args", fullArgs)

	// Создаем runner для выполнения команды
	r := &runner.Runner{
		RunString: c.RacPath,
		Params:    fullArgs,
	}

	// Выполняем команду с таймаутом через контекст
	done := make(chan struct{})
	var output []byte
	var err error

	go func() {
		defer close(done)
		output, err = r.RunCommand(cmdCtx, c.Logger)
		// Если есть консольный вывод, используем его
		if len(r.ConsoleOut) > 0 {
			output = r.ConsoleOut
		}
	}()

	// Ожидаем завершения команды или таймаута
	select {
	case <-done:
		// Команда завершена
	case <-cmdCtx.Done():
		// Таймаут или отмена контекста
		return "", fmt.Errorf("RAC command timeout or cancelled: %w", cmdCtx.Err())
	}

	if err != nil {
		// Конвертируем вывод в UTF-8 для корректного отображения
		convertedOutput, convErr := convertOutputToUTF8(output)
		if convErr != nil {
			c.Logger.Warn("Failed to convert error output to UTF-8", "error", convErr)
		}
		return "", fmt.Errorf("RAC command failed: %w, output: %s", err, convertedOutput)
	}

	// Конвертируем успешный вывод в UTF-8
	convertedOutput, convErr := convertOutputToUTF8(output)
	if convErr != nil {
		c.Logger.Warn("Failed to convert output to UTF-8", "error", convErr)
	}
	return convertedOutput, nil
}

// checkServerConnection проверяет доступность RAC сервера
func (c *Client) checkServerConnection(ctx context.Context) error {
	serverAddress := fmt.Sprintf("%s:%d", c.Server, c.Port)
	c.Logger.Debug("Checking RAC server connection", "address", serverAddress)

	// Создаем контекст с таймаутом для проверки подключения
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(connCtx, "tcp", serverAddress)
	if err != nil {
		c.Logger.Error("RAC server connection check failed",
			"address", serverAddress,
			"error", err)
		return fmt.Errorf("cannot connect to RAC server %s: %w", serverAddress, err)
	}
	if err := conn.Close(); err != nil {
		c.Logger.Warn("Failed to close RAC connection", "error", err)
	}
	c.Logger.Debug("RAC server connection check successful", "address", serverAddress)
	return nil
}

// GetClusterUUID получает UUID кластера 1C.
// Выполняет команду получения списка кластеров и извлекает
// UUID первого найденного кластера.
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - string: UUID кластера
//   - error: ошибка получения UUID или nil при успехе
func (c *Client) GetClusterUUID(ctx context.Context) (string, error) {
	c.Logger.Debug("Getting cluster UUID",
		"server", c.Server,
		"port", c.Port,
		"racPath", c.RacPath)

	// Проверяем доступность RAC сервера перед выполнением команды
	if err := c.checkServerConnection(ctx); err != nil {
		return "", fmt.Errorf("RAC server connectivity check failed: %w", err)
	}

	output, err := c.ExecuteCommand(ctx, "cluster", "list")
	if err != nil {
		c.Logger.Error("Failed to execute cluster list command",
			"error", err,
			"server", c.Server,
			"port", c.Port,
			"racPath", c.RacPath)
		return "", fmt.Errorf("failed to get cluster list: %w", err)
	}

	c.Logger.Debug("Cluster list command output", "output", output)

	// Ищем UUID кластера в выводе
	uuidRegex := regexp.MustCompile(`cluster\s*:\s*([a-f0-9-]{36})`)
	matches := uuidRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		c.Logger.Error("Cluster UUID not found in RAC output",
			"output", output,
			"outputLength", len(output))
		return "", fmt.Errorf("cluster UUID not found in output: %s", output)
	}

	c.Logger.Debug("Found cluster UUID", "uuid", matches[1])
	return matches[1], nil
}

// GetInfobaseUUID получает UUID информационной базы по её имени.
// Выполняет поиск информационной базы в указанном кластере
// и возвращает её уникальный идентификатор.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: UUID кластера для поиска
//   - infobaseName: имя информационной базы
//
// Возвращает:
//   - string: UUID информационной базы
//   - error: ошибка получения UUID или nil при успехе
func (c *Client) GetInfobaseUUID(ctx context.Context, clusterUUID, infobaseName string) (string, error) {
	// Формируем аргументы команды
	args := []string{"infobase", "summary", "list", "--cluster=" + clusterUUID}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	output, err := c.ExecuteCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get infobase list: %w", err)
	}

	// Ищем информационную базу по имени
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "name") && strings.Contains(line, infobaseName) {
			// Ищем UUID в предыдущих строках
			for j := i - 1; j >= 0; j-- {
				if strings.Contains(lines[j], "infobase") {
					uuidRegex := regexp.MustCompile(`infobase\s*:\s*([a-f0-9-]{36})`)
					matches := uuidRegex.FindStringSubmatch(lines[j])
					if len(matches) >= 2 {
						return matches[1], nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("infobase UUID not found for name: %s", infobaseName)
}

// IsValidUUID проверяет валидность UUID строки.
// Проверяет соответствие строки стандартному формату UUID
// (8-4-4-4-12 шестнадцатеричных символов).
// Параметры:
//   - uuid: строка для проверки
//
// Возвращает:
//   - bool: true если строка является валидным UUID, иначе false
func IsValidUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	return uuidRegex.MatchString(uuid)
}
