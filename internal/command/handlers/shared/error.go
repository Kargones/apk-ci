package shared

import (
	"fmt"
	"os"
)

// HandleError writes a standardized error message to stdout and returns a formatted error.
// Used by handlers to report errors in a consistent format.
func HandleError(message, code string) error {
	_, _ = fmt.Fprintf(os.Stdout, "Ошибка: %s\nКод: %s\n", message, code)
	return fmt.Errorf("%s: %s", code, message)
}
