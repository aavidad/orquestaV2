/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitoperaciones"
)

func newCoordinationService() *coordinacion.Service {
	return &coordinacion.Service{
		Locks:     db.SQLiteLockRepository{},
		Worktrees: db.SQLiteWorktreeRepository{},
		Projects:  db.SQLiteProjectRepository{},
		Sessions:  db.SQLiteSessionRepository{},
		Config:    db.SQLiteConfigRepository{},
		Workspace: gitoperaciones.WorktreeManager{},
	}
}
