package importacionapp

import "orquesta/db"

type Repository struct{}

func (Repository) ImportLegacyProposalHistory() error {
	return db.ImportarHistorialOPs()
}

func (Repository) ImportWave2Tasks() error {
	return db.ImportarTareasOla2()
}
