# Manual de revisión independiente — Orquesta rebuild V22

Estado del manual: vigente para la rama
`reconstruccion/v22-codex-e2e-integrated`.

## 1. Árbol que se debe revisar

```text
/home/alberto/Trabajo/orquesta-rebuild-worktrees/v22-codex-e2e-integrated
```

No revisar ni modificar como si fuera el candidato nuevo:

```text
/home/alberto/Trabajo/orquesta
```

El segundo árbol es Orquesta anterior y se conserva aislado. No se debe copiar
código legacy al rebuild ni corregirlo durante esta revisión.

Antes de revisar:

```bash
cd /home/alberto/Trabajo/orquesta-rebuild-worktrees/v22-codex-e2e-integrated
git status --short --branch
git rev-parse HEAD
scripts/bootstrap_agent_tooling.sh --status
```

No usar `codebase-memory-mcp`, no indexar el repositorio y no arrancar servidores
residentes para una revisión estática. Usar `rg`, lecturas acotadas y pruebas
focales. No instalar herramientas ni editar ficheros salvo orden expresa.

## 2. Autoridad documental

Leer, en este orden:

1. `AGENTS.md`.
2. `internal/AGENTS.md` si se revisa `internal/**`.
3. `docs/reconstruccion/ruta_total_100.md`, especialmente V22 y el gate de
   autoservicio.
4. `docs/reconstruccion/estado_y_handoff_rebuild.md`.
5. `docs/reconstruccion/analisis_y_contrato_v20_registro_comandos.md`, sección
   10.2 sobre autoridad execution-bound diferida a V22.
6. `docs/reconstruccion/analisis_y_contrato_v22_codex_e2e.md` cuando exista.
7. `acceptance/fixtures/v22_codex_e2e.json` y sus tests cuando existan.
8. `docs/inventario_bugs_orquesta_2026-06-30.md` para comprobar que todo bug
   observado queda registrado y no desaparece al corregirse.

Los documentos históricos no prevalecen sobre esas fuentes.

El informe forense Claude de 2026-07-23 es una auditoría estática de trabajo en
vuelo, no un receipt ni un sello P/S/E. Usarlo para comprobar 366/390/391,
prohibición de `Skip` y necesidad de revisión externa; no extrapolar deuda de
V25 o V34 a la aprobación V22.

## 3. Qué debe cerrar V22

V22 es el primer vertical Codex completo para trabajo ya planificado:

```text
plan explícito y DAG/write-sets
→ hijos Codex reales
→ mailbox causal y ACK
→ workspace Git aislado
→ tests independientes
→ autor + review primaria + review adversarial
→ integración o rework causal
→ cierre
→ stop selectivo, backup, crash/restart y shutdown limpio
```

El E2E debe usar cuatro Goals concurrentes A/B/C/D. Detener B no puede alterar
A, C ni D. Tras caída y reinicio deben preservarse estado, causalidad y
terminalidad. Al final no puede quedar ningún proceso propio vivo ni estados
terminales contradictorios.

La actividad de negocio se ejerce por el MCP público generado desde el registro
de comandos. El harness solo puede encargarse de acciones externas de prueba
como SIGKILL, restart, verificación de backup y censo de procesos.

## 4. Qué no pertenece a V22

No pedir ni introducir en esta versión:

- Wizard, dossier o conversión de petición abierta en plan: V23.
- Web/PWA: V24.
- Hermes, Claude, Gemini, Ollama o local: V25.
- registro general de tools/skills/resources: V26.
- RAG y routing: V27.
- plugins/Forge: V28.
- deploy/notificaciones: V29.
- PostgreSQL/S3/multihost: V31.
- operación completa/update/daemon: V32.
- autotransformación privilegiada de Orquesta: V33.

V22 tampoco debe crear otro Director, scheduler, cola, lifecycle, store de
estado global ni DTO autoritativo paralelo.

## 5. Invariantes críticas que revisar

### Arquitectura

- Dominio y aplicación no importan Codex, SQLite, MCP, HTTP, filesystem, env ni
  rutas físicas.
- `application` sigue siendo el único escritor del lifecycle.
- Codex entra solo por la familia neutral `AgentLauncher`/`AgentObserver` y
  capacidades opcionales.
- SQLite y el CredentialStore son adaptadores intercambiables; no se filtran al
  núcleo.
- No se reintroduce código legacy ni compatibilidad por adaptador.

### Identidad de ejecución

- Cada Codex hijo usa un service principal distinto de cualquier humano.
- La autoridad *material-free* se deriva del tuple durable Goal/WorkItem/
  Execution, intento, reemplazo, generaciones y spec; no se persiste una fila
  de sesión ni alias de autoridad nuevo.
- Un claim transportado por MCP nunca constituye autoridad por sí mismo.
- Proyecto ajeno, token ajeno, ejecución sucesora, intento/generación stale o
  ejecución revocada producen `forbidden` sin ejecutar el handler.
- Restart conserva el binding válido; stop/reemplazo invalida el anterior.
- El service principal no obtiene permisos principal-wide ni comandos humanos:
  fuera de los comandos execution-bound autenticados el dispatcher debe negar
  antes del handler; tampoco recibe integración, efectos o administración.

### Secretos y MCP del hijo

- El bearer de ejecución no aparece en DB, prompt, logs, diagnósticos,
  artefactos, receipts, argv, effective config ni Git.
- La autoridad durable no contiene material; el `CredentialStore` existente es
  el único almacén de secretos. La credencial solo se materializa durante el
  launch o la continuación post-artefacto y se limpia después.
- Codex recibe configuración MCP por su adaptador, nunca desde el core.
- Endpoint y nombre de variable se definen en una superficie canónica; no hay
  `os.Getenv`, strings de env ad hoc ni aliases dispersos.
- El prompt público usa el catálogo i18n V21; no reaparece una segunda fuente
  inglesa hard-coded.
- `artifacts.read` de un principal de ejecución solo puede resolver un
  artefacto incluido en un mailbox ya admitido cuyo destinatario coincide
  exactamente con su principal y `ExecutionRef`; no vale por proyecto, Goal ni
  por ser emisor.

### Handoff post-artefacto

- Tras persistir artefacto y éxito del hijo, el único worker reclama
  `admit_mailbox` en la outbox existente y llama `orquesta.mailbox.admit` por
  MCP con su identidad de ejecución exacta.
- No puede sustituirlo un principal humano ni puede aparecer una cola, scheduler
  o writer de lifecycle paralelo. Verificar replay y restart idempotentes.

### Registro y compatibilidad

- No se rompe el registro único ni se añaden endpoints manuales.
- V20 conserva su contrato histórico; cualquier extensión debe tener versión,
  generación y ratchet explícitos.
- HTTP, MCP, CLI y SDK comparten dispatcher, autorización, schemas, códigos e
  i18n.
- Ninguna interfaz escribe lifecycle.

### E2E y evidencia

- No aceptar fake como sustituto del Codex real del gate.
- No aceptar `Skip`, package `PASS` sin test ejecutado ni un E2E de un solo Goal.
- La salida JSON de `go test` debe contener eventos `run` y `pass` de cada test
  real exigido.
- Crash no puede simularse con `Shutdown` cooperativo.
- El E2E no accede al `Orchestrator` interno para completar manualmente pasos de
  negocio.
- PID/PGID se valida con identidad de proceso y marcador de nacimiento; no solo
  con `pgrep`.
- El sellado P/S/E debe ligar commits, tree, fixture, argv, salida y digests sin
  autorreferencias imposibles.
- La presencia del WIP anterior no prueba 366/390/391: exigir negativos de
  autoridad, lectura de artefacto sin mailbox exacto, handoff post-artefacto y
  restart antes de cualquier conclusión.

## 6. Pruebas mínimas de revisión

Mientras V22 esté en desarrollo, ejecutar primero las focales declaradas por su
contrato. Antes de aprobar el candidato final:

```bash
git diff --check
go test -mod=vendor -count=1 ./...
```

Además deben pasar, con nombres finales equivalentes a los congelados en el
contrato V22:

```text
TestAcceptanceV22CodexE2E
TestV22ExecutionServicePrincipalBindingRevocationSuccessorAndRestart
TestV22RealCodexFourGoalsSelectiveStopCrashRestartAndCloseThroughMCP
TestV22NoTerminalContradictionAndNoOwnedProcess
TestProductRoadmapV22ScopeAndLifecycleContract
```

El E2E Codex real puede ser opt-in y costoso, pero el receipt final no puede
existir sin haberlo ejecutado desde un candidato detached y limpio.

## 7. Cómo informar

Entregar hallazgos antes que resumen. Por cada hallazgo:

```text
Severidad: P0 | P1 | P2 | P3
Fichero:línea
Invariante o contrato roto
Escenario reproducible
Impacto real
Corrección mínima recomendada
Prueba que debe fallar antes y pasar después
```

Separar:

- bugs demostrados;
- riesgos no demostrados;
- deuda asignada a una V posterior;
- mejoras opcionales.

No abrir una “nueva generación” arquitectónica para resolver un hallazgo. Si el
mismo fallo aparece en varias capas, buscar primero una única causa en frontera,
autoridad, estado o composición. Todo bug demostrado debe añadirse al inventario
y conservar su fila tras el arreglo con prueba/commit de cierre.

## 8. Criterio de aprobación

Claude debe concluir una de estas tres salidas:

- `APROBADO`: contrato V22 completo, pruebas y evidencia válidas, sin P0/P1.
- `APROBADO CON DEUDA`: sin P0/P1; deuda concreta ya asignada fuera de V22 y sin
  falsear ninguna capability.
- `NO APROBADO`: cualquier falso verde, autoridad falsificable, fuga de secreto,
  doble lifecycle, E2E incompleto, proceso residual o contradicción terminal.

Una revisión no puede declarar V22 cerrada basándose solo en compilación, tests
unitarios, documentación o piezas ya acreditadas por V1–V21.
