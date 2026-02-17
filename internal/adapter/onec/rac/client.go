package rac

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// Коды ошибок RAC клиента.
const (
	ErrRACExec     = "RAC.EXEC_FAILED"     // Ошибка запуска RAC-процесса
	ErrRACTimeout  = "RAC.TIMEOUT"         // Timeout при выполнении команды
	ErrRACParse    = "RAC.PARSE_FAILED"    // Ошибка парсинга вывода RAC
	ErrRACNotFound = "RAC.NOT_FOUND"       // Объект не найден (cluster, infobase)
	ErrRACSession  = "RAC.SESSION_FAILED"  // Ошибка операции с сессиями
	ErrRACVerify   = "RAC.VERIFY_FAILED"   // Несоответствие ожидаемого и фактического состояния
)

// ClientOptions — параметры для создания RAC клиента.
type ClientOptions struct {
	// RACPath — путь к исполняемому файлу rac
	RACPath string
	// Server — адрес сервера 1C
	Server string
	// Port — порт RAC (по умолчанию "1545")
	Port string
	// Timeout — таймаут выполнения команд (по умолчанию 30s)
	Timeout time.Duration
	// ClusterUser — администратор кластера (опционально)
	ClusterUser string
	// ClusterPass — пароль администратора кластера (опционально)
	ClusterPass string
	// InfobaseUser — пользователь информационной базы (опционально)
	InfobaseUser string
	// InfobasePass — пароль пользователя информационной базы (опционально)
	InfobasePass string
	// Logger — логгер (если nil — slog.Default())
	Logger *slog.Logger
}

// racClient — реализация интерфейса Client для работы с RAC CLI.
type racClient struct {
	racPath      string
	server       string
	port         string
	timeout      time.Duration
	clusterUser  string
	clusterPass  string
	infobaseUser string
	infobasePass string
	logger       *slog.Logger
}

// Compile-time проверка интерфейса.
var _ Client = (*racClient)(nil)

// NewClient создаёт новый RAC клиент с валидацией параметров.
func NewClient(opts ClientOptions) (Client, error) {
	if opts.RACPath == "" {
		return nil, apperrors.NewAppError(ErrRACExec, "путь к rac не указан", nil)
	}
	// Проверка существования файла RAC для раннего обнаружения ошибок конфигурации
	if _, err := os.Stat(opts.RACPath); err != nil {
		if os.IsNotExist(err) {
			return nil, apperrors.NewAppError(ErrRACExec,
				fmt.Sprintf("исполняемый файл rac не найден: %s", opts.RACPath), err)
		}
		return nil, apperrors.NewAppError(ErrRACExec,
			fmt.Sprintf("ошибка доступа к rac: %s", opts.RACPath), err)
	}
	if opts.Server == "" {
		return nil, apperrors.NewAppError(ErrRACExec, "адрес сервера не указан", nil)
	}
	if opts.Port == "" {
		opts.Port = "1545"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &racClient{
		racPath:      opts.RACPath,
		server:       opts.Server,
		port:         opts.Port,
		timeout:      opts.Timeout,
		clusterUser:  opts.ClusterUser,
		clusterPass:  opts.ClusterPass,
		infobaseUser: opts.InfobaseUser,
		infobasePass: opts.InfobasePass,
		logger:       opts.Logger,
	}, nil
}

// executeRAC запускает RAC как subprocess с таймаутом.
func (c *racClient) executeRAC(ctx context.Context, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	serverAddr := c.server + ":" + c.port
	fullArgs := make([]string, 0, len(args)+1)
	fullArgs = append(fullArgs, serverAddr)
	fullArgs = append(fullArgs, args...)

	c.logger.Debug("Выполнение RAC команды",
		"racPath", c.racPath,
		"args", sanitizeArgs(fullArgs),
	)

	cmd := exec.CommandContext(ctx, c.racPath, fullArgs...) //nolint:gosec // аргументы формируются программно, не из пользовательского ввода
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", apperrors.NewAppError(ErrRACTimeout,
				fmt.Sprintf("таймаут выполнения команды (%s)", c.timeout), err)
		}
		if ctx.Err() == context.Canceled {
			return "", apperrors.NewAppError(ErrRACExec, "выполнение отменено", err)
		}
		// Извлекаем stderr из ExitError для диагностики (stdout и stderr разделены)
		// Применяем sanitizeString для предотвращения утечки credentials в сообщениях об ошибках
		errMsg := "ошибка выполнения RAC"
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := sanitizeString(strings.TrimSpace(string(exitErr.Stderr)))
			if stderr != "" {
				errMsg = fmt.Sprintf("ошибка выполнения RAC: %s", stderr)
			}
		}
		return "", apperrors.NewAppError(ErrRACExec, errMsg, err)
	}

	return string(output), nil
}
