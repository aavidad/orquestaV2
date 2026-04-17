package configuracionapp

import "orquesta/db"

type Repository struct{}

func (Repository) GetConfig(clave string) (string, error) {
	return db.ConfigGet(clave)
}

func (Repository) ListConfig() (map[string]string, error) {
	return db.ConfigAll()
}

func (Repository) SetConfig(clave, valor string) error {
	return db.ConfigSet(clave, valor)
}

func (Repository) RegisterAgent(nombre, rol string) error {
	return db.RegistrarAgente(nombre, rol)
}

func (Repository) RetireAgent(nombre string) error {
	return db.RetirarAgente(nombre)
}

func (Repository) RehabilitateAgent(nombre string) error {
	return db.RehabilitarAgente(nombre)
}
