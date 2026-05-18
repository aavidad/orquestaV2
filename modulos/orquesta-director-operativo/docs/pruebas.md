# Pruebas: orquesta-director-operativo

Comando:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
```

Cobertura actual:

- plan de programacion listo incluye lanzar subagentes, esperar, revisar,
  ejecutar tests y replanificar/cerrar;
- programacion sin worktree aislada, rama, write-set o tests se rechaza;
- dominio con contexto insuficiente queda bloqueado y pide contexto sin lanzar
  subagentes;
- dominio con contexto suficiente usa el mismo bucle operativo de subagentes;
- delegacion recursiva queda opt-in, acotada y revisada por el director;
- presupuestos se acotan y write-set se deduplica.
- proyeccion a olas conserva contrato operativo y no expone launch cuando falta
  contexto.

Pendiente de integracion:

- esperar por cohorte/ola concreta de subagentes usando `WaitAgentRefs` o refs
  durables derivadas de la ola;
- completar el materializador en
  `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`
  mas alla de `launch_subagents`: espera, review, tests, rework/replan y cierre;
- bloquear contexto OPES/domain-work insuficiente antes de lanzar agentes;
- no consumir decisiones de director hijo antes de ACK registrado.
- smoke real de recursion Codex gobernada con parent/child refs,
  profundidad/fanout, presupuesto y review causal.
