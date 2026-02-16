// Package extensionpublishhandler — handler для публикации расширений 1C.
package extensionpublishhandler

import (
	"context"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

func init() {
	command.RegisterWithAlias(&ExtensionPublishHandler{}, constants.ActExtensionPublish)
}

// ExtensionPublishHandler — handler для публикации расширений 1C.
type ExtensionPublishHandler struct{}

// Name возвращает имя команды.
func (h *ExtensionPublishHandler) Name() string {
	return "nr-extension-publish"
}

// Description возвращает описание команды.
func (h *ExtensionPublishHandler) Description() string {
	return "Публикация расширения 1C"
}

// Execute выполняет публикацию расширения.
func (h *ExtensionPublishHandler) Execute(ctx context.Context, cfg *config.Config) error {
	return ExtensionPublish(&ctx, cfg.Logger, cfg)
}
