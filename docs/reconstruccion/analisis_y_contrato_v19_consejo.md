# V19 — Consejo: contrato rojo

Fecha: 2026-07-22. Base: `e311a97e4f`.

Estado: **awaiting_dependency** de V18 (`independent_reviews`) y producto V19
ausente. Este contrato y su aceptación son deliberadamente rojos; no acreditan
ninguna capability ni anticipan APIs de V18.

## Alcance y autoridad

V19 es dueño de `GOV-11`, `GOV-13`, `GOV-14`, `STG-06`, `STG-08` y `EVD-07`.
El Consejo reúne propuestas, crítica, ballots, disenso, veto de seguridad y una
decisión durable para un Goal, generación, árbol, diff y tests exactos. La
aplicación sigue siendo el único escritor de lifecycle; un Director con lease
propone, pero no obtiene una ruta privada a la decisión.

El Consejo no es Director, scheduler, store ni sustituto de V18: autor,
reviewer primario y reviewer adversarial siguen siendo tres launches
independientes. La entrada V19 solo puede enlazar evidencia acreditada de esos
launches cuando V18 esté sellada.

## Política y hechos causales

Las políticas son exactas y excluyentes:

- `auto`: se solicita Consejo y su decisión se persiste antes de la promoción.
- `required`: la promoción queda bloqueada hasta decisión del Consejo.
- `skip_by_operator`: requiere principal autorizado, motivo no vacío, instante
  UTC y hash de spec; deja un hecho durable de skip, no un falso ballot.

Cada propuesta/crítica/ballot lleva actor o launch acreditado, proyecto, Goal,
generación, sujeto (tree/diff/tests), spec hash e idempotency key. Un ballot no
puede suplantar otro launch ni cruzar proyecto/generación/sujeto. Disenso y veto
de seguridad son hechos inmutables y observables; un veto bloquea la promoción.
La decisión no reescribe propuestas, ballots ni reviews.

## Recovery, replay y seguridad

La persistencia futura debe usar la misma transacción de snapshot, evento y
outbox; no se admite `CouncilStore`, cola, daemon o lifecycle paralelo. Replay
con la misma identidad devuelve el mismo hecho; cambio de payload o sujeto
causal falla. Crash/restart conserva propuesta, crítica, ballots, skip, disenso,
veto y decisión sin duplicar un launch ni convertir un ACK/texto en evidencia.

Toda consulta y escritura queda filtrada por principal/proyecto. Datos de
provider, prompt, secreto, PID, argv y workspace privado no forman parte del
ballot ni del receipt.

## Gates rojos y E2E futuro

El fixture `v19_council.json` declara tres E2E aislados: `auto`, `required` y
`skip_by_operator`. Cada uno usa runtime/estado temporal propio, la misma
identidad causal y su propia comprobación de restart/replay:

1. `auto` conserva propuesta, crítica, ballot, disenso y decisión ligada a los
   tres launches de V18.
2. `required` no promociona sin decisión; un veto de seguridad bloquea y queda
   durable.
3. `skip_by_operator` solo avanza con principal, motivo, hora UTC y spec hash;
   replay es idempotente y un skip cruzado falla.

Hasta que V18 esté acreditada y exista producto V19, `TestAcceptanceV19Council`
falla exclusivamente como `V19_PRODUCT_PENDING`. No se inventa un fake de V18
ni se vincula a tipos, métodos o paquetes aún inexistentes.

## P/S/E y write-set

`P` contendrá contrato productivo V19 y sus tests sin autoacreditarse. `S`
sellará árbol, binario, configuración efectiva y sujetos de review. `E`
ejecutará los tres E2E desde `detached_clean` de `S` y emitirá un receipt V3
externo al candidato.

Este corte solo puede tocar:

- `docs/reconstruccion/analisis_y_contrato_v19_*.md`;
- `docs/reconstruccion/worksets/v19_*.json`;
- `acceptance/v19_*.go` y `acceptance/fixtures/v19_*.json`.

Producto, roadmap, evidence, trace, configuración y cualquier superficie V18
quedan fuera. El siguiente write-set causal pertenece a V18 sellada y después
a la implementación V19, con contrato y APIs definidos por esa evidencia.
