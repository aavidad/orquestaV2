package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/coordinacion"
	"orquesta/storage"
)

type TipoProyecto string

const (
	ProyectoRaiz  TipoProyecto = "raiz"
	ProyectoGrupo TipoProyecto = "grupo"
	ProyectoRepo  TipoProyecto = "repo"
)

type Proyecto struct {
	ID         int64
	Slug       string
	Nombre     string
	RutaAbs    string
	OrigenRepo string
	RemoteURL  string
	BranchBase string
	Tipo       TipoProyecto
	ParentID   *int64
	Activo     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EstadisticasProyecto struct {
	Total       int
	Completadas int
	EnProgreso  int
	Asignadas   int
	Bloqueadas  int
}

type FiltroProyectos struct {
	Tipo     *TipoProyecto
	ParentID *int64
	Activo   *bool
}

type proyectoRutaSesion struct {
	agente string
	cwd    string
}

func UpsertProyecto(p *Proyecto) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("proyecto nil")
	}
	if strings.TrimSpace(p.RutaAbs) == "" {
		return 0, fmt.Errorf("ruta_abs obligatoria")
	}
	abs, err := filepath.Abs(p.RutaAbs)
	if err != nil {
		return 0, err
	}
	p.RutaAbs = filepath.Clean(abs)
	if strings.TrimSpace(p.Nombre) == "" {
		p.Nombre = filepath.Base(p.RutaAbs)
	}
	if strings.TrimSpace(p.Slug) == "" {
		p.Slug = slugProyecto(p.Nombre)
	}
	if strings.TrimSpace(string(p.Tipo)) == "" {
		p.Tipo = ProyectoRepo
	}
	if strings.TrimSpace(p.OrigenRepo) == "" {
		p.OrigenRepo = "local"
	}

	var existenteID int64
	err = DB.QueryRow(`SELECT id FROM proyectos WHERE slug = ? OR ruta_abs = ?`, p.Slug, p.RutaAbs).Scan(&existenteID)
	switch err {
	case nil:
		_, err = DB.Exec(`
			UPDATE proyectos
			SET slug=?, nombre=?, ruta_abs=?, origen_repo=?, remote_url=?, branch_base=?, tipo=?, parent_id=?, activo=?
			WHERE id=?`,
			p.Slug, p.Nombre, p.RutaAbs, p.OrigenRepo, p.RemoteURL, p.BranchBase, p.Tipo, p.ParentID, boolToInt(p.Activo), existenteID,
		)
		if err != nil {
			return 0, err
		}
		return existenteID, nil
	case sql.ErrNoRows:
		id, err := insertReturningID(`
			INSERT INTO proyectos (slug, nombre, ruta_abs, origen_repo, remote_url, branch_base, tipo, parent_id, activo)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			p.Slug, p.Nombre, p.RutaAbs, p.OrigenRepo, p.RemoteURL, p.BranchBase, p.Tipo, p.ParentID, boolToInt(p.Activo),
		)
		if err != nil {
			return 0, err
		}
		return id, nil
	default:
		return 0, err
	}
}

func UpdateProyecto(p *Proyecto) error {
	if p == nil {
		return fmt.Errorf("proyecto nil")
	}
	if p.ID == 0 {
		return fmt.Errorf("id de proyecto obligatorio")
	}
	if strings.TrimSpace(p.RutaAbs) == "" {
		return fmt.Errorf("ruta_abs obligatoria")
	}
	abs, err := filepath.Abs(p.RutaAbs)
	if err != nil {
		return err
	}
	p.RutaAbs = filepath.Clean(abs)
	if strings.TrimSpace(p.Nombre) == "" {
		p.Nombre = filepath.Base(p.RutaAbs)
	}
	if strings.TrimSpace(p.Slug) == "" {
		p.Slug = slugProyecto(p.Nombre)
	}
	if strings.TrimSpace(string(p.Tipo)) == "" {
		p.Tipo = ProyectoRepo
	}
	if strings.TrimSpace(p.OrigenRepo) == "" {
		p.OrigenRepo = "local"
	}
	if _, err := DB.Exec(`
		UPDATE proyectos
		SET slug=?, nombre=?, ruta_abs=?, origen_repo=?, remote_url=?, branch_base=?, tipo=?, parent_id=?, activo=?
		WHERE id=?`,
		p.Slug, p.Nombre, p.RutaAbs, p.OrigenRepo, p.RemoteURL, p.BranchBase, p.Tipo, p.ParentID, boolToInt(p.Activo), p.ID,
	); err != nil {
		return err
	}
	return nil
}

func GetProyecto(ref string) (*Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("referencia de proyecto vacía")
	}

	q := `
		SELECT id, slug, nombre, ruta_abs, origen_repo, remote_url, branch_base, tipo, parent_id, activo, created_at, updated_at
		FROM proyectos
		WHERE slug = ? OR ruta_abs = ?`
	args := []any{ref, filepath.Clean(ref)}

	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		q += ` OR id = ?`
		args = append(args, id)
	}

	return consultarConReintentos(func() (*Proyecto, error) {
		p, err := escanearProyecto(DB.QueryRow(q, args...))
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return p, err
	})
}

func GetProyectoPrepareLite(ref string) (*Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("referencia de proyecto vacía")
	}
	if proyecto, ok, err := getProyectoPrepareLiteReadOnly(ref); ok {
		return proyecto, err
	}
	return GetProyecto(ref)
}

func getProyectoPrepareLiteReadOnly(ref string) (*Proyecto, bool, error) {
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return nil, false, err
	}
	driver := normalizedDriverName(cfg.Driver)
	if !supportsPrepareLiteReadOnlyBackend(driver) {
		return nil, false, nil
	}
	disabled := false
	skipPost := true
	cfg = applyOpenOptions(cfg, OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
		ReadOnly:           true,
	})
	cfg.MaxOpenConns = 1
	raw, err := backend.Open(cfg)
	if err != nil {
		return nil, true, err
	}
	defer raw.Close()

	q := `
		SELECT id, slug, nombre, ruta_abs, origen_repo, remote_url, branch_base, tipo, parent_id, activo, created_at, updated_at
		FROM proyectos
		WHERE slug = ? OR ruta_abs = ?`
	args := []any{ref, filepath.Clean(ref)}
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		q += ` OR id = ?`
		args = append(args, id)
	}
	row := raw.QueryRow(storage.RebindQuery(driver, q), args...)
	p, err := escanearProyecto(row)
	if err == sql.ErrNoRows {
		return nil, true, nil
	}
	return p, true, err
}

func GetProyectoConRutaEfectiva(ref, cwdHint string) (*Proyecto, error) {
	proyecto, err := GetProyecto(ref)
	if err != nil {
		return nil, err
	}
	return ProyectoConRutaEfectiva(proyecto, cwdHint), nil
}

func ProyectoConRutaEfectiva(proyecto *Proyecto, cwdHint string) *Proyecto {
	if proyecto == nil {
		return nil
	}
	copia := *proyecto
	copia.RutaAbs = RutaProyectoEfectiva(copia.ID, copia.RutaAbs, cwdHint)
	return &copia
}

func ProyectoPrepareLiteConRutaEfectiva(proyecto *Proyecto, agente string) *Proyecto {
	if proyecto == nil {
		return nil
	}
	if canonical, err := CanonicalizeAgentName(agente); err == nil {
		agente = canonical
	}
	copia := *proyecto
	copia.RutaAbs = rutaProyectoPrepareLite(copia.ID, strings.TrimSpace(agente), copia.RutaAbs)
	return &copia
}

func ProyectosConRutaEfectiva(proyectos []*Proyecto, cwdHint string) []*Proyecto {
	if len(proyectos) == 0 {
		return proyectos
	}
	sesionesPorProyecto, worktreesPorProyecto := cargarContextoRutaProyectos(proyectos)
	out := make([]*Proyecto, 0, len(proyectos))
	for _, proyecto := range proyectos {
		out = append(out, proyectoConRutaEfectivaConContexto(
			proyecto,
			cwdHint,
			sesionesPorProyecto[proyectoIDOrZero(proyecto)],
			worktreesPorProyecto[proyectoIDOrZero(proyecto)],
		))
	}
	return out
}

func proyectoConRutaEfectivaConContexto(proyecto *Proyecto, cwdHint string, sesion *proyectoRutaSesion, worktrees []coordinacion.WorktreePathRef) *Proyecto {
	if proyecto == nil {
		return nil
	}
	copia := *proyecto
	copia.RutaAbs = rutaProyectoEfectivaConContexto(copia.ID, copia.RutaAbs, cwdHint, sesion, worktrees)
	return &copia
}

func RutaProyectoEfectiva(proyectoID int64, fallback, cwdHint string) string {
	return rutaProyectoEfectivaConContexto(proyectoID, fallback, cwdHint, nil, nil)
}

func rutaProyectoEfectivaConContexto(proyectoID int64, fallback, cwdHint string, sesion *proyectoRutaSesion, worktrees []coordinacion.WorktreePathRef) string {
	fallback = normalizarRutaProyecto(fallback)
	sessionCWD := ""
	sessionInsideActiveWorktree := false
	if sesion != nil {
		sessionCWD = normalizarRutaProyecto(sesion.cwd)
		sessionInsideActiveWorktree = rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID, sesion.agente, sessionCWD, worktrees, fallback)
		if coordinacion.CandidateEffectiveProjectPath(sessionCWD, sessionInsideActiveWorktree, tieneMarcadoresRepo) == "" {
			sessionCWD = ""
			sessionInsideActiveWorktree = false
		}
	}
	if sessionCWD == "" {
		if agente, cwd, ok := mejorRutaSesionProyecto(proyectoID, worktrees, fallback); ok {
			sessionCWD = normalizarRutaProyecto(cwd)
			sessionInsideActiveWorktree = rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID, agente, sessionCWD, worktrees, fallback)
		}
	}
	return coordinacion.ResolveProjectEffectivePath(
		fallback,
		cwdHint,
		rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID, "", cwdHint, worktrees, fallback),
		sessionCWD,
		sessionInsideActiveWorktree,
		tieneMarcadoresRepo,
	)
}

func candidataRutaProyectoEfectiva(proyectoID int64, agente, cwd string) string {
	cwd = normalizarRutaProyecto(cwd)
	if cwd == "" {
		return ""
	}
	return coordinacion.ResolveProjectEffectivePath(
		"",
		cwd,
		rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID, agente, cwd, nil, ""),
		"",
		false,
		tieneMarcadoresRepo,
	)
}

func candidataRutaProyectoEfectivaConWorktrees(proyectoID int64, agente, cwd string, worktrees []coordinacion.WorktreePathRef) string {
	cwd = normalizarRutaProyecto(cwd)
	if cwd == "" {
		return ""
	}
	return coordinacion.ResolveProjectEffectivePath(
		"",
		cwd,
		rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID, agente, cwd, worktrees, ""),
		"",
		false,
		tieneMarcadoresRepo,
	)
}

func ultimaRutaSesionProyecto(proyectoID int64) (string, string, bool) {
	if proyectoID <= 0 || DB == nil {
		return "", "", false
	}
	var agente, cwd string
	if err := DB.QueryRow(`
		SELECT agente, cwd
		FROM sesiones
		WHERE proyecto_id = ?
		  AND trim(cwd) <> ''
		ORDER BY activa DESC, id DESC
		LIMIT 1`, proyectoID,
	).Scan(&agente, &cwd); err != nil {
		return "", "", false
	}
	return strings.TrimSpace(agente), strings.TrimSpace(cwd), true
}

func mejorRutaSesionProyecto(proyectoID int64, worktrees []coordinacion.WorktreePathRef, fallback string) (string, string, bool) {
	if proyectoID <= 0 || DB == nil {
		return "", "", false
	}
	rows, err := DB.Query(`
		SELECT agente, cwd
		FROM sesiones
		WHERE proyecto_id = ?
		  AND trim(cwd) <> ''
		ORDER BY activa DESC, id DESC`, proyectoID,
	)
	if err != nil {
		return "", "", false
	}
	type sesionRuta struct {
		agente string
		cwd    string
	}
	sesiones := make([]sesionRuta, 0)
	for rows.Next() {
		var agente, cwd string
		if err := rows.Scan(&agente, &cwd); err != nil {
			rows.Close()
			return "", "", false
		}
		sesiones = append(sesiones, sesionRuta{
			agente: strings.TrimSpace(agente),
			cwd:    normalizarRutaProyecto(cwd),
		})
	}
	rows.Close()
	for _, sesion := range sesiones {
		if sesion.cwd == "" {
			continue
		}
		inside := rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID, sesion.agente, sesion.cwd, worktrees, fallback)
		if coordinacion.CandidateEffectiveProjectPath(sesion.cwd, inside, tieneMarcadoresRepo) == "" {
			continue
		}
		return sesion.agente, sesion.cwd, true
	}
	return "", "", false
}

func normalizarRutaProyecto(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func resolverRaizTrabajo(path string) string {
	raiz, _ := resolverRaizTrabajoConMarcadores(path)
	return raiz
}

func resolverRaizTrabajoConMarcadores(path string) (string, bool) {
	return coordinacion.ResolveWorkRoot(path, tieneMarcadoresRepo)
}

func RutaTrabajoPreferidaAgenteProyecto(agente string, proyecto *Proyecto, cwd string) string {
	if proyecto == nil {
		return normalizarRutaProyecto(cwd)
	}
	rutaProyectoEfectiva := rutaProyectoEfectivaDesdeProyecto(proyecto, "")
	return coordinacion.PreferredWorkPath(
		cwd,
		rutaWorktreeActivaAgenteProyecto(proyecto.ID, agente),
		rutaProyectoEfectiva,
		rutaSesionWorktreeProyectoConRuta(cwd, rutaProyectoEfectiva),
	)
}

func RutaTrabajoPreferidaAgenteProyectoPrepareLite(agente string, proyecto *Proyecto, cwd string) string {
	if proyecto == nil {
		return normalizarRutaProyecto(cwd)
	}
	if canonical, err := CanonicalizeAgentName(agente); err == nil {
		agente = canonical
	}
	agente = strings.TrimSpace(agente)
	rutaProyecto := rutaProyectoPrepareLite(proyecto.ID, agente, proyecto.RutaAbs)
	return coordinacion.PreferredWorkPath(
		cwd,
		rutaWorktreeActivaPrepareLite(proyecto.ID, agente, rutaProyecto),
		rutaProyecto,
		rutaSesionWorktreeProyectoConRuta(cwd, rutaProyecto),
	)
}

func rutaSesionWorktreeProyecto(proyecto *Proyecto, cwd string) bool {
	return rutaSesionWorktreeProyectoConRuta(cwd, rutaProyectoEfectivaDesdeProyecto(proyecto, ""))
}

func rutaSesionWorktreeProyectoConRuta(cwd, rutaProyectoEfectiva string) bool {
	cwd = normalizarRutaProyecto(cwd)
	rutaProyectoEfectiva = normalizarRutaProyecto(rutaProyectoEfectiva)
	if cwd == "" || rutaProyectoEfectiva == "" {
		return false
	}
	return coordinacion.CurrentPathInsideProjectWorktree(cwd, rutaProyectoEfectiva)
}

func rutaTrabajoPerteneceAProyectoAgente(proyecto *Proyecto, agente, cwd string) bool {
	cwd = normalizarRutaProyecto(cwd)
	if proyecto == nil || cwd == "" {
		return false
	}
	rutaBase := rutaProyectoEfectivaDesdeProyecto(proyecto, "")
	return coordinacion.WorkPathBelongsToProject(cwd, rutaBase, rutaWorktreeActivaAgenteProyecto(proyecto.ID, agente))
}

func rutaProyectoEfectivaDesdeProyecto(proyecto *Proyecto, cwdHint string) string {
	if proyecto == nil {
		return ""
	}
	return normalizarRutaProyecto(RutaProyectoEfectiva(proyecto.ID, proyecto.RutaAbs, cwdHint))
}

func rutaProyectoPrepareLite(proyectoID int64, agente, fallback string) string {
	fallback = normalizarRutaProyecto(fallback)
	if proyectoID <= 0 {
		return fallback
	}
	agente = strings.TrimSpace(agente)
	if agente != "" {
		if rutaSesion := rutaSesionProyectoPrepareLite(proyectoID, agente); rutaSesion != "" {
			return rutaSesion
		}
	}
	if rutaWorktree := rutaWorktreeActivaPrepareLite(proyectoID, agente, fallback); rutaWorktree != "" {
		if ruta := coordinacion.CandidateEffectiveProjectPath(rutaWorktree, true, tieneMarcadoresRepo); ruta != "" {
			return ruta
		}
	}
	if ruta := coordinacion.CandidateEffectiveProjectPath(fallback, false, tieneMarcadoresRepo); ruta != "" {
		return ruta
	}
	if rutaSesion := rutaSesionProyectoPrepareLite(proyectoID, ""); rutaSesion != "" {
		return rutaSesion
	}
	return fallback
}

func rutaSesionProyectoPrepareLite(proyectoID int64, agente string) string {
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return rutaSesionProyectoPrepareLiteMainDB(proyectoID, agente)
	}
	driver := normalizedDriverName(cfg.Driver)
	if !supportsPrepareLiteReadOnlyBackend(driver) {
		return rutaSesionProyectoPrepareLiteMainDB(proyectoID, agente)
	}
	disabled := false
	skipPost := true
	cfg = applyOpenOptions(cfg, OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
		ReadOnly:           true,
	})
	cfg.MaxOpenConns = 1
	raw, err := backend.Open(cfg)
	if err != nil {
		return rutaSesionProyectoPrepareLiteMainDB(proyectoID, agente)
	}
	defer raw.Close()
	return rutaSesionProyectoPrepareLiteWithQuery(
		func(q string, args ...any) (*sql.Rows, error) {
			return raw.Query(storage.RebindQuery(driver, q), args...)
		},
		proyectoID,
		agente,
	)
}

func rutaSesionProyectoPrepareLiteMainDB(proyectoID int64, agente string) string {
	if proyectoID <= 0 || DB == nil {
		return ""
	}
	return rutaSesionProyectoPrepareLiteWithQuery(
		func(q string, args ...any) (*sql.Rows, error) {
			return DB.Query(q, args...)
		},
		proyectoID,
		agente,
	)
}

func rutaSesionProyectoPrepareLiteWithQuery(queryFn func(string, ...any) (*sql.Rows, error), proyectoID int64, agente string) string {
	if proyectoID <= 0 || queryFn == nil {
		return ""
	}
	agenteCanonico, err := CanonicalizeAgentName(agente)
	if err != nil {
		agenteCanonico = strings.TrimSpace(agente)
	}
	rutaSesion := func(agentFilter string) string {
		q := `
			SELECT cwd
			FROM sesiones
			WHERE proyecto_id = ?
			  AND trim(cwd) <> ''`
		args := []any{proyectoID}
		if strings.TrimSpace(agentFilter) != "" {
			q += ` AND agente = ?`
			args = append(args, strings.TrimSpace(agentFilter))
		}
		q += `
			ORDER BY activa DESC, id DESC
			LIMIT 20`
		rows, err := queryFn(q, args...)
		if err != nil {
			return ""
		}
		defer rows.Close()
		for rows.Next() {
			var cwd string
			if err := rows.Scan(&cwd); err != nil {
				return ""
			}
			cwd = normalizarRutaProyecto(cwd)
			if cwd == "" {
				continue
			}
			insideWorktree := strings.Contains(cwd, string(filepath.Separator)+".orquesta-worktrees"+string(filepath.Separator))
			if ruta := coordinacion.CandidateEffectiveProjectPath(cwd, insideWorktree, tieneMarcadoresRepo); ruta != "" {
				return ruta
			}
		}
		return ""
	}
	return rutaSesion(agenteCanonico)
}

func rutaWorktreeActivaPrepareLite(proyectoID int64, agente, rutaProyecto string) string {
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return rutaWorktreeActivaPrepareLiteMainDB(proyectoID, agente, rutaProyecto)
	}
	driver := normalizedDriverName(cfg.Driver)
	if !supportsPrepareLiteReadOnlyBackend(driver) {
		return rutaWorktreeActivaPrepareLiteMainDB(proyectoID, agente, rutaProyecto)
	}
	disabled := false
	skipPost := true
	cfg = applyOpenOptions(cfg, OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
		ReadOnly:           true,
	})
	cfg.MaxOpenConns = 1
	raw, err := backend.Open(cfg)
	if err != nil {
		return rutaWorktreeActivaPrepareLiteMainDB(proyectoID, agente, rutaProyecto)
	}
	defer raw.Close()
	return rutaWorktreeActivaPrepareLiteWithQueryRow(
		func(q string, args ...any) scanner {
			return raw.QueryRow(storage.RebindQuery(driver, q), args...)
		},
		proyectoID,
		agente,
		rutaProyecto,
	)
}

func rutaWorktreeActivaPrepareLiteMainDB(proyectoID int64, agente, rutaProyecto string) string {
	if proyectoID <= 0 || DB == nil {
		return ""
	}
	return rutaWorktreeActivaPrepareLiteWithQueryRow(
		func(q string, args ...any) scanner {
			return DB.QueryRow(q, args...)
		},
		proyectoID,
		agente,
		rutaProyecto,
	)
}

func rutaWorktreeActivaPrepareLiteWithQueryRow(queryRowFn func(string, ...any) scanner, proyectoID int64, agente, rutaProyecto string) string {
	if proyectoID <= 0 || queryRowFn == nil {
		return ""
	}
	q := `SELECT ruta_abs FROM worktrees WHERE proyecto_id = ? AND estado = 'activa'`
	args := []any{proyectoID}
	agenteCanonico, err := CanonicalizeAgentName(agente)
	if err != nil {
		agenteCanonico = strings.TrimSpace(agente)
	}
	agente = strings.TrimSpace(agenteCanonico)
	if agente != "" {
		q += ` AND agente = ?`
		args = append(args, agente)
	}
	q += ` ORDER BY id DESC LIMIT 1`
	var rutaRaw string
	if err := queryRowFn(q, args...).Scan(&rutaRaw); err != nil {
		return ""
	}
	return rutaProyectoEscopadaCanonica(rutaProyecto, rutaRaw)
}

func rutaWorktreeActivaAgenteProyecto(proyectoID int64, agente string) string {
	if proyectoID <= 0 || DB == nil {
		return ""
	}
	rutaProyectoEfectiva := ""
	if proyecto, err := GetProyectoConRutaEfectiva(jsonNumber(proyectoID), ""); err == nil && proyecto != nil {
		rutaProyectoEfectiva = proyecto.RutaAbs
	}
	estado := coordinacion.WorktreeActive
	filter := coordinacion.WorktreeFilter{
		ProjectID: &proyectoID,
		State:     &estado,
	}
	if canonical, err := CanonicalizeAgentName(agente); err == nil {
		agente = canonical
	}
	agente = strings.TrimSpace(agente)
	if agente != "" {
		filter.Agent = &agente
	}
	worktrees, err := ListarWorktreesCoord(filter)
	if err != nil {
		return ""
	}
	refs := make([]coordinacion.WorktreePathRef, 0, len(worktrees))
	for _, worktree := range worktrees {
		if worktree == nil {
			continue
		}
		refs = append(refs, coordinacion.WorktreePathRef{
			Agent: strings.TrimSpace(worktree.Agent),
			Path:  strings.TrimSpace(worktree.Path),
		})
	}
	return coordinacion.SelectUsableActiveWorktreePath(refs, rutaProyectoEfectiva)
}

func rutaSesionPerteneceAWorktreeActiva(proyectoID int64, agente, cwd string) bool {
	return rutaSesionPerteneceAWorktreeActivaConRutaProyecto(proyectoID, agente, cwd, "")
}

func rutaSesionPerteneceAWorktreeActivaConRutaProyecto(proyectoID int64, agente, cwd, rutaProyectoRaw string) bool {
	if proyectoID <= 0 || DB == nil {
		return false
	}
	rutaProyectoRaw = normalizarRutaProyecto(rutaProyectoRaw)
	if rutaProyectoRaw == "" {
		rutaProyectoRaw = rutaProyectoWorktreeRaw(proyectoID)
	}
	q := `
		SELECT ruta_abs
		FROM worktrees
		WHERE proyecto_id = ?
		  AND estado = 'activa'`
	args := []any{proyectoID}
	agenteCanonico, err := CanonicalizeAgentName(agente)
	if err != nil {
		agenteCanonico = strings.TrimSpace(agente)
	}
	if strings.TrimSpace(agenteCanonico) != "" {
		q += ` AND agente = ?`
		args = append(args, strings.TrimSpace(agenteCanonico))
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return false
	}
	defer rows.Close()
	worktrees := make([]coordinacion.WorktreePathRef, 0)
	for rows.Next() {
		var worktreePath string
		if err := rows.Scan(&worktreePath); err != nil {
			return false
		}
		worktrees = append(worktrees, coordinacion.WorktreePathRef{
			Agent: strings.TrimSpace(agenteCanonico),
			Path:  rutaProyectoEscopadaCanonica(rutaProyectoRaw, worktreePath),
		})
	}
	return coordinacion.SessionPathInsideActiveWorktree(strings.TrimSpace(agenteCanonico), cwd, worktrees)
}

func rutaSesionPerteneceAWorktreeActivaConContexto(proyectoID int64, agente, cwd string, worktrees []coordinacion.WorktreePathRef, rutaProyectoRaw string) bool {
	if len(worktrees) > 0 {
		return coordinacion.SessionPathInsideActiveWorktree(agente, cwd, worktrees)
	}
	return rutaSesionPerteneceAWorktreeActivaConRutaProyecto(proyectoID, agente, cwd, rutaProyectoRaw)
}

func ResolveProyectoIDBySlug(slug string) (*int64, error) {
	proyecto, err := GetProyecto(slug)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, nil
	}
	return &proyecto.ID, nil
}

func ListarProyectos(f FiltroProyectos) ([]*Proyecto, error) {
	q := `
		SELECT id, slug, nombre, ruta_abs, origen_repo, remote_url, branch_base, tipo, parent_id, activo, created_at, updated_at
		FROM proyectos WHERE 1=1`
	var args []any
	if f.Tipo != nil {
		q += ` AND tipo = ?`
		args = append(args, *f.Tipo)
	}
	if f.ParentID != nil {
		q += ` AND parent_id = ?`
		args = append(args, *f.ParentID)
	}
	if f.Activo != nil {
		q += ` AND activo = ?`
		args = append(args, boolToInt(*f.Activo))
	}
	q += ` ORDER BY ruta_abs`

	return consultarConReintentos(func() ([]*Proyecto, error) {
		rows, err := DB.Query(q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var list []*Proyecto
		for rows.Next() {
			p, err := escanearProyecto(rows)
			if err != nil {
				return nil, err
			}
			list = append(list, p)
		}
		return list, rows.Err()
	})
}

func ListarProyectosConRutaEfectiva(f FiltroProyectos, cwdHint string) ([]*Proyecto, error) {
	proyectos, err := ListarProyectos(f)
	if err != nil {
		return nil, err
	}
	return ProyectosConRutaEfectiva(proyectos, cwdHint), nil
}

func cargarContextoRutaProyectos(proyectos []*Proyecto) (map[int64]*proyectoRutaSesion, map[int64][]coordinacion.WorktreePathRef) {
	sesionesPorProyecto := map[int64]*proyectoRutaSesion{}
	worktreesPorProyecto := map[int64][]coordinacion.WorktreePathRef{}
	if len(proyectos) == 0 || DB == nil {
		return sesionesPorProyecto, worktreesPorProyecto
	}
	ids := make([]int64, 0, len(proyectos))
	vistos := make(map[int64]struct{}, len(proyectos))
	rutasProyecto := make(map[int64]string, len(proyectos))
	for _, proyecto := range proyectos {
		id := proyectoIDOrZero(proyecto)
		if id <= 0 {
			continue
		}
		if _, ok := vistos[id]; ok {
			continue
		}
		vistos[id] = struct{}{}
		ids = append(ids, id)
		rutasProyecto[id] = normalizarRutaProyecto(proyecto.RutaAbs)
	}
	if len(ids) == 0 {
		return sesionesPorProyecto, worktreesPorProyecto
	}

	placeholders := joinInt64Placeholders(ids)
	args := int64Args(ids)
	if placeholders == "" || len(args) == 0 {
		return sesionesPorProyecto, worktreesPorProyecto
	}

	rows, err := DB.Query(`
		SELECT proyecto_id, agente, cwd
		FROM sesiones
		WHERE proyecto_id IN (`+placeholders+`)
		  AND trim(cwd) <> ''
		ORDER BY proyecto_id, activa DESC, id DESC`, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var (
				proyectoID int64
				agente     string
				cwd        string
			)
			if scanErr := rows.Scan(&proyectoID, &agente, &cwd); scanErr != nil {
				break
			}
			if _, ok := sesionesPorProyecto[proyectoID]; ok {
				continue
			}
			sesionesPorProyecto[proyectoID] = &proyectoRutaSesion{
				agente: strings.TrimSpace(agente),
				cwd:    strings.TrimSpace(cwd),
			}
		}
	}

	worktreeRows, err := DB.Query(`
		SELECT proyecto_id, agente, ruta_abs
		FROM worktrees
		WHERE proyecto_id IN (`+placeholders+`)
		  AND estado = 'activa'`, args...)
	if err == nil {
		defer worktreeRows.Close()
		for worktreeRows.Next() {
			var (
				proyectoID int64
				agente     string
				rutaAbs    string
			)
			if scanErr := worktreeRows.Scan(&proyectoID, &agente, &rutaAbs); scanErr != nil {
				break
			}
			worktreesPorProyecto[proyectoID] = append(worktreesPorProyecto[proyectoID], coordinacion.WorktreePathRef{
				Agent: strings.TrimSpace(agente),
				Path:  rutaProyectoEscopadaCanonica(rutasProyecto[proyectoID], rutaAbs),
			})
		}
	}
	return sesionesPorProyecto, worktreesPorProyecto
}

func proyectoIDOrZero(proyecto *Proyecto) int64 {
	if proyecto == nil {
		return 0
	}
	return proyecto.ID
}

func joinInt64Placeholders(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	placeholders := make([]string, 0, len(values))
	for range values {
		placeholders = append(placeholders, "?")
	}
	return strings.Join(placeholders, ",")
}

func int64Args(values []int64) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func DescubrirProyectos(root string) ([]*Proyecto, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		var err error
		root, err = WorkspaceRoot()
		if err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root = filepath.Clean(abs)

	raiz := &Proyecto{
		Slug:    slugProyecto(filepath.Base(root)),
		Nombre:  filepath.Base(root),
		RutaAbs: root,
		Tipo:    ProyectoRaiz,
		Activo:  true,
	}
	id, err := UpsertProyecto(raiz)
	if err != nil {
		return nil, err
	}
	raiz.ID = id

	resultados := []*Proyecto{raiz}
	subdirs, err := listarSubdirectorios(root)
	if err != nil {
		return nil, err
	}
	for _, subdir := range subdirs {
		disc, err := descubrirProyectoRec(subdir, &raiz.ID, false)
		if err != nil {
			return nil, err
		}
		resultados = append(resultados, disc...)
	}

	sort.Slice(resultados, func(i, j int) bool { return resultados[i].RutaAbs < resultados[j].RutaAbs })
	return resultados, nil
}

func WorkspaceRoot() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_WORKSPACE_ROOT")); v != "" {
		return filepath.Abs(v)
	}
	if v, err := ConfigGet("workspace_root"); err == nil && strings.TrimSpace(v) != "" {
		return filepath.Abs(v)
	}
	return "", fmt.Errorf("workspace_root no configurado; usa 'orquesta config set workspace_root <ruta>' o 'orquesta proyecto descubrir <ruta>'")
}

func descubrirProyectoRec(path string, parentID *int64, esRaiz bool) ([]*Proyecto, error) {
	path = filepath.Clean(path)
	subdirs, err := listarSubdirectorios(path)
	if err != nil {
		return nil, err
	}

	var hijosConProyecto []string
	for _, subdir := range subdirs {
		if contieneProyecto(subdir) {
			hijosConProyecto = append(hijosConProyecto, subdir)
		}
	}

	tieneMarcadores, err := tieneMarcadoresRepo(path)
	if err != nil {
		return nil, err
	}

	var tipo TipoProyecto
	switch {
	case esRaiz:
		tipo = ProyectoRaiz
	case len(hijosConProyecto) > 0:
		tipo = ProyectoGrupo
	case tieneMarcadores:
		tipo = ProyectoRepo
	default:
		return nil, nil
	}

	p := &Proyecto{
		Slug:     slugProyecto(filepath.Base(path)),
		Nombre:   filepath.Base(path),
		RutaAbs:  path,
		Tipo:     tipo,
		ParentID: parentID,
		Activo:   true,
	}
	id, err := UpsertProyecto(p)
	if err != nil {
		return nil, err
	}
	p.ID = id

	resultados := []*Proyecto{p}
	if tipo == ProyectoRepo {
		return resultados, nil
	}

	for _, hijo := range hijosConProyecto {
		disc, err := descubrirProyectoRec(hijo, &id, false)
		if err != nil {
			return nil, err
		}
		resultados = append(resultados, disc...)
	}
	return resultados, nil
}

func listarSubdirectorios(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		nombre := entry.Name()
		if strings.HasPrefix(nombre, ".") {
			continue
		}
		if nombre == "node_modules" || nombre == "vendor" || nombre == "dist" || nombre == "build" || nombre == "tmp" {
			continue
		}
		out = append(out, filepath.Join(path, nombre))
	}
	sort.Strings(out)
	return out, nil
}

func contieneProyecto(path string) bool {
	ok, err := tieneMarcadoresRepo(path)
	if err == nil && ok {
		return true
	}
	subdirs, err := listarSubdirectorios(path)
	if err != nil {
		return false
	}
	for _, subdir := range subdirs {
		ok, err := tieneMarcadoresRepo(subdir)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func tieneMarcadoresRepo(path string) (bool, error) {
	marcadores := []string{
		".git",
		"go.mod",
		"package.json",
		"pyproject.toml",
		"Cargo.toml",
		"Makefile",
	}
	for _, marcador := range marcadores {
		info, err := os.Stat(filepath.Join(path, marcador))
		if err == nil {
			if marcador == ".git" || !info.IsDir() {
				return true, nil
			}
		}
	}
	return false, nil
}

func slugProyecto(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "proyecto"
	}
	var b strings.Builder
	ultimoGuion := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			ultimoGuion = false
			continue
		}
		if !ultimoGuion {
			b.WriteRune('-')
			ultimoGuion = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "proyecto"
	}
	return out
}

func escanearProyecto(s scanner) (*Proyecto, error) {
	var p Proyecto
	var parentID sql.NullInt64
	err := s.Scan(&p.ID, &p.Slug, &p.Nombre, &p.RutaAbs, &p.OrigenRepo, &p.RemoteURL, &p.BranchBase, &p.Tipo, &parentID, &p.Activo, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		p.ParentID = &parentID.Int64
	}
	if strings.TrimSpace(p.OrigenRepo) == "" {
		p.OrigenRepo = "local"
	}
	return &p, nil
}

// ListarProyectosActivos devuelve todos los proyectos marcados como activos.
func ListarProyectosActivos() ([]*Proyecto, error) {
	rows, err := DB.Query(`SELECT id, slug, nombre, ruta_abs, origen_repo, remote_url, branch_base, tipo, parent_id, activo, created_at, updated_at FROM proyectos WHERE activo = 1 ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Proyecto
	for rows.Next() {
		p, err := escanearProyecto(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// GetEstadisticasProyecto calcula el recuento de tareas por estado para un proyecto.
func GetEstadisticasProyecto(proyectoID int64) (EstadisticasProyecto, error) {
	var stats EstadisticasProyecto
	rows, err := DB.Query(`SELECT estado, COUNT(*) FROM tareas WHERE proyecto_id = ? GROUP BY estado`, proyectoID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var estado string
		var count int
		if err := rows.Scan(&estado, &count); err != nil {
			return stats, err
		}
		stats.Total += count
		switch estado {
		case "completada":
			stats.Completadas = count
		case "en_progreso":
			stats.EnProgreso = count
		case "asignada":
			stats.Asignadas = count
		case "bloqueada":
			stats.Bloqueadas = count
		}
	}
	return stats, nil
}

// GetEstadoGit devuelve un resumen corto de cambios pendientes en el repo.
func GetEstadoGit(ruta string) (string, error) {
	cmd := exec.Command("git", "status", "--short")
	cmd.Dir = ruta
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
