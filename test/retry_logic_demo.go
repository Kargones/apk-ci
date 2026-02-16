package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

func main() {
	// Настройка логирования
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fmt.Println("=== Тестирование логики перезапуска сканера ===")

	// Создание конфигурации сканера
	cfg := &config.ScannerConfig{
		WorkDir: "/root/r/apk-ci/test",
		Timeout: 30 * time.Second,
	}

	// Создание экземпляра сканера
	scanner := sonarqube.NewSonarScannerEntity(cfg, logger)

	// Настройка базовых свойств
	scanner.SetProperty("sonar.projectKey", "test-retry-logic")
	scanner.SetProperty("sonar.projectName", "Test Retry Logic")
	scanner.SetProperty("sonar.projectVersion", "1.0")
	scanner.SetProperty("sonar.sources", ".")
	scanner.SetProperty("sonar.host.url", "http://localhost:9000")
	scanner.SetProperty("sonar.login", "test-token")

	fmt.Printf("Максимальное количество попыток: %d\n", sonarqube.MaxScanRetries)
	fmt.Printf("Рабочая директория: %s\n", cfg.WorkDir)

	// Проверка наличия проблемных файлов
	problematicFiles := []string{
		filepath.Join(cfg.WorkDir, "problematic_test.bsl"),
		filepath.Join(cfg.WorkDir, "another_problematic.bsl"),
	}

	fmt.Println("\nПроверка наличия проблемных файлов:")
	for _, file := range problematicFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("✓ Файл существует: %s\n", file)
		} else {
			fmt.Printf("✗ Файл не найден: %s\n", file)
		}
	}

	// Тестирование методов управления исключениями
	fmt.Println("\n=== Тестирование методов управления исключениями ===")

	// Тест ExtractProblematicBSLFiles
	testErrorOutput := `
ERROR: Error during SonarQube Scanner execution
org.sonar.api.utils.SonarException: Unable to parse file: /root/r/apk-ci/test/problematic_test.bsl
	at org.sonar.plugins.bsl.BSLTokenizer.tokenize(BSLTokenizer.java:45)
	at org.sonar.plugins.bsl.BSLSensor.execute(BSLSensor.java:78)
Caused by: BSL tokenization error in file: /root/r/apk-ci/test/another_problematic.bsl
	at line 5: Unexpected token
`

	fmt.Println("Тестирование ExtractProblematicBSLFiles:")
	extractedFiles := scanner.ExtractProblematicBSLFiles(testErrorOutput)
	fmt.Printf("Извлеченные проблемные файлы: %v\n", extractedFiles)

	// Тест AddFilesToExclusions
	fmt.Println("\nТестирование AddFilesToExclusions:")
	scanner.AddFilesToExclusions(extractedFiles)
	excludedFiles := scanner.GetExcludedFiles()
	fmt.Printf("Исключенные файлы: %v\n", excludedFiles)
	fmt.Printf("Свойство sonar.exclusions: %s\n", scanner.GetProperty("sonar.exclusions"))

	// Тест ClearExclusions
	fmt.Println("\nТестирование ClearExclusions:")
	scanner.ClearExclusions()
	fmt.Printf("Исключенные файлы после очистки: %v\n", scanner.GetExcludedFiles())
	fmt.Printf("Свойство sonar.exclusions после очистки: %s\n", scanner.GetProperty("sonar.exclusions"))

	// Симуляция выполнения сканирования с логикой перезапуска
	fmt.Println("\n=== Симуляция выполнения сканирования ===")
	fmt.Println("ВНИМАНИЕ: Это демонстрация логики, реальное сканирование может не выполниться")
	fmt.Println("из-за отсутствия настроенного SonarQube сервера")

	// Создание контекста с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Попытка выполнения сканирования
	fmt.Println("\nЗапуск сканирования с логикой перезапуска...")
	startTime := time.Now()

	result, err := scanner.Execute(ctx)
	duration := time.Since(startTime)

	fmt.Printf("\nРезультат выполнения (время: %v):\n", duration)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		if result != nil {
			fmt.Printf("Количество ошибок в результате: %d\n", len(result.Errors))
			if len(result.Errors) > 0 {
				fmt.Println("Первые несколько ошибок:")
				for i, errMsg := range result.Errors {
					if i >= 3 {
						fmt.Printf("... и еще %d ошибок\n", len(result.Errors)-3)
						break
					}
					fmt.Printf("  %d. %s\n", i+1, errMsg)
				}
			}
		}
	} else {
		fmt.Println("Сканирование завершено успешно!")
		if result != nil {
			fmt.Printf("Успех: %v\n", result.Success)
			fmt.Printf("Длительность: %v\n", result.Duration)
			fmt.Printf("ID анализа: %s\n", result.AnalysisID)
		}
	}

	// Проверка финального состояния исключений
	fmt.Println("\n=== Финальное состояние ===")
	finalExcluded := scanner.GetExcludedFiles()
	fmt.Printf("Финальный список исключенных файлов: %v\n", finalExcluded)
	fmt.Printf("Финальное свойство sonar.exclusions: %s\n", scanner.GetProperty("sonar.exclusions"))

	fmt.Println("\n=== Тестирование завершено ===")
}