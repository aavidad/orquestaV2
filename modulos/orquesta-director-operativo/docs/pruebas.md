# Pruebas: orquesta-director-operativo

Comando:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
```

Cobertura actual:

- plan de programacion listo incluye lanzar subagentes, esperar, revisar,
  ejecutar tests y replanificar/cerrar;
- programacion sin worktree aislada, rama, write-set o tests se rechaza;
- `domain_work` listo sin `domain_refs` opacas o `write_set` seguro se rechaza;
- alias conservadores de `mode` y `context_status` se normalizan antes de
  validar para no bloquear entradas reparables;
- dominio con contexto insuficiente queda bloqueado y pide contexto sin lanzar
  subagentes;
- dominio con contexto suficiente usa el mismo bucle operativo de subagentes;
- `MaxParallelAgents` produce varios `launch_subagents` en la misma ola y el
  `write_set` se reparte como ownership inicial cuando hay paths suficientes;
- el presupuesto recursivo calcula el caso operativo 10 padres + 6 hijos por
  padre como 70 agentes planificados y lo bloquea solo si el presupuesto
  explicito queda por debajo;
- delegacion recursiva queda opt-in, acotada y revisada por el director;
- presupuestos se acotan y write-set se deduplica.
- proyeccion a olas conserva contrato operativo y no expone launch cuando falta
  contexto.
- politica general de reparacion conserva salidas aprovechables y decide
  normalizar, pedir correccion, delegar revision, secuenciar o posponer antes de
  rechazar.
- pasos e items conservan `work_profile_kind`; el materializador lo proyecta a
  `WorkflowTaskV0`.
- la proyeccion rechaza planes mutados fuera del builder con parent/child refs
  inexistentes, profundidad/fanout por encima del presupuesto, ciclo operativo
  listo incompleto, `domain_work` listo sin refs/write-set o `needs_context` que
  intenta lanzar subagentes.
- el materializador conserva refs opacas compactas de dominio/evidencia en
  `WorkflowTaskV0.context_refs`, con prefijos tipados de Director Operativo.

Pendiente de integracion:

- no ampliar este contrato puro con runtime, proveedor ni reglas de producto;
- conservar el cierre funcional del smoke real OPES temporal de derivados/cierre
  por goal-first; no reabrir `CODEX-WAVE-REAL`, `CODEX-RECURSION-REAL` ni
  `OPES-DER-RESTO` salvo regresion demostrada;

Invariantes de integracion ya cubiertos fuera del modulo puro:

- no consumir decisiones de director hijo antes de ACK registrado;
- smoke real de ola/cohorte Codex amplia: cerrado por `CODEX-WAVE-REAL`;
- smoke real de recursion Codex gobernada con parent/child refs,
  profundidad/fanout, presupuesto y review causal: cerrado por
  `CODEX-RECURSION-REAL`.

## Notas practicas

- Para una ola complementaria de autoprogramacion con 10 padres y hasta 6 hijos
  por padre, usar `MaxParallelAgents=10`, `MaxSubagentsPerAgent=6`,
  `MaxDelegationDepth=1` y `MaxRecursiveAgents>=70`.
- Si `MaxRecursiveAgents` es menor que el total planificado, el bloqueo correcto
  es `recursive_agent_budget_exceeded`; no debe materializar agentes parciales.
- `ACK.tests` o un resumen textual no son evidencia suficiente para cierre:
  las pruebas requeridas deben llegar como evidencia durable y causal en la capa
  de orquestacion.
