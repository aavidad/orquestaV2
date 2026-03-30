package skillsapp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHTTPServerOrSkip(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(strings.ToLower(msg), "operation not permitted") {
				t.Skipf("sandbox sin permisos para listeners locales: %s", msg)
			}
			panic(r)
		}
	}()
	return httptest.NewServer(handler)
}
