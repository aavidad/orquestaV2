/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"orquesta/coordination"
	"orquesta/db"
	"orquesta/gitops"
)

func newCoordinationService() *coordination.Service {
	return &coordination.Service{
		Locks:     db.SQLiteLockRepository{},
		Worktrees: db.SQLiteWorktreeRepository{},
		Projects:  db.SQLiteProjectRepository{},
		Sessions:  db.SQLiteSessionRepository{},
		Config:    db.SQLiteConfigRepository{},
		Workspace: gitops.WorktreeManager{},
	}
}
