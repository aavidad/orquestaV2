// Este fichero reúne únicamente constructores de prueba para la vertical local.
package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const testAuthorization = "autorizacion-local-de-prueba-123456"

func testInventory(t *testing.T) (*inventory, inventoryItem) {
	t.Helper()
	directory := t.TempDir()
	filePath := filepath.Join(directory, "inventario.jsonl")
	value := map[string]any{
		"id":            "conducta:uno",
		"title":         "<script>alert('x')</script> Gestión de agentes",
		"summary":       "Conserva una conducta histórica para revisarla.",
		"family":        "agentes",
		"sources":       []string{"legacy/ref uno", "legacy/ref dos"},
		"attempts":      []string{"intento v1", "intento v2"},
		"uncertainties": []string{"cuota no acreditada"},
		"commit_id":     "abc123",
	}
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	inventory, err := loadInventory(filePath)
	if err != nil {
		t.Fatal(err)
	}
	item, found := inventory.find("conducta:uno")
	if !found {
		t.Fatal("el elemento de prueba no fue cargado")
	}
	return inventory, item
}

func testApplication(t *testing.T) (*application, inventoryItem) {
	t.Helper()
	inventory, item := testInventory(t)
	app := &application{
		inventory:     inventory,
		store:         newProposalStore(filepath.Join(t.TempDir(), "propuestas.jsonl")),
		catalog:       newCatalog("es"),
		authorization: []byte(testAuthorization),
		actorRef:      "actor:prueba",
		projectRef:    "project:prueba",
		allowedHost:   "127.0.0.1:8787",
		allowedOrigin: "http://127.0.0.1:8787",
		nonce: func() (string, error) {
			return strings.Repeat("a", 32), nil
		},
	}
	return app, item
}

func validProposalRequest(item inventoryItem, expected int, key string) proposalRequest {
	return proposalRequest{
		ItemRef:          item.ID,
		ItemRevision:     item.Revision,
		ExpectedRevision: expected,
		Disposition:      dispositionStudy,
		Reason:           "La necesidad conserva valor y requiere más evidencia.",
		FoundedSolution:  "Estudiar el mecanismo con una prueba neutral reproducible.",
		Confidence:       65,
		RuleCompliance:   ruleValues(true),
		RuleNotes:        "Todas las reglas se consideran compatibles en esta propuesta.",
		ActorRef:         "actor:prueba",
		ProjectRef:       "project:prueba",
		IdempotencyKey:   key,
	}
}

func ruleValues(value bool) map[string]bool {
	result := map[string]bool{}
	for _, rule := range requiredRules {
		result[rule] = value
	}
	return result
}

func validForm(app *application, item inventoryItem, expected int, nonce string) url.Values {
	values := url.Values{
		"item_ref":          {item.ID},
		"item_revision":     {item.Revision},
		"expected_revision": {strconv.Itoa(expected)},
		"csrf_token":        {app.signedValue("csrf", item.ID)},
		"form_nonce":        {nonce},
		"idempotency_key":   {app.signedValue("idempotency", item.ID, item.Revision, strconv.Itoa(expected), nonce)},
		"disposition":       {string(dispositionStudy)},
		"reason":            {"La necesidad conserva valor y requiere más evidencia."},
		"founded_solution":  {"Estudiar el mecanismo con una prueba neutral reproducible."},
		"confidence":        {"65"},
		"rule_notes":        {"Las reglas se han revisado de forma explícita."},
	}
	for _, rule := range requiredRules {
		values.Set("rule_"+rule, "yes")
	}
	return values
}

func performRequest(app *application, method, target string, body url.Values, authorized bool) *httptest.ResponseRecorder {
	var encoded string
	if body != nil {
		encoded = body.Encode()
	}
	request := httptest.NewRequest(method, target, strings.NewReader(encoded))
	request.Host = app.allowedHost
	if authorized {
		request.SetBasicAuth("local", testAuthorization)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Origin", app.allowedOrigin)
	}
	response := httptest.NewRecorder()
	app.handler().ServeHTTP(response, request)
	return response
}

func proposalFileMode(t *testing.T, filePath string) os.FileMode {
	t.Helper()
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
