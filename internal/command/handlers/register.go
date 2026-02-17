// Package handlers provides explicit registration of all command handlers.
// This replaces the previous init()-based implicit registration pattern,
// making the dependency graph explicit, testable, and free of import side effects.
package handlers

import (
	"github.com/Kargones/apk-ci/internal/command/handlers/converthandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/createstoreshandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/createtempdbhandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/dbrestorehandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/dbupdatehandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/deprecatedaudithandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/executeepfhandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/extensionpublishhandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/forcedisconnecthandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/git2storehandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/gitea/actionmenu"
	"github.com/Kargones/apk-ci/internal/command/handlers/gitea/testmerge"
	"github.com/Kargones/apk-ci/internal/command/handlers/help"
	"github.com/Kargones/apk-ci/internal/command/handlers/migratehandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/servicemodedisablehandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/servicemodeenablehandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/servicemodestatushandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/projectupdate"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/reportbranch"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/scanbranch"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/scanpr"
	"github.com/Kargones/apk-ci/internal/command/handlers/store2dbhandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/storebindhandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/version"
)

// RegisterAll explicitly registers all command handlers in the global registry.
// Call this once from main() before using any commands.
func RegisterAll() {
	converthandler.RegisterCmd()
	createstoreshandler.RegisterCmd()
	createtempdbhandler.RegisterCmd()
	dbrestorehandler.RegisterCmd()
	dbupdatehandler.RegisterCmd()
	deprecatedaudithandler.RegisterCmd()
	executeepfhandler.RegisterCmd()
	extensionpublishhandler.RegisterCmd()
	forcedisconnecthandler.RegisterCmd()
	git2storehandler.RegisterCmd()
	actionmenu.RegisterCmd()
	testmerge.RegisterCmd()
	help.RegisterCmd()
	migratehandler.RegisterCmd()
	servicemodedisablehandler.RegisterCmd()
	servicemodeenablehandler.RegisterCmd()
	servicemodestatushandler.RegisterCmd()
	projectupdate.RegisterCmd()
	reportbranch.RegisterCmd()
	scanbranch.RegisterCmd()
	scanpr.RegisterCmd()
	store2dbhandler.RegisterCmd()
	storebindhandler.RegisterCmd()
	version.RegisterCmd()
}
