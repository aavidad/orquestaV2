package configapp

import "testing"

type stubStore struct {
	clave string
	valor string
	agent string
	role  string
}

func (s *stubStore) GetConfig(clave string) (string, error) { return "high", nil }
func (s *stubStore) ListConfig() (map[string]string, error) { return map[string]string{"x": "y"}, nil }
func (s *stubStore) SetConfig(clave, valor string) error {
	s.clave = clave
	s.valor = valor
	return nil
}
func (s *stubStore) RegisterAgent(nombre, rol string) error {
	s.agent = nombre
	s.role = rol
	return nil
}
func (s *stubStore) RetireAgent(nombre string) error {
	s.agent = nombre
	return nil
}
func (s *stubStore) RehabilitateAgent(nombre string) error {
	s.agent = nombre
	return nil
}

func TestServiceNormalizaEntradas(t *testing.T) {
	store := &stubStore{}
	service := NewService(store)
	if err := service.Set(" clave ", " valor "); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if store.clave != "clave" || store.valor != "valor" {
		t.Fatalf("set inesperado: %+v", store)
	}
	if err := service.RegisterAgent(" Codex2 ", " programador "); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if store.agent != "Codex2" || store.role != "programador" {
		t.Fatalf("register inesperado: %+v", store)
	}
}
