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
- dominio con contexto insuficiente queda bloqueado y pide contexto sin lanzar
  subagentes;
- dominio con contexto suficiente usa el mismo bucle operativo de subagentes;
- `MaxParallelAgents` produce varios `launch_subagents` en la misma ola y el
  `write_set` se reparte como ownership inicial cuando hay paths suficientes;
- delegacion recursiva queda opt-in, acotada y revisada por el director;
- presupuestos se acotan y write-set se deduplica.
- proyeccion a olas conserva contrato operativo y no expone launch cuando falta
  contexto.
- pasos e items conservan `work_profile_kind`; el materializador lo proyecta a
  `WorkflowTaskV0`.
- la proyeccion rechaza planes mutados fuera del builder con parent/child refs
  inexistentes, profundidad/fanout por encima del presupuesto, ciclo operativo
  listo incompleto, `domain_work` listo sin refs/write-set o `needs_context` que
  intenta lanzar subagentes.

Pendiente de integracion:

- no ampliar este contrato puro con runtime, proveedor ni reglas de producto;
- completar los smokes reales de las composiciones que consumen este contrato;
- no consumir decisiones de director hijo antes de ACK registrado.
- smoke real de recursion Codex gobernada con parent/child refs,
  profundidad/fanout, presupuesto y review causal.
