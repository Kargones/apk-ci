package version

import (
	"context"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
)

type fakeNRHandler struct{ name string }

func (h *fakeNRHandler) Name() string                                      { return h.name }
func (h *fakeNRHandler) Description() string                               { return "fake" }
func (h *fakeNRHandler) Execute(_ context.Context, _ *config.Config) error { return nil }

func TestMain(m *testing.M) {
	RegisterCmd()
	command.RegisterWithAlias(&fakeNRHandler{name: "nr-fake-test"}, "fake-test")
	os.Exit(m.Run())
}
