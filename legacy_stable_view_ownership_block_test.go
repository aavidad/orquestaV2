// Este contrato acredita el bloqueo de autoridad de la infraestructura del
// censo mediante datos estructurados y sin consultar el equipo ni el legado.
package orquesta_test

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

const stableViewOwnershipBlockPath = "docs/reconstruccion/bloqueo_propiedad_vista_estable_censo_2026-07-30.md"

type blockedOwnershipRoadmap struct {
	AcceptanceContracts []struct {
		ID         string   `json:"id"`
		Vertical   string   `json:"vertical"`
		Status     string   `json:"status"`
		Assertions []string `json:"assertions"`
	} `json:"acceptance_contracts"`
	CapabilityEntries []struct {
		ID                  string   `json:"id"`
		Kind                string   `json:"kind"`
		OwnerContext        string   `json:"owner_context"`
		Status              string   `json:"status"`
		AcceptanceContracts []string `json:"acceptance_contracts"`
	} `json:"capability_entries"`
}

func TestStableViewOwnershipRemainsBlockedByRoadmapGap(t *testing.T) {
	document := readOwnershipBlockFile(t, stableViewOwnershipBlockPath)
	if lines := strings.Count(document, "\n"); lines > 220 {
		t.Fatalf("bloqueo demasiado grande: %d líneas; máximo 220", lines)
	}
	normalized := strings.Join(strings.Fields(document), " ")
	for _, required := range []string{
		"Estado: contradicción de autoridad abierta; ejecución real en **NO-GO**.",
		"`AC-V34-CUTOVER-ACCREDITATION` es un **contrato de aceptación planificado**, no una capacidad.",
		"El ratchet vigente es `ninguna capacidad decidida`.",
		"Es una recomendación, no una decisión.",
		"resolución debe hacerse allí en un único cambio revisable antes de programar",
		"dispositivo distinto O vista externa cercada",
		"cuota o preasignación real mínima de 44 GiB",
		"Ninguno acredita, infiere ni sustituye los seis recibos de infraestructura",
		"cinco recibos infraestructurales previos -> mapeo -> sujeto -> bruto -> recibo de intento -> confirmación -> recibo de lote -> revisión independiente del mismo candidato -> liberación y restitución -> sexto recibo infraestructural final",
		"Está permitido ahora bajo `GOV-16` un validador puro de los bytes V3, universo y candidato",
		"Solo puede certificar forma, ligadura por bytes y coherencia interna.",
		"no consume descriptores de fichero persistidos",
		"Permanece bloqueado cualquier validador o adquiridor que afirme propiedad o autenticidad física",
		"menos de 44 GiB reales",
		"No se autoriza implementar la composición física, adquirir, montar, reservar, afirmar autenticidad física ni abrir raíces reales.",
	} {
		if !strings.Contains(normalized, required) {
			t.Errorf("el bloqueo perdió el contrato %q", required)
		}
	}
	assertOwnershipRows(t, document, []string{
		"recibo_autorizacion_operativa",
		"recibo_quiescencia_cercado",
		"recibo_adquisicion_vista",
		"recibo_aptitud_salida",
		"recibo_reserva_fisica",
		"recibo_liberacion_restitucion",
	})
	assertOwnershipRows(t, document, []string{
		"orquesta.physical-census-subject.v1",
		"orquesta.physical-census-attempt-receipt.v1",
		"orquesta.physical-census-batch-receipt.v1",
		"orquesta.physical-census-confirmation.v1",
	})
	for _, forbidden := range []string{
		"propietario futuro único", "Su único propietario admisible",
		"Estado: decisión de alcance",
		"se introduce `StableViewProvider`",
	} {
		if strings.Contains(document, forbidden) {
			t.Fatalf("el bloqueo fingió una decisión o autorización: %q", forbidden)
		}
	}
	assertCurrentOwnershipGap(t)
}

func assertOwnershipRows(t *testing.T, document string, identifiers []string) {
	t.Helper()
	for _, identifier := range identifiers {
		if strings.Count(document, "| `"+identifier+"` |") != 1 {
			t.Errorf("%q no tiene una única fila estructurada", identifier)
		}
	}
}

func assertCurrentOwnershipGap(t *testing.T) {
	t.Helper()
	var roadmap blockedOwnershipRoadmap
	if err := json.Unmarshal(
		[]byte(readOwnershipBlockFile(t, "product/roadmap.json")),
		&roadmap,
	); err != nil {
		t.Fatalf("decodificar roadmap: %v", err)
	}
	contractCount := 0
	wantAssertions := []string{
		"all accepted capabilities are accredited against immutable candidate digests",
		"P0 and P1 are zero",
		"legacy writers imports bridges leases and unreconciled receipts are zero",
		"release restore rollback and shutdown are green",
	}
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID != "AC-V34-CUTOVER-ACCREDITATION" {
			continue
		}
		contractCount++
		if contract.Vertical != "cutover_accreditation" ||
			contract.Status != "planned" ||
			!slices.Equal(contract.Assertions, wantAssertions) {
			t.Fatalf("el contrato V34 cambió su estructura: %#v", contract)
		}
	}
	if contractCount != 1 {
		t.Fatal("el contrato V34 vigente no conserva su estructura planificada")
	}
	want := map[string]string{
		"STG-17": "stage_template",
		"EVD-08": "evidence",
		"EVD-10": "evidence",
	}
	for _, capability := range roadmap.CapabilityEntries {
		if capability.ID == "AC-V34-CUTOVER-ACCREDITATION" {
			t.Fatal("el contrato V34 apareció indebidamente como capacidad")
		}
		kind, selected := want[capability.ID]
		if !selected {
			if capability.ID == "STG-17" ||
				capability.ID == "EVD-08" ||
				capability.ID == "EVD-10" {
				t.Fatalf("capacidad candidata duplicada: %s", capability.ID)
			}
			continue
		}
		if capability.Kind != kind ||
			capability.OwnerContext != "cutover_accreditation" ||
			capability.Status != "declared" ||
			!slices.Equal(
				capability.AcceptanceContracts,
				[]string{"AC-V34-CUTOVER-ACCREDITATION"},
			) {
			t.Fatalf("candidata %s cambió su estructura: %#v", capability.ID, capability)
		}
		delete(want, capability.ID)
	}
	if len(want) != 0 {
		t.Fatalf("faltan capacidades candidatas: %#v", want)
	}
}

func readOwnershipBlockFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	return string(content)
}
