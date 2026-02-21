// Package handlers provides explicit registration of all command handlers.
// This replaces the previous init()-based implicit registration pattern,
// making the dependency graph explicit, testable, and free of import side effects.
package handlers

import (
	"github.com/Kargones/apk-ci/internal/command/handlers/converthandler"
	"github.com/Kargones/apk-ci/internal/command/handlers/convertpipelinehandler"
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
// Returns an error if any handler registration fails.
func RegisterAll() error {
	if err := converthandler.RegisterCmd(); err != nil {
		return err
	}
	if err := convertpipelinehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := createstoreshandler.RegisterCmd(); err != nil {
		return err
	}
	if err := createtempdbhandler.RegisterCmd(); err != nil {
		return err
	}
	if err := dbrestorehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := dbupdatehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := deprecatedaudithandler.RegisterCmd(); err != nil {
		return err
	}
	if err := executeepfhandler.RegisterCmd(); err != nil {
		return err
	}
	if err := extensionpublishhandler.RegisterCmd(); err != nil {
		return err
	}
	if err := forcedisconnecthandler.RegisterCmd(); err != nil {
		return err
	}
	if err := git2storehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := actionmenu.RegisterCmd(); err != nil {
		return err
	}
	if err := testmerge.RegisterCmd(); err != nil {
		return err
	}
	if err := help.RegisterCmd(); err != nil {
		return err
	}
	if err := migratehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := servicemodedisablehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := servicemodeenablehandler.RegisterCmd(); err != nil {
		return err
	}
	if err := servicemodestatushandler.RegisterCmd(); err != nil {
		return err
	}
	if err := projectupdate.RegisterCmd(); err != nil {
		return err
	}
	if err := reportbranch.RegisterCmd(); err != nil {
		return err
	}
	if err := scanbranch.RegisterCmd(); err != nil {
		return err
	}
	if err := scanpr.RegisterCmd(); err != nil {
		return err
	}
	if err := store2dbhandler.RegisterCmd(); err != nil {
		return err
	}
	if err := storebindhandler.RegisterCmd(); err != nil {
		return err
	}
	if err := version.RegisterCmd(); err != nil {
		return err
	}
	return nil
}
