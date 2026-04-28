/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/supervisionapp"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Orquesta revisión y mejora de un repositorio/proyecto por la vía canónica",
}

type repoAddResult struct {
	Proyecto      *db.Proyecto
	DiscoveryRoot string
	RutaAbs       string
	RemoteURL     string
	BranchBase    string
}

var errRepoBadRequest = errors.New("repo bad request")

func repoBadRequestf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errRepoBadRequest, fmt.Sprintf(format, args...))
}

func imprimirResumenRepoMaterializado(resultado *repoAddResult) {
	if resultado == nil || resultado.Proyecto == nil {
		return
	}
	if resultado.RemoteURL != "" {
		fmt.Printf("✓ Repo remoto materializado desde %s\n", resultado.RemoteURL)
	} else {
		fmt.Printf("✓ Repo local materializado desde %s\n", resultado.RutaAbs)
	}
	fmt.Printf("Proyecto: #%d %s (%s)\n", resultado.Proyecto.ID, resultado.Proyecto.Slug, resultado.Proyecto.Tipo)
	fmt.Printf("Ruta: %s\n", resultado.Proyecto.RutaAbs)
	fmt.Printf("Origen: %s\n", emptyDash(resultado.Proyecto.OrigenRepo))
	if strings.TrimSpace(resultado.Proyecto.RemoteURL) != "" {
		fmt.Printf("Remote URL: %s\n", resultado.Proyecto.RemoteURL)
	}
	if strings.TrimSpace(resultado.Proyecto.BranchBase) != "" {
		fmt.Printf("Branch base: %s\n", resultado.Proyecto.BranchBase)
	}
	fmt.Printf("Discovery root: %s\n", emptyDash(resultado.DiscoveryRoot))
}

var repoAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adjunta un repo local o clona uno remoto y lo materializa como proyecto canónico",
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath, _ := cmd.Flags().GetString("path")
		remoteURL, _ := cmd.Flags().GetString("git")
		branch, _ := cmd.Flags().GetString("branch")
		destino, _ := cmd.Flags().GetString("destino")

		resultado, err := repoAddViaAPI(strings.TrimSpace(localPath), strings.TrimSpace(remoteURL), strings.TrimSpace(branch), strings.TrimSpace(destino))
		if err != nil {
			return err
		}
		if resultado == nil || resultado.Proyecto == nil {
			return fmt.Errorf("alta de repo sin proyecto materializado")
		}
		imprimirResumenRepoMaterializado(resultado)
		return nil
	},
}

var repoRevisarCmd = &cobra.Command{
	Use:   "revisar",
	Short: "Revisa un repositorio/proyecto usando el pipeline local existente",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, materializado, err := resolverProyectoRepoDesdeFlags(cmd, "repo revisar")
		if err != nil {
			return err
		}
		if materializado != nil {
			imprimirResumenRepoMaterializado(materializado)
		}
		planOnly, _ := cmd.Flags().GetBool("plan")
		resp, ok, err := repoRevisarViaAPI(apiRepoRevisarRequest{
			Proyecto: proyecto,
			Plan:     planOnly,
		})
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("repo revisar")
		}
		if planOnly {
			paso := (*capacidadapp.PasoPipelineLocalDeterminista)(nil)
			if resp != nil {
				paso = resp.Paso
			}
			imprimirResumenRepoPaso(paso)
			return nil
		}
		var resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal
		if resp != nil {
			resultado = resp.Resultado
		}
		imprimirResumenRepoResultado(resultado)
		return nil
	},
}

var repoMejorarCmd = &cobra.Command{
	Use:   "mejorar",
	Short: "Siembra una mejora como tarea y la engancha al pipeline del repositorio/proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, materializado, err := resolverProyectoRepoDesdeFlags(cmd, "repo mejorar")
		if err != nil {
			return err
		}
		if materializado != nil {
			imprimirResumenRepoMaterializado(materializado)
		}
		titulo, _ := cmd.Flags().GetString("titulo")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		modulo, _ := cmd.Flags().GetString("modulo")
		prioridad, _ := cmd.Flags().GetString("prioridad")
		creadoPor, _ := cmd.Flags().GetString("por")
		notas, _ := cmd.Flags().GetString("notas")
		funcionObjetivo, _ := cmd.Flags().GetString("funcion")
		writeSetRaw, _ := cmd.Flags().GetString("write-set")
		modelosRaw, _ := cmd.Flags().GetString("modelos")
		preservarArquitectura, _ := cmd.Flags().GetBool("preservar-arquitectura")
		finishApp, _ := cmd.Flags().GetBool("finish-app")
		autonomiaPersistente, _ := cmd.Flags().GetBool("autonomia-persistente")
		supervisorAgente, _ := cmd.Flags().GetString("supervisor")
		reviewerAgente, _ := cmd.Flags().GetString("reviewer")
		maxWorkers, _ := cmd.Flags().GetInt("max-workers")
		despachar, _ := cmd.Flags().GetBool("despachar")

		if strings.TrimSpace(titulo) == "" {
			return fmt.Errorf("--titulo es obligatorio")
		}
		resp, ok, err := repoMejorarViaAPI(apiRepoMejorarRequest{
			Proyecto:              proyecto,
			Titulo:                strings.TrimSpace(titulo),
			Descripcion:           strings.TrimSpace(descripcion),
			Modulo:                strings.TrimSpace(modulo),
			Prioridad:             strings.TrimSpace(prioridad),
			CreadoPor:             strings.TrimSpace(creadoPor),
			Notas:                 strings.TrimSpace(notas),
			FuncionObjetivo:       strings.TrimSpace(funcionObjetivo),
			WriteSet:              splitCSV(writeSetRaw),
			ModelosCandidatos:     splitCSV(modelosRaw),
			PreservarArquitectura: preservarArquitectura,
			FinishApp:             finishApp,
			AutonomiaPersistente:  autonomiaPersistente,
			SupervisorAgente:      strings.TrimSpace(supervisorAgente),
			ReviewerAgente:        strings.TrimSpace(reviewerAgente),
			MaxWorkers:            maxWorkers,
			Despachar:             repoBoolPtr(despachar),
		})
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("repo mejorar")
		}
		if resp != nil && resp.Tarea != nil {
			fmt.Printf("✓ Tarea #%d creada para %s\n", resp.Tarea.ID, proyecto)
		} else {
			fmt.Printf("✓ Tarea creada para %s\n", proyecto)
		}
		imprimirResumenRepoFork(resp.Fork)
		imprimirResumenRepoPolicy(resp.Policy)
		if !despachar || resp == nil {
			return nil
		}
		imprimirResumenRepoResultado(resp.Resultado)
		return nil
	},
}

func imprimirResumenRepoPaso(paso *capacidadapp.PasoPipelineLocalDeterminista) {
	if paso == nil {
		fmt.Println("No hay paso disponible.")
		return
	}
	fmt.Printf("Accion: %s\n", paso.Accion)
	fmt.Printf("Motivo: %s\n", paso.Motivo)
	if paso.FaseActual != "" {
		fmt.Printf("Fase actual: %s\n", paso.FaseActual)
	}
	if paso.FaseObjetivo != "" {
		fmt.Printf("Fase objetivo: %s\n", paso.FaseObjetivo)
	}
	if paso.TareaObjetivo != nil {
		fmt.Printf("Tarea: #%d %s (%s)\n", paso.TareaObjetivo.ID, paso.TareaObjetivo.Titulo, paso.TareaObjetivo.Estado)
	}
	if paso.EtapaObjetivo != nil {
		fmt.Printf("Carril: %s\n", emptyDash(paso.EtapaObjetivo.Carril))
		fmt.Printf("Entrega: %s\n", emptyDash(paso.EtapaObjetivo.EntregaCanonica))
	}
	if paso.GateBloqueante != nil {
		fmt.Printf("Gate: %d (%s)\n", paso.GateBloqueante.ID, paso.GateBloqueante.Estado)
	}
}

func resolverProyectoRepoDesdeFlags(cmd *cobra.Command, commandName string) (string, *repoAddResult, error) {
	proyecto, _ := cmd.Flags().GetString("proyecto")
	localPath, _ := cmd.Flags().GetString("path")
	remoteURL, _ := cmd.Flags().GetString("git")
	branch, _ := cmd.Flags().GetString("branch")
	destino, _ := cmd.Flags().GetString("destino")

	proyecto = strings.TrimSpace(proyecto)
	localPath = strings.TrimSpace(localPath)
	remoteURL = strings.TrimSpace(remoteURL)
	branch = strings.TrimSpace(branch)
	destino = strings.TrimSpace(destino)

	if proyecto != "" {
		if localPath != "" || remoteURL != "" {
			return "", nil, fmt.Errorf("usa --proyecto o bien --path/--git, pero no ambos")
		}
		return proyecto, nil, nil
	}
	if localPath == "" && remoteURL == "" {
		return "", nil, fmt.Errorf("--proyecto es obligatorio salvo que uses --path o --git")
	}
	resultado, err := repoAddViaAPI(localPath, remoteURL, branch, destino)
	if err != nil {
		return "", nil, err
	}
	if resultado == nil || resultado.Proyecto == nil || strings.TrimSpace(resultado.Proyecto.Slug) == "" {
		return "", nil, fmt.Errorf("%s sin proyecto materializado", strings.TrimSpace(commandName))
	}
	return strings.TrimSpace(resultado.Proyecto.Slug), resultado, nil
}

func imprimirResumenRepoResultado(resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal) {
	if resultado == nil || resultado.Paso == nil {
		fmt.Println("No hay paso ejecutable.")
		return
	}
	imprimirResumenRepoPaso(resultado.Paso)
	if resultado.FaseActivada != nil {
		fmt.Printf("Fase activada: %s\n", *resultado.FaseActivada)
	}
	if resultado.TareaActualizada != nil {
		fmt.Printf("Tarea actualizada: #%d %s (%s)\n", resultado.TareaActualizada.ID, resultado.TareaActualizada.Titulo, resultado.TareaActualizada.Estado)
	}
	if resultado.Despacho != nil {
		fmt.Printf("Agente sugerido: %s\n", emptyDash(resultado.Despacho.AgenteSugerido))
		imprimirModoDespachoCLI(resultado.Despacho)
		imprimirSeleccionAgenteCLI(resultado.Despacho.SeleccionAgente)
	}
	if resultado.DispatchRuntime != nil {
		fmt.Printf("Estado dispatch: %s\n", emptyDash(resultado.DispatchRuntime.Estado))
		if motivo := strings.TrimSpace(resultado.DispatchRuntime.Motivo); motivo != "" {
			fmt.Printf("Motivo dispatch: %s\n", motivo)
		}
	}
}

func imprimirResumenRepoFork(fork *apiRepoFunctionForkSpec) {
	if fork == nil {
		return
	}
	if target := strings.TrimSpace(fork.FuncionObjetivo); target != "" {
		fmt.Printf("Fork función: %s\n", target)
	}
	if materia := strings.TrimSpace(fork.Materia); materia != "" {
		fmt.Printf("Materia fork: %s\n", materia)
	}
	if fork.ForkLines > 0 {
		fmt.Printf("Líneas fork: %d\n", fork.ForkLines)
	}
	if len(fork.SelectedModels) > 0 {
		fmt.Printf("Modelos elegidos: %s\n", strings.Join(fork.SelectedModels, ", "))
	} else if len(fork.ModelosCandidatos) > 0 {
		fmt.Printf("Modelos sugeridos: %s\n", strings.Join(fork.ModelosCandidatos, ", "))
	}
	if fork.PreservarArquitectura {
		fmt.Printf("Restricción fork: preservar arquitectura\n")
	}
	if reason := strings.TrimSpace(fork.DecisionReason); reason != "" {
		fmt.Printf("Motivo fork: %s\n", reason)
	}
}

func imprimirResumenRepoPolicy(policy *supervisionapp.Policy) {
	if policy == nil || !policy.Enabled {
		return
	}
	fmt.Printf("Autonomía persistente: activa\n")
	if supervisor := strings.TrimSpace(policy.SupervisorAgente); supervisor != "" {
		fmt.Printf("Supervisor Codex: %s\n", supervisor)
	}
	if reviewer := strings.TrimSpace(policy.ReviewerAgente); reviewer != "" {
		fmt.Printf("Reviewer Codex: %s\n", reviewer)
	}
	if policy.MaxWorkers > 0 {
		fmt.Printf("Workers máximos: %d\n", policy.MaxWorkers)
	}
}

func repoAddViaAPI(localPath, remoteURL, branch, destino string) (*repoAddResult, error) {
	var resp apiRepoMaterializarResponse
	ok, err := apiPost("/api/repos/materializar", apiRepoMaterializarRequest{
		Path:    strings.TrimSpace(localPath),
		Git:     strings.TrimSpace(remoteURL),
		Branch:  strings.TrimSpace(branch),
		Destino: strings.TrimSpace(destino),
	}, &resp)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, serverFirstCommandError("repo add")
	}
	if !resp.OK || resp.Proyecto == nil {
		return nil, fmt.Errorf("alta de repo sin proyecto materializado")
	}
	return &repoAddResult{
		Proyecto:      resp.Proyecto,
		DiscoveryRoot: resp.DiscoveryRoot,
		RutaAbs:       resp.RutaAbs,
		RemoteURL:     resp.RemoteURL,
		BranchBase:    resp.BranchBase,
	}, nil
}

func repoRevisarViaAPI(req apiRepoRevisarRequest) (*apiRepoRevisarResponse, bool, error) {
	var resp apiRepoRevisarResponse
	ok, err := apiPost("/api/repos/revisar", req, &resp)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &resp, true, nil
}

func repoMejorarViaAPI(req apiRepoMejorarRequest) (*apiRepoMejorarResponse, bool, error) {
	var resp apiRepoMejorarResponse
	ok, err := apiPost("/api/repos/mejorar", req, &resp)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &resp, true, nil
}

func repoBoolPtr(v bool) *bool {
	return &v
}

func repoAddServer(localPath, remoteURL, branch, destino string) (*repoAddResult, error) {
	hasLocal := strings.TrimSpace(localPath) != ""
	hasRemote := strings.TrimSpace(remoteURL) != ""
	switch {
	case hasLocal == hasRemote:
		return nil, repoBadRequestf("usa exactamente uno de path o git")
	case hasLocal:
		repoRoot, err := resolverRepoRootLocal(localPath)
		if err != nil {
			return nil, err
		}
		branchBase, err := resolverBranchGit(repoRoot)
		if err != nil {
			return nil, err
		}
		workspaceRoot, workspaceConfigured, err := cargarWorkspaceRootServer()
		if err != nil {
			return nil, err
		}
		discoveryRoot := resolverDiscoveryRootRepo(repoRoot, workspaceRoot)
		proyecto, err := descubrirYMaterializarRepoServer(discoveryRoot, repoRoot, workspaceRoot, workspaceConfigured, apiProyectoActualizarRequest{
			OrigenRepo: "local",
			RemoteURL:  "",
			BranchBase: branchBase,
			Tipo:       string(db.ProyectoRepo),
		})
		if err != nil {
			return nil, err
		}
		return &repoAddResult{
			Proyecto:      proyecto,
			DiscoveryRoot: discoveryRoot,
			RutaAbs:       repoRoot,
			BranchBase:    branchBase,
		}, nil
	default:
		workspaceRoot, workspaceConfigured, err := cargarWorkspaceRootServer()
		if err != nil {
			return nil, err
		}
		repoRoot, discoveryRoot, branchBase, err := clonarRepoRemotoParaMaterializar(remoteURL, branch, destino, workspaceRoot)
		if err != nil {
			return nil, err
		}
		proyecto, err := descubrirYMaterializarRepoServer(discoveryRoot, repoRoot, workspaceRoot, workspaceConfigured, apiProyectoActualizarRequest{
			OrigenRepo: "git",
			RemoteURL:  remoteURL,
			BranchBase: branchBase,
			Tipo:       string(db.ProyectoRepo),
		})
		if err != nil {
			return nil, err
		}
		return &repoAddResult{
			Proyecto:      proyecto,
			DiscoveryRoot: discoveryRoot,
			RutaAbs:       repoRoot,
			RemoteURL:     remoteURL,
			BranchBase:    branchBase,
		}, nil
	}
}

func cargarWorkspaceRootServer() (string, bool, error) {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_WORKSPACE_ROOT")); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", false, err
		}
		return filepath.Clean(abs), true, nil
	}
	v, err := db.ConfigGet("workspace_root")
	if err != nil || strings.TrimSpace(v) == "" {
		return "", false, nil
	}
	abs, err := filepath.Abs(strings.TrimSpace(v))
	if err != nil {
		return "", false, err
	}
	return filepath.Clean(abs), true, nil
}

func descubrirYMaterializarRepoServer(discoveryRoot, repoRoot, workspaceRoot string, restoreWorkspace bool, update apiProyectoActualizarRequest) (*db.Proyecto, error) {
	proyectos, err := db.DescubrirProyectos(discoveryRoot)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(discoveryRoot) != "" {
		if abs, err := filepath.Abs(discoveryRoot); err == nil {
			_ = configService.Set("workspace_root", abs)
		}
	}
	if restoreWorkspace && strings.TrimSpace(workspaceRoot) != "" && filepath.Clean(discoveryRoot) != filepath.Clean(workspaceRoot) {
		_ = configService.Set("workspace_root", workspaceRoot)
	}
	proyecto := seleccionarProyectoPorRuta(proyectos, repoRoot)
	if proyecto == nil {
		return nil, fmt.Errorf("no se encontró proyecto materializado para %s tras descubrir %s", repoRoot, discoveryRoot)
	}
	actualizado := *proyecto
	if v := strings.TrimSpace(update.OrigenRepo); v != "" {
		actualizado.OrigenRepo = v
	}
	actualizado.RemoteURL = strings.TrimSpace(update.RemoteURL)
	actualizado.BranchBase = strings.TrimSpace(update.BranchBase)
	if v := strings.TrimSpace(update.Tipo); v != "" {
		actualizado.Tipo = db.TipoProyecto(v)
	}
	if err := db.UpdateProyecto(&actualizado); err != nil {
		return nil, err
	}
	return db.GetProyecto(fmt.Sprintf("%d", actualizado.ID))
}

func resolverProyectoRepoServer(proyecto, localPath, remoteURL, branch, destino string) (*db.Proyecto, *repoAddResult, error) {
	proyecto = strings.TrimSpace(proyecto)
	localPath = strings.TrimSpace(localPath)
	remoteURL = strings.TrimSpace(remoteURL)
	branch = strings.TrimSpace(branch)
	destino = strings.TrimSpace(destino)

	if proyecto != "" {
		if localPath != "" || remoteURL != "" {
			return nil, nil, repoBadRequestf("usa proyecto o bien path/git, pero no ambos")
		}
		item, err := apiGetProyectoTimeboxed(proyecto)
		if err != nil {
			return nil, nil, err
		}
		if item == nil {
			return nil, nil, repoBadRequestf("proyecto no encontrado: %s", proyecto)
		}
		return item, nil, nil
	}
	if localPath == "" && remoteURL == "" {
		return nil, nil, repoBadRequestf("proyecto es obligatorio salvo que uses path o git")
	}
	resultado, err := repoAddServer(localPath, remoteURL, branch, destino)
	if err != nil {
		return nil, nil, err
	}
	if resultado == nil || resultado.Proyecto == nil {
		return nil, nil, fmt.Errorf("materialización sin proyecto")
	}
	return resultado.Proyecto, resultado, nil
}

func resolverRepoRootLocal(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", repoBadRequestf("--path es obligatorio")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", repoBadRequestf("ruta local no válida como repo git: %s", strings.TrimSpace(string(out)))
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func cargarWorkspaceRootDesdeAPI() (string, bool, error) {
	resp, ok, err := cargarConfigDesdeAPI("")
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, serverFirstCommandError("repo add")
	}
	if resp == nil || resp.Config == nil {
		return "", false, nil
	}
	root := strings.TrimSpace(resp.Config["workspace_root"])
	if root == "" {
		return "", false, nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", false, err
	}
	return filepath.Clean(abs), true, nil
}

func resolverDiscoveryRootRepo(repoRoot, workspaceRoot string) string {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	workspaceRoot = filepath.Clean(strings.TrimSpace(workspaceRoot))
	if repoRoot == "" {
		return ""
	}
	if workspaceRoot != "" && repoRoot != workspaceRoot && pathDentroDeRaiz(workspaceRoot, repoRoot) {
		return workspaceRoot
	}
	parent := filepath.Dir(repoRoot)
	if parent != "" && parent != repoRoot {
		return parent
	}
	return repoRoot
}

func descubrirYMaterializarRepoPorAPI(discoveryRoot, repoRoot, workspaceRoot string, restoreWorkspace bool, update apiProyectoActualizarRequest) (*db.Proyecto, error) {
	proyectos, ok, err := descubrirProyectosPorAPI(discoveryRoot)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, serverFirstCommandError("repo add")
	}
	if restoreWorkspace && strings.TrimSpace(workspaceRoot) != "" && filepath.Clean(discoveryRoot) != filepath.Clean(workspaceRoot) {
		if ok, err := configurarValorPorAPI("workspace_root", workspaceRoot); err != nil {
			return nil, err
		} else if !ok {
			return nil, serverFirstCommandError("repo add")
		}
	}
	proyecto := seleccionarProyectoPorRuta(proyectos, repoRoot)
	if proyecto == nil {
		return nil, fmt.Errorf("no se encontró proyecto materializado para %s tras descubrir %s", repoRoot, discoveryRoot)
	}
	recargado, ok, err := actualizarProyectoPorAPI(fmt.Sprintf("%d", proyecto.ID), update)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, serverFirstCommandError("repo add")
	}
	if recargado != nil {
		return recargado, nil
	}
	return proyecto, nil
}

func seleccionarProyectoPorRuta(proyectos []*db.Proyecto, rutaAbs string) *db.Proyecto {
	target := filepath.Clean(strings.TrimSpace(rutaAbs))
	for _, proyecto := range proyectos {
		if proyecto == nil {
			continue
		}
		if filepath.Clean(strings.TrimSpace(proyecto.RutaAbs)) == target {
			return proyecto
		}
	}
	return nil
}

func clonarRepoRemotoParaMaterializar(remoteURL, branch, destino, workspaceRoot string) (string, string, string, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", "", "", repoBadRequestf("--git es obligatorio")
	}
	destino = strings.TrimSpace(destino)
	var clonePath string
	switch {
	case destino != "":
		abs, err := filepath.Abs(destino)
		if err != nil {
			return "", "", "", err
		}
		clonePath = filepath.Clean(abs)
	case strings.TrimSpace(workspaceRoot) == "":
		return "", "", "", repoBadRequestf("workspace_root no configurado; usa 'orquesta config set workspace_root <ruta>' o pasa --destino")
	default:
		clonePath = filepath.Join(workspaceRoot, inferirNombreRepoRemoto(remoteURL))
	}
	if info, err := os.Stat(clonePath); err == nil && info != nil {
		return "", "", "", repoBadRequestf("el destino ya existe: %s", clonePath)
	}
	if err := os.MkdirAll(filepath.Dir(clonePath), 0o755); err != nil {
		return "", "", "", err
	}
	args := []string{"clone"}
	if strings.TrimSpace(branch) != "" {
		args = append(args, "--branch", strings.TrimSpace(branch), "--single-branch")
	}
	args = append(args, remoteURL, clonePath)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", "", fmt.Errorf("git clone: %w: %s", err, strings.TrimSpace(string(out)))
	}
	branchBase, err := resolverBranchGit(clonePath)
	if err != nil {
		return "", "", "", err
	}
	return clonePath, resolverDiscoveryRootRepo(clonePath, workspaceRoot), branchBase, nil
}

func resolverBranchGit(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolver branch base: %w: %s", err, strings.TrimSpace(string(out)))
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", nil
	}
	return branch, nil
}

func inferirNombreRepoRemoto(remoteURL string) string {
	raw := strings.TrimSpace(remoteURL)
	if idx := strings.Index(raw, "://"); idx >= 0 {
		partes := strings.Split(strings.TrimPrefix(raw[idx+3:], "/"), "/")
		if len(partes) > 0 {
			raw = partes[len(partes)-1]
		}
	} else if idx := strings.LastIndex(raw, ":"); idx >= 0 && strings.Contains(raw[:idx], "@") {
		raw = raw[idx+1:]
	}
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "/")
	raw = filepath.Base(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" || raw == "." || raw == string(filepath.Separator) {
		return "repo"
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(strings.TrimSpace(b.String()), "-.")
	if out == "" {
		return "repo"
	}
	return out
}

func pathDentroDeRaiz(root, path string) bool {
	root = filepath.Clean(strings.TrimSpace(root))
	path = filepath.Clean(strings.TrimSpace(path))
	if root == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func init() {
	repoAddCmd.Flags().String("path", "", "Ruta local al repo git a registrar")
	repoAddCmd.Flags().String("git", "", "URL o remote git a clonar y materializar")
	repoAddCmd.Flags().String("branch", "", "Rama inicial para el clone remoto")
	repoAddCmd.Flags().String("destino", "", "Ruta destino local para el clone remoto")

	repoRevisarCmd.Flags().String("proyecto", "", "Slug o referencia del proyecto/repositorio")
	repoRevisarCmd.Flags().String("path", "", "Ruta local al repo git a materializar y revisar")
	repoRevisarCmd.Flags().String("git", "", "URL o remote git a clonar, materializar y revisar")
	repoRevisarCmd.Flags().String("branch", "", "Rama inicial para el clone remoto")
	repoRevisarCmd.Flags().String("destino", "", "Ruta destino local para el clone remoto")
	repoRevisarCmd.Flags().Bool("plan", false, "Solo calcula el siguiente paso; no lo despacha")

	repoMejorarCmd.Flags().String("proyecto", "", "Slug o referencia del proyecto/repositorio")
	repoMejorarCmd.Flags().String("path", "", "Ruta local al repo git a materializar antes de sembrar la mejora")
	repoMejorarCmd.Flags().String("git", "", "URL o remote git a clonar y materializar antes de sembrar la mejora")
	repoMejorarCmd.Flags().String("branch", "", "Rama inicial para el clone remoto")
	repoMejorarCmd.Flags().String("destino", "", "Ruta destino local para el clone remoto")
	repoMejorarCmd.Flags().String("titulo", "", "Título de la mejora")
	repoMejorarCmd.Flags().String("descripcion", "", "Descripción de la mejora")
	repoMejorarCmd.Flags().String("modulo", "", "Módulo afectado")
	repoMejorarCmd.Flags().String("prioridad", "media", "Prioridad de la tarea semilla")
	repoMejorarCmd.Flags().String("por", "repo_orchestrator", "Autor de la tarea semilla")
	repoMejorarCmd.Flags().String("notas", "", "Notas o contexto adicional")
	repoMejorarCmd.Flags().String("funcion", "", "Función o símbolo objetivo para un fork de función")
	repoMejorarCmd.Flags().String("write-set", "", "Lista CSV del write-set permitido para el fork de función")
	repoMejorarCmd.Flags().String("modelos", "", "Lista CSV de modelos candidatos para variantes del fork")
	repoMejorarCmd.Flags().Bool("preservar-arquitectura", true, "Preserva explícitamente la arquitectura canónica del proyecto")
	repoMejorarCmd.Flags().Bool("finish-app", false, "Marca la mejora como frente de finalización continua de la app hasta cierre salvo bloqueo real")
	repoMejorarCmd.Flags().Bool("autonomia-persistente", false, "Activa supervisor/reviewer Codex persistentes para cerrar la app sin supervisión humana salvo bloqueo real")
	repoMejorarCmd.Flags().String("supervisor", "", "Agente supervisor preferido para autonomía persistente")
	repoMejorarCmd.Flags().String("reviewer", "", "Agente reviewer preferido para autonomía persistente")
	repoMejorarCmd.Flags().Int("max-workers", 3, "Máximo de workers paralelos para autonomía persistente")
	repoMejorarCmd.Flags().Bool("despachar", true, "Tras crear la tarea, despacha el siguiente paso del pipeline")

	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoRevisarCmd)
	repoCmd.AddCommand(repoMejorarCmd)
	rootCmd.AddCommand(repoCmd)
}
