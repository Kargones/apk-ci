package createtempdbhandler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/progress"
)

// generateDbPath генерирует уникальный путь для временной БД.
// Валидирует что путь находится в разрешённой директории.
// Использует наносекундную точность и случайный суффикс для предотвращения коллизий.
func (h *CreateTempDbHandler) generateDbPath(cfg *config.Config) (string, error) {
	// Используем TmpDir из конфигурации или дефолтную директорию
	baseDir := constants.TempDir
	if cfg.TmpDir != "" {
		baseDir = cfg.TmpDir
	}

	// H2 fix: валидация что baseDir — безопасный путь (должен быть в /tmp или TempDir)
	cleanPath := filepath.Clean(baseDir)

	// Разрешаем symlinks для предотвращения path traversal через symlinks
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		// Путь существует и symlinks разрешены — используем разрешённый путь для проверки
		cleanPath = resolvedPath
	}
	// Если ошибка (путь не существует) — проверяем оригинальный cleanPath

	allowedPrefixes := []string{"/tmp", constants.TempDir, os.TempDir()}
	isAllowed := false
	for _, prefix := range allowedPrefixes {
		resolvedPrefix := filepath.Clean(prefix)
		// Также пытаемся разрешить symlinks в prefix
		if rp, err := filepath.EvalSymlinks(resolvedPrefix); err == nil {
			resolvedPrefix = rp
		}
		if strings.HasPrefix(cleanPath, resolvedPrefix) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return "", fmt.Errorf("путь %s не находится в разрешённой директории (допустимы: /tmp, %s)", baseDir, constants.TempDir)
	}

	// Наносекундная точность + случайный суффикс для гарантии уникальности
	// Формат: temp_db_YYYYMMDD_HHMMSS_NNNNNNNNN_RRRRRRRR
	now := time.Now()
	randomSuffix := make([]byte, 4)
	if _, err := rand.Read(randomSuffix); err != nil {
		// Fallback на наносекунды если crypto/rand недоступен
		randomSuffix = []byte(fmt.Sprintf("%04d", now.Nanosecond()%10000))
	}
	timestamp := now.Format("20060102_150405") + fmt.Sprintf("_%09d_%s", now.Nanosecond(), hex.EncodeToString(randomSuffix))
	dbPath := filepath.Join(baseDir, fmt.Sprintf("temp_db_%s", timestamp))

	// H-3 fix: проверка максимальной длины пути
	if len(dbPath) > maxPathLength {
		return "", fmt.Errorf("путь слишком длинный (%d символов, максимум %d): %s", len(dbPath), maxPathLength, dbPath)
	}

	// M-6 fix: проверка что путь ещё не существует (защита от коллизий)
	if _, err := os.Stat(dbPath); err == nil {
		return "", fmt.Errorf("путь уже существует (коллизия): %s", dbPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("ошибка проверки пути: %w", err)
	}

	return dbPath, nil
}

// parseExtensions парсит список расширений из переменной окружения или конфигурации.
// Приоритет: BR_EXTENSIONS > cfg.AddArray.
// M-5 fix: ограничивает количество расширений до maxExtensions для защиты от DoS.
func (h *CreateTempDbHandler) parseExtensions(cfg *config.Config) []string {
	log := slog.Default()

	// Приоритет: BR_EXTENSIONS > cfg.AddArray
	extEnv := os.Getenv("BR_EXTENSIONS")
	if extEnv != "" {
		// Парсим через запятую и очищаем пробелы
		parts := strings.Split(extEnv, ",")
		var extensions []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				extensions = append(extensions, trimmed)
			}
		}
		// M-5 fix: лимит на количество расширений
		if len(extensions) > maxExtensions {
			log.Warn("Количество расширений превышает лимит, обрезано",
				slog.Int("requested", len(extensions)),
				slog.Int("max", maxExtensions))
			extensions = extensions[:maxExtensions]
		}
		return extensions
	}

	// Fallback на cfg.AddArray
	if len(cfg.AddArray) > 0 {
		log.Debug("BR_EXTENSIONS не задан, используется cfg.AddArray",
			slog.Any("extensions", cfg.AddArray))
		extensions := cfg.AddArray
		// M-5 fix: лимит на количество расширений
		if len(extensions) > maxExtensions {
			log.Warn("Количество расширений из cfg.AddArray превышает лимит, обрезано",
				slog.Int("requested", len(extensions)),
				slog.Int("max", maxExtensions))
			extensions = extensions[:maxExtensions]
		}
		return extensions
	}

	return nil
}

// getTimeout возвращает таймаут для операции создания БД.
func (h *CreateTempDbHandler) getTimeout() time.Duration {
	// Проверяем явный таймаут через BR_TIMEOUT_MIN
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			return time.Duration(timeoutMin) * time.Minute
		}
	}
	return defaultTimeout
}

// getTTLHours возвращает TTL в часах из переменной окружения.
func (h *CreateTempDbHandler) getTTLHours() int {
	ttlStr := os.Getenv("BR_TTL_HOURS")
	if ttlStr == "" {
		return 0
	}
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl < 0 {
		return 0
	}
	return ttl
}

// getOrCreateClient возвращает существующий или создаёт новый клиент.
func (h *CreateTempDbHandler) getOrCreateClient(cfg *config.Config) onec.TempDatabaseCreator {
	if h.dbCreator != nil {
		return h.dbCreator
	}
	return onec.NewTempDbCreator()
}

// writeTTLMetadata записывает метаданные TTL в файл рядом с БД.
func (h *CreateTempDbHandler) writeTTLMetadata(dbPath string, ttlHours int, createdAt time.Time) error {
	metadata := TTLMetadata{
		CreatedAt: createdAt,
		TTLHours:  ttlHours,
		ExpiresAt: createdAt.Add(time.Duration(ttlHours) * time.Hour),
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация TTL metadata: %w", err)
	}

	ttlPath := dbPath + ".ttl"

	// Проверяем существование родительской директории
	parentDir := filepath.Dir(ttlPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if mkdirErr := os.MkdirAll(parentDir, 0755); mkdirErr != nil {
			return fmt.Errorf("создание директории для TTL: %w", mkdirErr)
		}
	}

	// Используем 0600 для метаданных (только владелец может читать/писать)
	if err := os.WriteFile(ttlPath, data, 0600); err != nil {
		return fmt.Errorf("запись TTL metadata: %w", err)
	}

	return nil
}

// createProgress создаёт progress bar для отображения прогресса создания БД.
// M-4 fix: используем общий helper progress.NewIndeterminate().
func (h *CreateTempDbHandler) createProgress() progress.Progress {
	return progress.NewIndeterminate()
}
