# Decisiones: orquesta-domain-work-memory

```text
Fecha: 2026-05-17
Decision: Mantener `orquesta-domain-work-memory` como referencia volatil tras
crear `orquesta-domain-work-file`.
Motivo: el adaptador in-memory sigue siendo util para tests rapidos,
concurrencia y contratos offline, pero la referencia durable de reinicio pasa a
ser el adaptador file-based.
Alternativas: borrar el modulo in-memory; mover filesystem al in-memory;
fusionar ambos adaptadores.
Impacto: los casos de uso pueden seguir inyectando cualquiera de los dos por
`DomainWorkJobCreatorPortV0`. La persistencia durable se prueba en
`orquesta-domain-work-file`.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Crear `orquesta-domain-work-memory` como conector in-memory generico
para `DomainWorkJobCreatorPortV0`.
Motivo: el nucleo ya tenia DTOs y puerto de jobs de dominio, pero no habia una
implementacion neutral que permitiera probar flujos completos sin OPES, Codex,
runtime, DB, filesystem ni red. La pieza deja un comportamiento ejecutable para
futuros conectores durables.
Alternativas: meter ledger en `orquesta-domain-work`; reutilizar
`orquesta-run-queue`; acoplar la prueba a OPES; crear directamente un conector
DB.
Impacto: el modulo crea jobs aceptados en memoria, aplica idempotencia por
`domain_ref + idempotency_key`, rechaza conflictos sin sobrescribir e implementa
lectura filtrada de records para probar el contrato neutral sin IO.
Estado: aceptada_local
```

```text
Fecha: 2026-05-17
Decision: Devolver `DomainWorkJobV0{status=invalid}` para requests invalidos o
conflictos de idempotencia en lugar de usar error Go.
Motivo: `DomainWorkJobCreatorPortV0` representa rechazo de dominio como estado
de job, mientras que los errores Go quedan reservados para fallos operativos
como contexto cancelado.
Alternativas: propagar error tipado en conflicto; panic en request invalido;
persistir todos los rechazos.
Impacto: los casos de uso superiores pueden tratar rechazo como resultado de
contrato y detenerse sin perder progreso previo. El conector no guarda requests
invalidos ni conflictos.
Estado: aceptada_local
```
