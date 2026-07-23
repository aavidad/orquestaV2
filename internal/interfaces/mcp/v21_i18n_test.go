package mcpinterface

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/i18n"
)

func TestMCPMetadataUsesResolvedLocaleAndSpanishFallback(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		locale string
	}{
		{name: "Spanish", locale: "es"},
		{name: "English", locale: "en"},
		{name: "UnsupportedFallsBackToSpanish", locale: "gl"},
	} {
		t.Run(test.name, func(t *testing.T) {
			executor := &v20Executor{}
			presentation, err := New(Config{
				Dispatcher: executor,
				Identity:   v20Identity{principal: v20Principal(t)},
				Catalog:    catalog,
				Locale:     test.locale,
				Version:    "test",
			})
			if err != nil {
				t.Fatal(err)
			}
			endpoint := httptest.NewServer(presentation.Handler())
			t.Cleanup(endpoint.Close)
			session := connectOfficialClient(t, endpoint.URL)
			wantInstructions, err := catalog.Text(test.locale, "server.instructions")
			if err != nil {
				t.Fatal(err)
			}
			if got := session.InitializeResult().Instructions; got != wantInstructions {
				t.Fatalf("instructions=%q want=%q", got, wantInstructions)
			}
			listed, err := session.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			wantDescription, err := catalog.Text(test.locale, "command.system.status.description")
			if err != nil {
				t.Fatal(err)
			}
			var gotDescription string
			for _, tool := range listed.Tools {
				if tool.Name == "orquesta.system.status" {
					gotDescription = tool.Description
					break
				}
			}
			if gotDescription != wantDescription || gotDescription == "command.system.status.description" {
				t.Fatalf("description=%q want=%q", gotDescription, wantDescription)
			}
		})
	}
}

func TestMCPRejectsInvalidLocaleBeforePublishingServer(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(Config{
		Dispatcher: &v20Executor{},
		Identity:   v20Identity{principal: v20Principal(t)},
		Catalog:    catalog,
		Locale:     "not_a_locale!",
		Version:    "test",
	})
	if !errors.Is(err, i18n.ErrLocaleInvalid) {
		t.Fatalf("New() error=%v", err)
	}
}

func TestMCPTransportLimitDoesNotInventHumanText(t *testing.T) {
	handler := requestBodyLimit(4, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("oversized request reached MCP server")
	}))
	request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	request.Body = http.NoBody
	request.ContentLength = 5
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge || response.Body.Len() != 0 {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestMCPLocaleChangesOnlyHumanMetadata(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	want := commandcore.Result{
		Failure: &commandcore.Failure{
			Code:       commandcore.CodeConflict,
			MessageKey: "error.conflict",
		},
		AuditRef: "command-audit:locale-invariant",
	}
	var reference commandcore.Result
	for index, locale := range []string{"es", "en", "gl"} {
		t.Run(locale, func(t *testing.T) {
			executor := &v20Executor{result: want}
			presentation, err := New(Config{
				Dispatcher: executor,
				Identity:   v20Identity{principal: v20Principal(t)},
				Catalog:    catalog,
				Locale:     locale,
				Version:    "test",
			})
			if err != nil {
				t.Fatal(err)
			}
			endpoint := httptest.NewServer(presentation.Handler())
			t.Cleanup(endpoint.Close)
			called := callTool(t, connectOfficialClient(t, endpoint.URL), "orquesta.system.status", map[string]any{
				"version": "1", "request_ref": "request:i18n-invariant",
				"project_ref": "project:v21", "payload": map[string]any{},
			})
			var output CommandToolOutput
			decodeStructured(t, called, &output)
			if index == 0 {
				reference = output.Result
				return
			}
			if !reflect.DeepEqual(output.Result, reference) {
				left, _ := json.Marshal(reference)
				right, _ := json.Marshal(output.Result)
				t.Fatalf("locale=%s result=%s reference=%s", locale, right, left)
			}
		})
	}
}
