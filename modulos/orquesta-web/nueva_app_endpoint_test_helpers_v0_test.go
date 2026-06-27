package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

type fakeNuevaAppClientV0 struct {
	calls    int
	received WebNuevaAppFormV0
	vm       WebNuevaAppViewModelV0
	err      error
}

type fakeArrancarDirectorAppClientV0 struct {
	calls    int
	received WebNuevaAppFormV0
	vm       WebNuevaAppViewModelV0
	err      error
}

type fakePreviewDirectorAppClientV0 struct {
	calls    int
	received WebNuevaAppFormV0
	vm       WebNuevaAppViewModelV0
	err      error
}

func (client *fakeArrancarDirectorAppClientV0) ArrancarDirectorApp(
	ctx context.Context,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	client.calls++
	client.received = form
	if client.vm.Locale == "" {
		client.vm.Locale = form.Locale
	}
	if client.vm.RequestID == "" {
		client.vm.RequestID = form.RequestID
	}
	return client.vm, client.err
}

func (client *fakePreviewDirectorAppClientV0) PreviewDirectorApp(
	ctx context.Context,
	form WebNuevaAppFormV0,
) (WebNuevaAppViewModelV0, error) {
	client.calls++
	client.received = form
	if client.vm.Locale == "" {
		client.vm.Locale = form.Locale
	}
	if client.vm.RequestID == "" {
		client.vm.RequestID = form.RequestID
	}
	return client.vm, client.err
}

func (client *fakeNuevaAppClientV0) SolicitarNuevaApp(ctx context.Context, form WebNuevaAppFormV0) (WebNuevaAppViewModelV0, error) {
	client.calls++
	client.received = form
	if client.vm.Locale == "" {
		client.vm.Locale = form.Locale
	}
	if client.vm.RequestID == "" {
		client.vm.RequestID = form.RequestID
	}
	return client.vm, client.err
}

func decodeNuevaAppWebPageTestV0(t *testing.T, rec *httptest.ResponseRecorder) NuevaAppWebPageV0 {
	t.Helper()
	var page NuevaAppWebPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v body=%s", err, rec.Body.String())
	}
	return page
}

func nuevaAppCampoByPathTestV0(fields []NuevaAppWebCampoV0, path string) NuevaAppWebCampoV0 {
	for _, field := range fields {
		if field.Path == path {
			return field
		}
	}
	return NuevaAppWebCampoV0{}
}

func nuevaAppHasOptionTestV0(options []NuevaAppWebOpcionV0, value string) bool {
	for _, option := range options {
		if option.Valor == value {
			return true
		}
	}
	return false
}
