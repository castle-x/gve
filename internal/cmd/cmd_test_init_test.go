package cmd

import (
	"os"
	"testing"

	"github.com/castle-x/gve/internal/i18n"
)

func TestMain(m *testing.M) {
	i18n.MustInit()
	os.Exit(m.Run())
}
