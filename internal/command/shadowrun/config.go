// Package shadowrun реализует middleware shadow-run для параллельного сравнения
// NR-команд с legacy-версиями. Активируется через BR_SHADOW_RUN=true.
package shadowrun

import (
	"os"
	"strings"

	"github.com/Kargones/apk-ci/internal/constants"
)

// IsEnabled возвращает true если shadow-run режим активирован через BR_SHADOW_RUN=true.
// При отсутствии переменной или любом другом значении возвращает false (backward compatible).
func IsEnabled() bool {
	val := os.Getenv(constants.EnvShadowRun)
	return strings.EqualFold(val, "true") || val == "1"
}
