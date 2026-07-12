package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ingestion "orquesta/modulos/orquesta-data-ingestion"
	ingestionfile "orquesta/modulos/orquesta-data-ingestion-file"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// dataProfileExecutorV0 cablea la ingesta de datos sobre el adaptador de fichero.
// El adaptador exige un CATALOGO de fuentes: no acepta rutas sueltas. El catalogo
// se reconstruye en cada llamada escaneando el inbox, de modo que un fichero
// nuevo esta disponible sin reiniciar el servidor pero nada fuera del inbox
// llega nunca a ser un dataset.
type dataProfileExecutorV0 struct {
	inboxDir string
}

var _ orquestamcp.MCPDataProfileProfilerPortV0 = dataProfileExecutorV0{}

func newDataProfileExecutorV0(inboxDir string) (dataProfileExecutorV0, error) {
	if strings.TrimSpace(inboxDir) == "" {
		return dataProfileExecutorV0{}, fmt.Errorf("data profile: raiz de ingesta requerida")
	}
	return dataProfileExecutorV0{inboxDir: inboxDir}, nil
}

func (executor dataProfileExecutorV0) ProfileDataSourceV0(
	ctx context.Context,
	input orquestamcp.MCPDataProfileToolInputV0,
) (orquestamcp.MCPDataProfileToolResultV0, error) {
	catalogo, err := executor.catalogoV0()
	if err != nil {
		return orquestamcp.MCPDataProfileToolResultV0{}, err
	}

	if input.DatasetRef == "" {
		datasets := make([]orquestamcp.MCPDataProfileDatasetV0, 0, len(catalogo))
		for ref, registro := range catalogo {
			datasets = append(datasets, orquestamcp.MCPDataProfileDatasetV0{
				DatasetRef: ref,
				SourceKind: string(registro.SourceKind),
			})
		}
		sort.Slice(datasets, func(i, j int) bool { return datasets[i].DatasetRef < datasets[j].DatasetRef })
		return orquestamcp.MCPDataProfileToolResultV0{Datasets: datasets}, nil
	}

	if len(catalogo) == 0 {
		return orquestamcp.MCPDataProfileToolResultV0{}, fmt.Errorf("data_profile_catalogo_vacio")
	}
	adapter, err := ingestionfile.NewAdapterV0(ingestionfile.ConfigurationV0{
		AllowedRoot: executor.inboxDir,
		Sources:     catalogo,
	})
	if err != nil {
		return orquestamcp.MCPDataProfileToolResultV0{}, err
	}
	source, err := adapter.ResolveDataSourceV0(ctx, input.DatasetRef)
	if err != nil {
		return orquestamcp.MCPDataProfileToolResultV0{}, err
	}
	profile, err := adapter.ProfileDataSetV0(ctx, source)
	if err != nil {
		return orquestamcp.MCPDataProfileToolResultV0{}, err
	}

	columns := make([]orquestamcp.MCPDataProfileColumnV0, 0, len(profile.Columns))
	for _, column := range profile.Columns {
		columns = append(columns, orquestamcp.MCPDataProfileColumnV0{
			ColumnRef: column.ColumnRef,
			Name:      column.Name,
			DataType:  column.DataType,
			Nullable:  column.Nullable,
		})
	}
	return orquestamcp.MCPDataProfileToolResultV0{
		DatasetRef:  profile.DatasetRef,
		ProfileRef:  profile.ProfileRef,
		SourceKind:  string(source.SourceKind),
		ContentHash: source.ContentHash,
		AdapterRef:  adapter.AdapterIdentityV0().AdapterRef,
		RowCount:    profile.RowCount,
		Columns:     columns,
	}, nil
}

// catalogoV0 registra como dataset cada fichero del inbox cuya extension mapea a
// un DataSourceKind conocido. Lo que no reconoce, no entra: un fichero no
// catalogado no es alcanzable ni nombrandolo.
func (executor dataProfileExecutorV0) catalogoV0() (map[string]ingestionfile.SourceRegistrationV0, error) {
	entries, err := os.ReadDir(executor.inboxDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ingestionfile.SourceRegistrationV0{}, nil
		}
		return nil, err
	}
	catalogo := make(map[string]ingestionfile.SourceRegistrationV0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		kind, ok := dataSourceKindForFileV0(entry.Name())
		if !ok {
			continue
		}
		catalogo[entry.Name()] = ingestionfile.SourceRegistrationV0{
			DatasetRef:   entry.Name(),
			RelativePath: entry.Name(),
			SourceKind:   kind,
		}
	}
	return catalogo, nil
}

// Solo CSV y JSON: son los unicos formatos que el adaptador sabe leer de verdad
// (`adapter_v0.go:241`). Catalogar xlsx u ods seria anunciar una capacidad que no
// existe y fallar en la llamada, que es justo la enfermedad que perseguimos.
func dataSourceKindForFileV0(name string) (ingestion.DataSourceKindV0, bool) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".csv":
		return ingestion.DataSourceKindCSVV0, true
	case ".json":
		return ingestion.DataSourceKindJSONV0, true
	default:
		return "", false
	}
}
