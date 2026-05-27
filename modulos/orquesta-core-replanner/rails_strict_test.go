package orquestacorereplanner

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("ORQUESTA_SECURITY_MODE", "production")
	os.Setenv("ORQUESTA_DETAIL_PROHIBITED_RAILS", "on")
	os.Exit(m.Run())
}
