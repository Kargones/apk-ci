// Package extensionpublishhandler — тонкий адаптер для legacy app.ExtensionPublish.
// TODO: Полноценная миграция логики из internal/app/extension_publish.go в этот handler.
package extensionpublishhandler

import (
	"context"

	"github.com/Kargones/apk-ci/internal/app"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

func init() {
	command.RegisterWithAlias(&ExtensionPublishHandler{}, constants.ActExtensionPublish)
}

// ExtensionPublishHandler — адаптер, делегирующий выполнение в legacy app.ExtensionPublish.
type ExtensionPublishHandler struct{}

// Name возвращает имя команды.
func (h *ExtensionPublishHandler) Name() string {
	return "nr-extension-publish"
}

// Description возвращает описание команды.
func (h *ExtensionPublishHandler) Description() string {
	return "Публикация расширения 1C"
}

// Execute делегирует выполнение в legacy app.ExtensionPublish.
func (h *ExtensionPublishHandler) Execute(ctx context.Context, cfg *config.Config) error {
	return app.ExtensionPublish(&ctx, cfg.Logger, cfg)
}
