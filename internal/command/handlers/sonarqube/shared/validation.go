// Package shared содержит общую логику для SonarQube команд.
package shared

import (
	"strings"

	"github.com/Kargones/apk-ci/internal/constants"
)

// IsValidBranchForScanning проверяет соответствие ветки критериям сканирования.
// Принимаемые ветки: "main" или "t" + 6-7 цифр (например, "t123456", "t1234567").
// L-2 fix: вынесено из scanbranch и reportbranch для устранения дублирования.
func IsValidBranchForScanning(branch string) bool {
	if branch == constants.BaseBranch {
		return true
	}
	if !strings.HasPrefix(branch, "t") {
		return false
	}
	digits := strings.TrimPrefix(branch, "t")
	if len(digits) < 6 || len(digits) > 7 {
		return false
	}
	for _, char := range digits {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
