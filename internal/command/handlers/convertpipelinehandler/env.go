package convertpipelinehandler

import "os"

// getenv — обёртка для os.Getenv (тестируемость через подмену в тестах).
var getenv = os.Getenv
