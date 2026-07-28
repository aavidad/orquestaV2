# Corte P1: pool multi-HOME Codex en una sola Orquesta

Estado: implementación en curso. Este documento no acredita código ni smoke
real.

## Regresión

`BUG-ORQ-20260728-583` corrige una conclusión incompleta de BUG-455. El lock
exclusivo y `MaxConcurrentExecutions=1` son correctos para una cuenta Codex:
evitan que dos procesos renueven o escriban el mismo perfil simultáneamente.
No justifican arrancar una Orquesta por cuenta.

El invariante productivo es:

> una sola instancia residente de Orquesta puede ejecutar varios agentes Codex
> simultáneos; cada ejecución queda ligada a un perfil persistente, exclusivo
> y opaco, con `HOME=CODEX_HOME`, autenticación, journal y work root propios.

Capabilities afectadas: `AGT-01`, `AGT-03`, `GOV-21` y `ORC-10`. El corte no
acredita `AC-V25-PROVIDER-ADAPTERS`, Claude, Gemini ni selección heterogénea de
modelos.

## Autoridad y frontera

- `Goal` y `WorkItem` no conocen cuentas, HOME, credenciales ni paths.
- `application.Orchestrator` conserva la única autoridad de lifecycle.
- SQLite, outbox, leases y fences existentes no se duplican.
- El adapter Codex posee la selección del perfil y su binding durable antes de
  arrancar el proceso.
- Cada perfil conserva el adapter endurecido actual: lock vitalicio, binding
  opaco, journal causal, validación de propietario, permisos, enlaces y
  `auth.json`.
- El pool no crea otra cola, scheduler, base de datos ni estado de lifecycle.
- Nunca se copia `auth.json`; el perfil persistente recibe sus propias
  renovaciones.
- La composición mantiene un scheduler lógico y una sola superficie MCP/HTTP.

## Evidencia legacy que se recupera

El commit `1ca95b2` de
`/home/alberto/Trabajo/orquesta-autonomia-clean` ya implementaba el flujo:

1. `agentesapp/runtime_service.go` preparaba el agente solicitado;
2. `internal/lanzamientoruntime/preparacion.go` delegaba en el registro de
   runtimes;
3. `runtimeagente/driver.go` resolvía `ProvisionedProfile` y proyectaba su
   entorno;
4. `internal/controlruntime/arranque.go` lanzaba el proceso;
5. `/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil` fijaba
   `HOME=CODEX_HOME` para `Codex1`...`Codex13`.

También existían contratos `AgentHomeV0` para cuenta, cuota, cooldown y
sesiones. Se reutiliza la idea y la evidencia adversarial, no sus autoridades
paralelas ni sus paths dentro del dominio.

## Implementación acotada

El primer cierre es Codex homogéneo:

- configuración canónica de varios `account_profiles`;
- un adapter Codex seguro por perfil, con capacidad interna uno;
- work roots privados y distintos derivados de la referencia opaca;
- selección de otro perfil cuando uno no tiene slot;
- replay, observación y stop encaminados por el journal durable exacto;
- cierre del pool que espera y apaga todos sus adapters;
- compatibilidad temporal del perfil escalar, sin presentarlo como arquitectura
  productiva.

La selección de proveedores/modelos heterogéneos pertenece a V25. La futura
factoría neutral podrá reutilizar esta frontera, pero no bloquea el cierre
Codex.

## Aceptación obligatoria

No se marcará corregido hasta conservar evidencia de:

1. una sola escucha/instancia Orquesta;
2. dos Goals Codex ejecutándose a la vez con perfiles autenticados distintos;
3. `HOME` y `CODEX_HOME` exactos y diferentes, sin secreto en recibos o logs;
4. journals, work roots y locks disjuntos;
5. replay y restart que vuelven al perfil original;
6. stop de una ejecución sin afectar a la otra;
7. perfil ocupado que no impide usar otro libre;
8. retiro/corrupción de un perfil con trabajo durable rechazado en cerrado;
9. shutdown con cero procesos, cgroups o locks residuales.

El smoke con dos cuentas reales es opt-in y debe informar bloqueo de
autenticación o cuota sin fabricar un verde.
