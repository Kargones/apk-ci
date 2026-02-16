package shadowrun

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareResults_IdenticalSuccess(t *testing.T) {
	result := CompareResults(nil, nil, "output data", "output data")
	assert.True(t, result.Match)
	assert.Empty(t, result.Differences)
}

func TestCompareResults_IdenticalError(t *testing.T) {
	nrErr := errors.New("some error")
	legacyErr := errors.New("some error")
	// Обе вернули одинаковую ошибку — результат идентичен
	result := CompareResults(nrErr, legacyErr, "same output", "same output")
	assert.True(t, result.Match)
	assert.Empty(t, result.Differences)
}

func TestCompareResults_DifferentErrorText(t *testing.T) {
	// Review #31: Обе вернули ошибку, но с разным текстом — должно быть различие
	nrErr := errors.New("timeout after 30s")
	legacyErr := errors.New("connection refused")
	result := CompareResults(nrErr, legacyErr, "same output", "same output")
	assert.False(t, result.Match)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "error", result.Differences[0].Field)
	assert.Equal(t, "timeout after 30s", result.Differences[0].NRValue)
	assert.Equal(t, "connection refused", result.Differences[0].LegacyValue)
}

func TestCompareResults_DifferentErrors(t *testing.T) {
	nrErr := errors.New("nr error")
	// NR вернула ошибку, legacy — нет
	result := CompareResults(nrErr, nil, "", "")
	assert.False(t, result.Match)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "error", result.Differences[0].Field)
	assert.Equal(t, "nr error", result.Differences[0].NRValue)
	assert.Equal(t, "<nil>", result.Differences[0].LegacyValue)
}

func TestCompareResults_DifferentErrors_Reverse(t *testing.T) {
	legacyErr := errors.New("legacy error")
	// Legacy вернула ошибку, NR — нет
	result := CompareResults(nil, legacyErr, "", "")
	assert.False(t, result.Match)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "error", result.Differences[0].Field)
	assert.Equal(t, "<nil>", result.Differences[0].NRValue)
	assert.Equal(t, "legacy error", result.Differences[0].LegacyValue)
}

func TestCompareResults_DifferentOutput(t *testing.T) {
	result := CompareResults(nil, nil, "nr output", "legacy output")
	assert.False(t, result.Match)
	require.Len(t, result.Differences, 1)
	assert.Equal(t, "output", result.Differences[0].Field)
	assert.Equal(t, "nr output", result.Differences[0].NRValue)
	assert.Equal(t, "legacy output", result.Differences[0].LegacyValue)
}

func TestCompareResults_BothDifferent(t *testing.T) {
	nrErr := errors.New("nr err")
	// Различия и в ошибке, и в выводе
	result := CompareResults(nrErr, nil, "nr output", "legacy output")
	assert.False(t, result.Match)
	assert.Len(t, result.Differences, 2)
}

func TestCompareResults_WhitespaceTrimming(t *testing.T) {
	// Trailing whitespace должен игнорироваться
	result := CompareResults(nil, nil, "output\n", "output  \n")
	assert.True(t, result.Match)
}

func TestCompareResults_EmptyOutput(t *testing.T) {
	result := CompareResults(nil, nil, "", "")
	assert.True(t, result.Match)
	assert.Empty(t, result.Differences)
}

func TestFormatDiff_Match(t *testing.T) {
	comparison := &ComparisonResult{Match: true}
	diff := FormatDiff(comparison)
	assert.Equal(t, "Результаты идентичны", diff)
}

func TestFormatDiff_WithDifferences(t *testing.T) {
	comparison := &ComparisonResult{
		Match: false,
		Differences: []Difference{
			{Field: "error", NRValue: "err1", LegacyValue: "<nil>"},
			{Field: "output", NRValue: "abc", LegacyValue: "xyz"},
		},
	}
	diff := FormatDiff(comparison)
	assert.Contains(t, diff, "Обнаружены различия")
	assert.Contains(t, diff, "[error]")
	assert.Contains(t, diff, "[output]")
	assert.Contains(t, diff, "NR:")
	assert.Contains(t, diff, "Legacy:")
}

func TestTruncate_Short(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
}

func TestTruncate_Long(t *testing.T) {
	result := truncate("hello world", 5)
	assert.Equal(t, "hello...", result)
}

func TestTruncate_Exact(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 5))
}

func TestTruncate_UTF8_Cyrillic(t *testing.T) {
	// Кириллическая строка: каждый символ 2 байта в UTF-8
	s := "Здравствуй мир"
	result := truncate(s, 5)
	assert.Equal(t, "Здрав...", result, "должен обрезать по рунам, а не по байтам")
}

func TestTruncate_UTF8_NoCut(t *testing.T) {
	// Короткая кириллическая строка — не обрезается
	s := "Привет"
	result := truncate(s, 10)
	assert.Equal(t, "Привет", result)
}

func TestTruncate_UTF8_Mixed(t *testing.T) {
	// Смешанная строка: ASCII + кириллица
	s := "test_Жёлтый"
	result := truncate(s, 7)
	assert.Equal(t, "test_Жё...", result)
}
