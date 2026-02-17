// Package runner предоставляет функциональность для выполнения команд
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxConsoleOut = 2048

// Runner структура для выполнения команд и управления их параметрами
type Runner struct {
	RunString   string
	Params      []string
	WorkDir     string
	TmpDir      string
	OutFileName string
	ConsoleOut  []byte
	FileOut     []byte
}

// ClearParams очищает все параметры команды.
// Сбрасывает все настройки Runner к значениям по умолчанию,
// подготавливая его для выполнения новой команды.
func (r *Runner) ClearParams() {
	r.Params = []string{}
}

// RunCommand выполняет команду и возвращает результат.
// Запускает системную команду с заданными параметрами и возвращает
// её вывод или ошибку выполнения.
// Параметры:
//   - ctx: контекст выполнения с возможным таймаутом
//   - l: логгер для записи сообщений
//
// Возвращает:
//   - []byte: вывод команды из файла
//   - error: ошибка выполнения или nil при успехе
func (r *Runner) RunCommand(ctx context.Context, l *slog.Logger) ([]byte, error) {
	var err error
	var errIn error
	var fileOutContent []byte
	var tFile *os.File
	var tOut *os.File
	var lParams []string
	if len(r.Params) > 0 && r.Params[0] == "@" {
		tFile, err = os.CreateTemp(r.TmpDir, "*.par")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp params file: %w", err)
		}
		// defer os.Remove(tFile.Name())
		// l.Debug("Params", "Params", r.Params) // Удалено: выводит пароли до маскировки
		tParams := []string{}
		for i, value := range r.Params {
			if i == 0 {
				continue
			}
			if r.Params[i-1] == "/ConfigurationRepositoryP" {
				lParams = append(lParams, "*****")
			} else {
				// Маскируем пароль после /P в строке (например: /S server /N user /P password)
				maskedValue := maskPasswordInParam(value)
				lParams = append(lParams, maskedValue)
			}
			if len(value) >= 2 && value[0:2] == "/c" {
				tParams = append(tParams, "/c")
				tParams = append(tParams, value[2:])
				continue
			}
			if len(value) >= 4 && value[0:4] == "/Out" {
				tOut, err = os.CreateTemp(r.TmpDir, "*.out")
				if err != nil {
					return nil, fmt.Errorf("failed to create temp output file: %w", err)
				}
				// defer os.Remove(tOut.Name())
				r.OutFileName = tOut.Name()
				value = "/Out " + r.OutFileName
				if errClose := tOut.Close(); errClose != nil {
					l.Warn("Failed to close temp output file", "error", errClose)
				}
			}
			if _, errWrite := tFile.WriteString(" " + value); errWrite != nil {
				l.Warn("Failed to write to temp file", "error", errWrite)
			}
		}

		r.Params = []string{"@", tFile.Name()}
		r.Params = append(r.Params, tParams...)
		if errClose := tFile.Close(); errClose != nil {
			l.Warn("Failed to close temp file", "error", errClose)
		}
	}
	l.Info("Параметры запуска",
		slog.String("Исполняемый файл", r.RunString),
		slog.String("WorkDir", r.WorkDir),
		slog.String("Параметры", fmt.Sprint(r.Params)),
		slog.String("Передаваемые параметры", fmt.Sprint(lParams)),
	)

	// Валидация исполняемого файла
	if r.RunString == "" {
		return nil, errors.New("executable path is empty")
	}

	// Валидация параметров
	for _, param := range r.Params {
		if strings.Contains(param, ";") || strings.Contains(param, "&") || strings.Contains(param, "|") {
			return nil, fmt.Errorf("potentially unsafe parameter detected: %s", param)
		}
	}

	// #nosec G204 - parameters are validated above
	cmd := exec.CommandContext(ctx, r.RunString, r.Params...)

	if len(r.Params) > 0 && r.Params[0] == "@" {
		cmd.Env = appendEnviron("DISPLAY=:99", "XAUTHORITY=/tmp/.Xauth99")
		// l.Debug("Переменные окружения", slog.String("cmd.Env", fmt.Sprint(cmd.Env)))
	}
	cmd.Dir = r.WorkDir
	r.ConsoleOut, err = cmd.Output()
	if exists(r.OutFileName) {
		fileOutContent, errIn = os.ReadFile(r.OutFileName)
		if errIn != nil {
			l.Error("Runner",
				slog.String("Ошибка при чтении файла", errIn.Error()),
				slog.String("Файл", r.OutFileName),
			)
		} else {
			r.FileOut = fileOutContent
		}
	}

	if err != nil {
		errText := TrimOut(r.ConsoleOut)
		errText += "\n" + string(fileOutContent)

		l.Error("Runner",
			slog.String("Ошибка при запуске", err.Error()),
			slog.String("Исполняемый файл", r.RunString),
			slog.String("WorkDir", r.WorkDir),
			slog.String("Параметры", fmt.Sprint(r.Params)),
			slog.String("Ошибка при запуске", errText),
		)
		// DEBUG
		// os.Exit(111)
	}
	l.Debug("Runner",
		slog.String("Вывод консоли", TrimOut(r.ConsoleOut)),
	)
	// err = cmd.Run()
	// if err != nil && err.Error() != "exec: already started" {
	// 	l.Error("Неопознанная ошибка",
	// 		slog.String("Ошибка при запуске", err.Error()),
	// 	)
	// } else if err != nil && err.Error() == "exec: already started" {
	// 	err = nil
	// }
	r.Params = []string{}
	return r.FileOut, err
}

func appendEnviron(kv ...string) []string {
	env := os.Environ()
	for _, newVar := range kv {
		found := false
		eqIndex := strings.Index(newVar, "=")
		if eqIndex == -1 {
			continue // Пропускаем некорректные переменные окружения
		}
		key := newVar[:eqIndex]

		// Ищем существующую переменную
		for i, v := range env {
			if strings.HasPrefix(v, key+"=") {
				env[i] = newVar // заменяем значение
				found = true
				break
			}
		}

		// Если не нашли - добавляем
		if !found {
			env = append(env, newVar)
		}
	}
	return env
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

// TrimOut обрезает вывод команды, удаляя лишние символы.
// Ограничивает размер вывода до максимального значения,
// показывая начало и конец при превышении лимита.
// Параметры:
//   - b: байтовый массив вывода для обработки
//
// Возвращает:
//   - string: обработанная строка с ограниченным размером
func TrimOut(b []byte) string {
	if len(b) < maxConsoleOut {
		return string(b)
	}
	return string(b[:1020]) + "\n********\n" + string(b[len(b)-1020:])
}

// DisplayConfig отображает конфигурацию дисплея и управляет Xvfb процессами.
// Обрабатывает настройки виртуального дисплея, включая запуск Xvfb сервера
// и управление файлами блокировки для корректной работы графических приложений.
func DisplayConfig(l *slog.Logger) error {
	// Остановка существующих процессов Xvfb
	// l.Debug("Config", slog.String("action", "Остановка существующих процессов Xvfb"))
	// #nosec G204 - all arguments are hardcoded constants
	killCmd := exec.Command("pkill", "-f", "Xvfb")
	_ = killCmd.Run() // Игнорируем ошибки

	// Очистка lock файлов
	// l.Debug("Config", slog.String("action", "Очистка lock файлов"))
	// #nosec G204 - all arguments are hardcoded constants
	removeCmd := exec.Command("rm", "-f", "/tmp/.X99-lock", "/tmp/.X*-lock")
	_ = removeCmd.Run() // Игнорируем ошибки

	// Запуск виртуального дисплея
	// l.Debug("Config", slog.String("action", "Запуск виртуального дисплея :99"))
	// #nosec G204 - all arguments are hardcoded constants
	startCmd := exec.Command("Xvfb", ":99", "-screen", "0", "1920x1080x24", "-ac", "+extension", "GLX", "+render", "-noreset")
	startCmd.Stdout = nil
	startCmd.Stderr = nil

	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Xvfb: %w", err)
	}

	xvfbPID := startCmd.Process.Pid
	// l.Debug("Config", "Xvfb запущен", slog.Int("PID", xvfbPID))

	// Ожидание запуска Xvfb
	time.Sleep(3 * time.Second)

	// Проверка запуска Xvfb
	if xvfbPID <= 0 {
		return fmt.Errorf("некорректный PID процесса Xvfb: %d", xvfbPID)
	}
	pidStr := strconv.Itoa(xvfbPID)
	// #nosec G204 - pidStr is safe (string representation of int)
	checkCmd := exec.Command("ps", "-p", pidStr)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("xvfb не запустился (PID: %d)", xvfbPID)
	}
	// l.Debug("Config", "✓ Xvfb успешно запущен", slog.Int("PID", xvfbPID))

	// Установка переменных окружения
	// l.Debug("Config", slog.String("action", "Установка переменных окружения"))
	if err := os.Setenv("DISPLAY", ":99"); err != nil {
		return fmt.Errorf("failed to set DISPLAY=:99: %w", err)
	}
	if err := os.Setenv("XAUTHORITY", "/tmp/.Xauth99"); err != nil {
		return fmt.Errorf("failed to set XAUTHORITY: %w", err)
	}

	// Создание файла авторизации X11
	// l.Debug("Config", slog.String("action", "Создание файла авторизации X11"))
	// #nosec G204 - all arguments are hardcoded constants
	touchCmd := exec.Command("touch", "/tmp/.Xauth99")
	if err := touchCmd.Run(); err != nil {
		l.Warn("failed to create Xauth file", slog.String("error", err.Error()))
	}

	// Генерация случайного ключа для xauth
	// #nosec G204 - all arguments are hardcoded constants
	randCmd := exec.Command("xxd", "-l", "16", "-p", "/dev/urandom")
	randOutput, err := randCmd.Output()
	if err == nil {
		randKey := strings.TrimSpace(string(randOutput))
		// Валидация сгенерированного ключа (должен быть hex строкой длиной 32)
		if len(randKey) == 32 {
			// #nosec G204 - randKey is validated (32 char hex string)
			xauthCmd := exec.Command("xauth", "add", ":99", ".", randKey)
			_ = xauthCmd.Run() // Игнорируем ошибки
		}
	}

	// Проверка подключения к дисплею
	// l.Debug("Config", slog.String("action", "Проверка подключения к дисплею"))
	// #nosec G204 - all arguments are hardcoded constants
	testCmd := exec.Command("xdpyinfo", "-display", ":99")
	testCmd.Stdout = nil
	testCmd.Stderr = nil
	if err := testCmd.Run(); err != nil {
		// l.Debug("Config", slog.String("action", "Не удается подключиться к дисплею :99, попытка исправления"))

		// Альтернативный запуск с другими параметрами
		// #nosec G204 - all arguments are hardcoded constants
		killCmd2 := exec.Command("pkill", "-f", "Xvfb")
		if err := killCmd2.Run(); err != nil {
			l.Debug("pkill Xvfb retry (may not be running)", slog.String("error", err.Error()))
		}
		time.Sleep(2 * time.Second)

		// #nosec G204 - all arguments are hardcoded constants
		startCmd2 := exec.Command("Xvfb", ":99", "-screen", "0", "1920x1080x24", "-dpi", "96", "-ac", "+extension", "RANDR")
		startCmd2.Stdout = nil
		startCmd2.Stderr = nil
		if err := startCmd2.Start(); err != nil {
			return fmt.Errorf("failed to restart Xvfb: %w", err)
		}
		time.Sleep(3 * time.Second)

		// #nosec G204 - all arguments are hardcoded constants
		testCmd2 := exec.Command("xdpyinfo", "-display", ":99")
		testCmd2.Stdout = nil
		testCmd2.Stderr = nil
		if err := testCmd2.Run(); err != nil {
			return fmt.Errorf("все еще не удается подключиться к дисплею :99")
		}
		l.Debug("Config", slog.String("result", "✓ Подключение к дисплею :99 успешно (после исправления)"))
	} else {
		l.Debug("Config", slog.String("result", "✓ Подключение к дисплею :99 успешно"))
	}

	return nil
}

// maskPasswordInParam маскирует пароль после /P в строке параметров.
// Обрабатывает паттерны вида: /S server /N user /P password
// Возвращает строку с замаскированным паролем: /S server /N user /P *****
func maskPasswordInParam(value string) string {
	// Регулярное выражение для поиска /P с последующим значением
	// Паттерн: пробел или начало строки, /P, пробел, непробельные символы
	re := regexp.MustCompile(`(\s/P\s)(\S+)`)
	return re.ReplaceAllString(value, "${1}*****")
}
