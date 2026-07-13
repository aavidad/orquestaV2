package main

import (
	"context"
	"encoding/json"
	"fmt"

	autonomy "orquesta/modulos/orquesta-autonomy-program"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// autonomyProgramExecutorV0 cablea el programa de autonomia sobre su dominio. El
// adaptador no decide nada: traduce. La topologia, la frontera y la recuperacion
// viven en el nucleo.
type autonomyProgramExecutorV0 struct {
	store autonomyProgramStoreV0
}

var _ orquestamcp.MCPAutonomyProgramPortV0 = autonomyProgramExecutorV0{}

func newAutonomyProgramExecutorV0(store autonomyProgramStoreV0) autonomyProgramExecutorV0 {
	return autonomyProgramExecutorV0{store: store}
}

func (executor autonomyProgramExecutorV0) RunAutonomyProgramActionV0(
	ctx context.Context,
	input orquestamcp.MCPAutonomyProgramToolInputV0,
) (orquestamcp.MCPAutonomyProgramToolResultV0, error) {
	switch input.Action {
	case orquestamcp.MCPAutonomyProgramActionSaveV0:
		return executor.saveV0(ctx, input)
	case orquestamcp.MCPAutonomyProgramActionObserveV0:
		program, err := executor.loadV0(ctx, input)
		if err != nil {
			return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
		}
		return resultadoDePrograma(program, nil, nil)
	case orquestamcp.MCPAutonomyProgramActionFrontierV0:
		return executor.frontierV0(ctx, input)
	case orquestamcp.MCPAutonomyProgramActionRecoverV0:
		return executor.recoverV0(ctx, input)
	default:
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, fmt.Errorf("autonomy_program_action_desconocida: %q", input.Action)
	}
}

func (executor autonomyProgramExecutorV0) saveV0(
	ctx context.Context,
	input orquestamcp.MCPAutonomyProgramToolInputV0,
) (orquestamcp.MCPAutonomyProgramToolResultV0, error) {
	var program autonomy.AutonomyProgramV0
	if err := json.Unmarshal(input.Program, &program); err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, fmt.Errorf("autonomy_program_invalido: %w", err)
	}
	normalizado, err := autonomy.NewAutonomyProgramV0(program)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	if err := executor.store.SaveAutonomyProgramV0(ctx, normalizado); err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	return resultadoDePrograma(normalizado, nil, nil)
}

func (executor autonomyProgramExecutorV0) loadV0(
	ctx context.Context,
	input orquestamcp.MCPAutonomyProgramToolInputV0,
) (autonomy.AutonomyProgramV0, error) {
	return executor.store.LoadAutonomyProgramV0(ctx, input.ProjectRef, input.RootRef, input.ProgramRef)
}

// frontierV0 calcula que nodos estan listos y PERSISTE el avance con CAS. Si otro
// planificador se adelanto, no se pisa: se devuelve conflicto y el llamante
// recarga. Sin CAS, dos planificadores relanzarian los mismos nodos.
func (executor autonomyProgramExecutorV0) frontierV0(
	ctx context.Context,
	input orquestamcp.MCPAutonomyProgramToolInputV0,
) (orquestamcp.MCPAutonomyProgramToolResultV0, error) {
	actual, err := executor.loadV0(ctx, input)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	frontier, err := autonomy.PrepareAutonomyProgramFrontierV0(actual)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	intercambiado, err := executor.store.CompareAndSwapAutonomyProgramV0(ctx, actual, frontier.Program)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	if !intercambiado {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, fmt.Errorf(
			"%w: otro planificador avanzo el programa", ErrAutonomyProgramConflictV0,
		)
	}
	lanzamientos := make([]string, 0, len(frontier.Launches))
	for _, launch := range frontier.Launches {
		lanzamientos = append(lanzamientos, launch.LaunchRef)
	}
	return resultadoDePrograma(frontier.Program, lanzamientos, nil)
}

func (executor autonomyProgramExecutorV0) recoverV0(
	ctx context.Context,
	input orquestamcp.MCPAutonomyProgramToolInputV0,
) (orquestamcp.MCPAutonomyProgramToolResultV0, error) {
	program, err := executor.loadV0(ctx, input)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	recovery, err := autonomy.RecoverAutonomyProgramActionsV0(program)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	refs := make([]string, 0, len(recovery.Launches))
	for _, launch := range recovery.Launches {
		refs = append(refs, launch.LaunchRef)
	}
	return resultadoDePrograma(program, nil, refs)
}

func resultadoDePrograma(
	program autonomy.AutonomyProgramV0,
	lanzamientos []string,
	recuperaciones []string,
) (orquestamcp.MCPAutonomyProgramToolResultV0, error) {
	bytes, err := json.Marshal(program)
	if err != nil {
		return orquestamcp.MCPAutonomyProgramToolResultV0{}, err
	}
	return orquestamcp.MCPAutonomyProgramToolResultV0{
		ProgramRef:   program.ProgramRef,
		Status:       string(program.Status),
		NodeCount:    len(program.Nodes),
		LaunchRefs:   lanzamientos,
		RecoveryRefs: recuperaciones,
		Program:      bytes,
	}, nil
}
