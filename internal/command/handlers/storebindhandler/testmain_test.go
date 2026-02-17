package storebindhandler

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	RegisterCmd()
	os.Exit(m.Run())
}
