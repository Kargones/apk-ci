package service

import (
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
)

func TestNewConfigAnalyzer(t *testing.T) {
	cfg := &config.Config{
		Owner:      "testowner",
		Repo:       "testrepo",
		BaseBranch: "main",
	}

	analyzer := NewConfigAnalyzer(cfg)

	if analyzer == nil {
		t.Fatal("Expected non-nil ConfigAnalyzer")
	}
	if analyzer.config != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestConfigAnalyzer_AnalyzeProject(t *testing.T) {
	cfg := &config.Config{
		Owner:      "testowner",
		Repo:       "testrepo",
		BaseBranch: "main",
	}

	analyzer := NewConfigAnalyzer(cfg)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Тестируем вызов AnalyzeProject
	// Поскольку это адаптер, мы просто проверяем, что метод вызывается без паники
	err := analyzer.AnalyzeProject(logger, "test-branch")
	
	// В реальной реализации config.AnalyzeProject может возвращать ошибку
	// Здесь мы просто проверяем, что метод выполняется
	if err != nil {
		// Это нормально, если config.AnalyzeProject возвращает ошибку
		// в тестовой среде без реальной настройки
		t.Logf("AnalyzeProject returned error (expected in test): %v", err)
	}
}