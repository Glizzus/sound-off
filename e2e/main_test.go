package e2e_test

import (
	"os"
	"testing"

	"github.com/glizzus/sound-off/e2e"
)

func TestMain(m *testing.M) {
	code := m.Run()
	e2e.TerminatePostgresForE2E()
	os.Exit(code)
}
