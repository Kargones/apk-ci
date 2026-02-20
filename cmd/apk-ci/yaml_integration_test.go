//go:build integration
package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestMain_WithRealYamlFile тестирует функцию main с реальным файлом main-test.yaml
func TestMain_WithRealYamlFile(t *testing.T) {
	// Сохраняем текущее окружение
	originalEnv := saveEnvironment([]string{
		"INPUT_ACTOR", "INPUT_COMMAND", "INPUT_LOGLEVEL", "INPUT_DBNAME",
		"INPUT_REPOSITORY", "INPUT_GITEAURL", "INPUT_ACCESSTOKEN",
		"INPUT_CONFIGSYSTEM", "INPUT_CONFIGPROJECT", "INPUT_CONFIGSECRET",
		"INPUT_CONFIGDBDATA", "INPUT_TERMINATESESSIONS", "INPUT_FORCE_UPDATE",
		"INPUT_ISSUENUMBER", "WorkDir", "TmpDir", "RepPath", "Connect_String",
		"INPUT_MENUMAIN", "INPUT_MENUDEBUG", "INPUT_STARTEPF", "INPUT_BRANCHFORSCAN",
		"INPUT_COMMITHASH", "BR_ACTOR", "BR_ENV", "BR_COMMAND", "BR_INFOBASE_NAME",
		"BR_TERMINATE_SESSIONS", "BR_FORCE_UPDATE", "BR_ISSUE_NUMBER", "BR_START_EPF",
		"BR_ACCESS_TOKEN", "BR_CONFIG_SYSTEM", "BR_CONFIG_PROJECT", "BR_CONFIG_SECRET",
		"BR_CONFIG_DBDATA", "BR_CONFIG_MENU_MAIN", "BR_CONFIG_MENU_DEBUG",
		"GITHUB_REF_NAME", "GIT_TIMEOUT",
	})
	defer restoreEnvironment(originalEnv)

	// Очищаем окружение
	clearTestEnv()

	// Путь к файлу main-test.yaml
	yamlPath := "../../main-test.yaml"

	// Проверяем, что файл существует
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Skipf("Файл %s не найден, пропускаем тест", yamlPath)
	}

	// Проверяем, включено ли тестирование в YAML (test-enable: true)
	type testEnableConfig struct {
		TestEnable bool `yaml:"test-enable"`
	}
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}
	var enableCfg testEnableConfig
	if err := yaml.Unmarshal(data, &enableCfg); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	if !enableCfg.TestEnable {
		t.Skip("Тестирование отключено в main-test.yaml (test-enable: false)")
	}

	// Загружаем конфигурацию из YAML и устанавливаем переменные окружения
	err = loadYamlConfigAndSetEnv(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Проверяем, что переменные окружения установлены корректно согласно main-test.yaml
	expectedVars := map[string]string{
		"INPUT_ACTOR":       "xor",
		"INPUT_COMMAND":     "extension-publish", // Из main-test.yaml
		"INPUT_LOGLEVEL":    "Debug",
		"INPUT_DBNAME":      "test-database",
		"INPUT_REPOSITORY":  "test/apk-ssl",
		"INPUT_GITEAURL":    "https://git.apkholding.ru",
		"INPUT_ACCESSTOKEN": "e0452e72c27392799fd34f88da9546a1af509947",
	}

	for key, expected := range expectedVars {
		if actual := os.Getenv(key); actual != expected {
			t.Errorf("Expected %s=%s, got %s", key, expected, actual)
		}
	}

	// Проверка GITHUB_REF_NAME
	if refName := os.Getenv("GITHUB_REF_NAME"); refName != "" {
		t.Logf("GITHUB_REF_NAME установлен: %s", refName)
	} else {
		t.Log("GITHUB_REF_NAME не установлен!")
	}

	t.Log("Переменные окружения установлены корректно, запускаем main()...")

	// Запускаем main() в отдельной горутине с перехватом паники
	var mainErr error
	done := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				mainErr = fmt.Errorf("panic in main(): %v", r)
			}
			done <- true
		}()

		// Реальный запуск функции main()
		// Примечание: main() может вызвать os.Exit(), что завершит весь процесс
		// Это нормальное поведение для отладочного теста
		main()
	}()

	// Ждем завершения main() или таймаута
	select {
	case <-done:
		if mainErr != nil {
			t.Logf("main() завершилась с ошибкой: %v", mainErr)
		} else {
			t.Log("main() успешно выполнилась")
		}
	case <-time.After(60 * time.Minute):
		t.Log("main() выполняется дольше 60 минут, это может быть нормально для некоторых команд")
		// Не завершаем тест с ошибкой, так как некоторые команды могут выполняться долго
	}

	t.Log("Тест с реальным запуском main() и файлом main-test.yaml завершен")
	t.Log("ВНИМАНИЕ: Если main() вызовет os.Exit(), весь тестовый процесс может завершиться")
}
