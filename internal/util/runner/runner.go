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
const maskedValue = "*****"

func (r *Runner) ClearParams() {
	r.Params = []string{}
}

// processParamValue обрабатывает одно значение параметра:
// разделяет /c-параметры, создаёт временный файл для /Out-параметров.
func (r *Runner) processParamValue(value string, l *slog.Logger) (fileValue string, cmdParams []string, err error) {
	if len(value) >= 2 && value[0:2] == "/c" {
		return "", []string{"/c", value[2:]}, nil
	}
	if len(value) >= 4 && value[0:4] == "/Out" {
		tOut, err := os.CreateTemp(r.TmpDir, "*.out")
		if err != nil {
			return "", nil, fmt.Errorf("failed to create temp output file: %w", err)
		}
		r.OutFileName = tOut.Name()
		if errClose := tOut.Close(); errClose != nil {
			l.Warn("Failed to close temp output file", "error", errClose)
		}
		return "/Out " + r.OutFileName, nil, nil
	}
	return value, nil, nil
}

// buildMaskedParam возвращает маскированную версию параметра для логирования.
func buildMaskedParam(params []string, i int, value string) string {
	if params[i-1] == "/ConfigurationRepositoryP" {
		return maskedValue
	}
	return maskPasswordInParam(value)
}

// prepareAtParams подготавливает параметры в режиме "@":
// создаёт временный файл с параметрами и возвращает маскированные параметры для логирования.
func (r *Runner) prepareAtParams(l *slog.Logger) (lParams []string, err error) {
	tFile, err := os.CreateTemp(r.TmpDir, "*.par")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp params file: %w", err)
	}

	var tParams []string
	for i, value := range r.Params {
		if i == 0 {
			continue
		}
		lParams = append(lParams, buildMaskedParam(r.Params, i, value))

		fileValue, cmdExtra, err := r.processParamValue(value, l)
		if err != nil {
			return nil, err
		}
		if cmdExtra != nil {
			tParams = append(tParams, cmdExtra...)
			continue
		}
		if _, errWrite := tFile.WriteString(" " + fileValue); errWrite != nil {
			l.Warn("Failed to write to temp file", "error", errWrite)
		}
	}

	r.Params = append([]string{"@", tFile.Name()}, tParams...)
	if errClose := tFile.Close(); errClose != nil {
		l.Warn("Failed to close temp file", "error", errClose)
	}
	return lParams, nil
}

// validateParams проверяет корректность исполняемого файла и параметров.
func (r *Runner) validateParams() error {
	if r.RunString == "" {
		return errors.New("executable path is empty")
	}
	for _, param := range r.Params {
		if strings.Contains(param, ";") || strings.Contains(param, "&") || strings.Contains(param, "|") {
			return fmt.Errorf("potentially unsafe parameter detected: %s", param)
		}
	}
	return nil
}

// readOutputFile читает файл вывода и сохраняет результат в r.FileOut.
func (r *Runner) readOutputFile(l *slog.Logger) []byte {
	if !exists(r.OutFileName) {
		return nil
	}
	fileOutContent, errIn := os.ReadFile(r.OutFileName)
	if errIn != nil {
		l.Error("Runner",
			slog.String("Ошибка при чтении файла", errIn.Error()),
			slog.String("Файл", r.OutFileName),
		)
		return nil
	}
	r.FileOut = fileOutContent
	return fileOutContent
}

// RunCommand выполняет команду и возвращает результат.
func (r *Runner) RunCommand(ctx context.Context, l *slog.Logger) ([]byte, error) {
	var lParams []string

	if len(r.Params) > 0 && r.Params[0] == "@" {
		var err error
		lParams, err = r.prepareAtParams(l)
		if err != nil {
			return nil, err
		}
	}

	l.Info("Параметры запуска",
		slog.String("Исполняемый файл", r.RunString),
		slog.String("WorkDir", r.WorkDir),
		slog.String("Параметры", fmt.Sprint(r.Params)),
		slog.String("Передаваемые параметры", fmt.Sprint(lParams)),
	)

	if err := r.validateParams(); err != nil {
		return nil, err
	}

	// #nosec G204 - parameters are validated above
	cmd := exec.CommandContext(ctx, r.RunString, r.Params...)

	if len(r.Params) > 0 && r.Params[0] == "@" {
		cmd.Env = appendEnviron("DISPLAY=:99", "XAUTHORITY=/tmp/.Xauth99")
	}
	cmd.Dir = r.WorkDir

	var err error
	r.ConsoleOut, err = cmd.Output()

	fileOutContent := r.readOutputFile(l)

	if err != nil {
		errText := TrimOut(r.ConsoleOut) + "\n" + string(fileOutContent)
		l.Error("Runner",
			slog.String("Ошибка при запуске", err.Error()),
			slog.String("Исполняемый файл", r.RunString),
			slog.String("WorkDir", r.WorkDir),
			slog.String("Параметры", fmt.Sprint(r.Params)),
			slog.String("Ошибка при запуске", errText),
		)
	}
	l.Debug("Runner",
		slog.String("Вывод консоли", TrimOut(r.ConsoleOut)),
	)

	r.Params = []string{}
	return r.FileOut, err
}

func appendEnviron(kv ...string) []string {
	env := os.Environ()
	for _, newVar := range kv {
		eqIndex := strings.Index(newVar, "=")
		if eqIndex == -1 {
			continue
		}
		key := newVar[:eqIndex]
		found := false
		for i, v := range env {
			if strings.HasPrefix(v, key+"=") {
				env[i] = newVar
				found = true
				break
			}
		}
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

// TrimOut обрезает вывод команды.
func TrimOut(b []byte) string {
	if len(b) < maxConsoleOut {
		return string(b)
	}
	return string(b[:1020]) + "\n********\n" + string(b[len(b)-1020:])
}

// DisplayConfig отображает конфигурацию дисплея и управляет Xvfb процессами.
func DisplayConfig(l *slog.Logger) error {
	// #nosec G204 - all arguments are hardcoded constants
	killCmd := exec.Command("pkill", "-f", "Xvfb")
	_ = killCmd.Run()

	// #nosec G204 - all arguments are hardcoded constants
	removeCmd := exec.Command("rm", "-f", "/tmp/.X99-lock", "/tmp/.X*-lock")
	_ = removeCmd.Run()

	// #nosec G204 - all arguments are hardcoded constants
	startCmd := exec.Command("Xvfb", ":99", "-screen", "0", "1920x1080x24", "-ac", "+extension", "GLX", "+render", "-noreset")
	startCmd.Stdout = nil
	startCmd.Stderr = nil

	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Xvfb: %w", err)
	}

	xvfbPID := startCmd.Process.Pid
	time.Sleep(3 * time.Second)

	if xvfbPID <= 0 {
		return fmt.Errorf("некорректный PID процесса Xvfb: %d", xvfbPID)
	}
	pidStr := strconv.Itoa(xvfbPID)
	// #nosec G204 - pidStr is safe (string representation of int)
	checkCmd := exec.Command("ps", "-p", pidStr)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("xvfb не запустился (PID: %d)", xvfbPID)
	}

	if err := os.Setenv("DISPLAY", ":99"); err != nil {
		return fmt.Errorf("failed to set DISPLAY=:99: %w", err)
	}
	if err := os.Setenv("XAUTHORITY", "/tmp/.Xauth99"); err != nil {
		return fmt.Errorf("failed to set XAUTHORITY: %w", err)
	}

	// #nosec G204 - all arguments are hardcoded constants
	touchCmd := exec.Command("touch", "/tmp/.Xauth99")
	if err := touchCmd.Run(); err != nil {
		l.Warn("failed to create Xauth file", slog.String("error", err.Error()))
	}

	// #nosec G204 - all arguments are hardcoded constants
	randCmd := exec.Command("xxd", "-l", "16", "-p", "/dev/urandom")
	randOutput, err := randCmd.Output()
	if err == nil {
		randKey := strings.TrimSpace(string(randOutput))
		if len(randKey) == 32 {
			// #nosec G204 - randKey is validated (32 char hex string)
			xauthCmd := exec.Command("xauth", "add", ":99", ".", randKey)
			_ = xauthCmd.Run()
		}
	}

	// #nosec G204 - all arguments are hardcoded constants
	testCmd := exec.Command("xdpyinfo", "-display", ":99")
	testCmd.Stdout = nil
	testCmd.Stderr = nil
	if err := testCmd.Run(); err != nil {
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
func maskPasswordInParam(value string) string {
	re := regexp.MustCompile(`(\s/P\s)(\S+)`)
	return re.ReplaceAllString(value, "${1}*****")
}
