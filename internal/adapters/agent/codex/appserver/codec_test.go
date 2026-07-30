package appserver

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"
)

func read(t *testing.T, input string, max int) (Message, error) {
	t.Helper()
	decoder, err := NewDecoder(iotest.OneByteReader(strings.NewReader(input)), max)
	if err != nil {
		t.Fatal(err)
	}
	return decoder.Next()
}

func mustRead(t *testing.T, input string) Message {
	t.Helper()
	message, err := read(t, input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func TestCodecBoundsAndClassification(t *testing.T) {
	frame := `{"id":1,"result":{"ok":true}}`
	if message, err := read(t, frame, len(frame)); err != nil || message.Kind != KindSuccess {
		t.Fatalf("exact: %#v %v", message, err)
	}
	if _, err := read(t, frame+"x", len(frame)); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("over: %v", err)
	}
	decoder, err := NewDecoder(strings.NewReader(strings.Repeat("x", 4096)+frame+"\n"+frame+"\n"), 1024)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		if _, err := decoder.Next(); !errors.Is(err, ErrFrameTooLarge) {
			t.Fatalf("intento %d tras exceso: %v", attempt, err)
		}
	}
	bad := map[string]error{
		"\n": ErrMalformed, `{"id":`: ErrMalformed, `{"id":1,"result":{}}x`: ErrMalformed,
		`{"id":1,"method":"x","result":{}}`: ErrMalformed, `{"id":true,"result":{}}`: ErrInvalidID,
		`{"id":1,"result":{},"error":{"code":1,"message":"x"}}`: ErrMalformed,
		`{"id":1,"error":{"message":"x"}}`:                      ErrMalformed,
		`{"id":1,"error":{"code":1}}`:                           ErrMalformed,
		`{"jsonrpc":"2.0","method":"x"}`:                        ErrMalformed,
	}
	for input, want := range bad {
		if _, err := read(t, input, len(input)); !errors.Is(err, want) {
			t.Errorf("%q: %v", input, err)
		}
	}
	request := mustRead(t, `{"id":"server","method":"approval","params":{}}`)
	if request.Kind != KindRequest || request.Method != "approval" {
		t.Fatalf("server request misclassified: %#v", request)
	}
}

func TestSessionSequenceAllowlistAndCorrelation(t *testing.T) {
	session := NewQuotaSession()
	if _, err := session.ReadQuota(int64(1)); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
	if _, err := session.Accept(mustRead(t, `{"method":"account/rateLimits/updated","params":{}}`)); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
	if _, err := session.Initialized(); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
	initialize, err := session.Initialize("init", ClientInfo{Name: "orquesta", Version: "1", Title: "quota"})
	if err != nil || !strings.Contains(string(initialize), `"method":"initialize"`) ||
		!strings.Contains(string(initialize), `"capabilities":null`) ||
		strings.Contains(string(initialize), "experimentalApi") ||
		strings.Contains(string(initialize), "requestAttestation") {
		t.Fatalf("%s %v", initialize, err)
	}
	if _, err := session.Initialize("again", ClientInfo{}); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
	if _, err := session.Accept(mustRead(t, `{"id":"init","result":{"userAgent":"x","codexHome":"/tmp/codex","platformFamily":"unix","platformOs":"linux"}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := session.ReadQuota(int64(1)); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
	if initialized, err := session.Initialized(); err != nil || string(initialized) != "{\"method\":\"initialized\"}\n" {
		t.Fatalf("%q %v", initialized, err)
	}
	if _, err := session.Initialized(); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
	if _, err := session.ReadQuota(true); !errors.Is(err, ErrInvalidID) {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if _, err := session.ReadQuota(id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := session.ReadQuota(int64(1)); !errors.Is(err, ErrDuplicateID) {
		t.Fatal(err)
	}
	for _, id := range []int64{2, 1} {
		method, err := session.Accept(mustRead(t, `{"id":`+string(rune('0'+id))+`,"result":{"raw":true}}`))
		if err != nil || method != "account/rateLimits/read" {
			t.Fatalf("%d %q %v", id, method, err)
		}
	}
	if _, err := session.Accept(mustRead(t, `{"id":99,"result":{}}`)); !errors.Is(err, ErrUnknownID) {
		t.Fatal(err)
	}
	if _, err := session.Accept(mustRead(t, `{"method":"thread/started","params":{}}`)); !errors.Is(err, ErrMethodNotAllowed) {
		t.Fatal(err)
	}
	if method, err := session.Accept(mustRead(t, `{"method":"account/rateLimits/updated","params":{"raw":true}}`)); err != nil || method == "" {
		t.Fatal(method, err)
	}
}

func TestErrorsAreRedactedAndInitializeFailsClosed(t *testing.T) {
	session := NewQuotaSession()
	if _, err := session.Initialize("init", ClientInfo{Name: "x", Version: "1"}); err != nil {
		t.Fatal(err)
	}
	failure := mustRead(t, `{"id":"init","error":{"code":-1,"message":"secret","data":"token"}}`)
	if failure.Kind != KindFailure || len(failure.Payload) != 0 {
		t.Fatalf("fallo remoto conservó contenido: %#v", failure)
	}
	_, err := session.Accept(failure)
	if !errors.Is(err, ErrInitialize) || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token") {
		t.Fatal(err)
	}
	if _, err := session.Initialized(); !errors.Is(err, ErrSequence) {
		t.Fatal(err)
	}
}

func TestInitializeRejectsIncompleteOrInvalidResult(t *testing.T) {
	for _, result := range []string{
		`{"userAgent":"x","codexHome":"/tmp/codex","platformFamily":"unix"}`,
		`{"userAgent":1,"codexHome":"/tmp/codex","platformFamily":"unix","platformOs":"linux"}`,
	} {
		session := NewQuotaSession()
		if _, err := session.Initialize("init", ClientInfo{Name: "x", Version: "1"}); err != nil {
			t.Fatal(err)
		}
		message := mustRead(t, `{"id":"init","result":`+result+`}`)
		if _, err := session.Accept(message); !errors.Is(err, ErrInitialize) ||
			strings.Contains(err.Error(), "/tmp/codex") {
			t.Fatalf("resultado inválido no cerró de forma redactada: %v", err)
		}
		if _, err := session.Initialized(); !errors.Is(err, ErrSequence) {
			t.Fatalf("sesión continuó tras resultado inválido: %v", err)
		}
	}
}

func TestArchitectureGuard(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	for _, name := range []string{"doc.go", "codec.go", "session.go"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(file), name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"firecracker", "internal/application", "internal/ports", "config/"} {
			if strings.Contains(strings.ToLower(string(data)), forbidden) {
				t.Fatalf("%s imports %s", name, forbidden)
			}
		}
	}
}
