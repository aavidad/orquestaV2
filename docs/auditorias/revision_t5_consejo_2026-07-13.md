# Revisión independiente T5 — Consejo de sabios

Fecha: 2026-07-13  
Revisor: Codex  
Commits revisados: `5b84e55b5f`, `46399b0aee`

## Veredicto

El núcleo y la tool MCP son avances válidos, pero **T5 no está cerrada** respecto
al requisito del operador: el consejo debe intervenir realmente al crear una app
y las entregas materiales deben recibir al menos dos revisiones independientes.

## Acreditado

- `orquesta-council` asigna roles dinámicos, aplica overrides, evita
  autorrevisión y conserva veto de seguridad.
- `orquesta.council.convene.v0` está registrada y cableada en el bootstrap MCP.
- El smoke de transporte y las suites focales pasan.

Evidencia reejecutada:

```text
GOPROXY=off go test -mod=vendor -count=1 \
  ./modulos/orquesta-council ./modulos/orquesta-mcp ./cmd/orquesta-server \
  -run 'Council|Consejo|EnvVarsBudget'
ok
```

## Brechas que impiden el cierre

1. **No existe gate de creación de app.** Las únicas llamadas productivas a
   `AssignRolesV0` y `DecideV0` están en `cmd/orquesta-server/council_executor_v0.go`,
   alcanzable por invocación MCP explícita. El camino que inicia la app no
   convoca ni exige una decisión aceptada antes de programar.
2. **El caller aporta miembros y votos.** La tool no obtiene candidatos de una
   fuente real de capacidad/cuota y por tanto no demuestra roles en caliente
   sobre agentes disponibles.
3. **Asignación, override y decisión no son durables.** No hay store, CAS,
   receipt, replay ni idempotencia del consejo; reiniciar el servidor pierde la
   decisión.
4. **No hay doble revisión de entregas.** El cambio no crea ni exige dos recibos
   de revisores independientes antes de cerrar una entrega material.
5. **El override no tiene modalidad persistente.** El operador puede forzar un
   rol solo en la petición concreta; falta configuración persistente y
   precedencia durable.
6. **Posible fuga de detalle interno.** `MCPCouncilToolExecutorV0.Execute`
   proyecta `err.Error()` en `Rationale`; la superficie pública debe usar códigos
   tipados y no texto interno arbitrario.

## Corte mínimo recomendado

1. Store durable de convocatorias/asignaciones/ballots/decisiones con CAS,
   idempotencia y replay.
2. Fuente real de miembros/capacidad/cuota; el caller solo declara autor,
   criticidad y overrides autorizados.
3. Gate previo al arranque de la creación: estado `council_pending`; solo una
   decisión durable `accepted` lanza una vez el goal de implementación.
4. Dos revisores independientes por entrega material; impedir que autor y
   revisores compartan identidad/familia cuando haya alternativas.
5. Override puntual y persistente por API/MCP/web, con auditoría y veto de
   seguridad no anulable.
6. Smoke E2E: crear app → consejo real → decisión → implementación → dos
   revisiones → cierre; mutación del gate debe ponerlo rojo.

