# Voto tardío de agentes en propuestas cerradas

## Contexto

Cuando una propuesta alcanza consenso automático (todos los agentes habilitados votan acuerdo), se cierra de inmediato. Si un agente se incorporó al sistema después de que se crearon propuestas antiguas, o si el consenso se alcanzó antes de que pudiera emitir su posición, su voto quedaba sin registrar y el sistema bloqueaba cualquier intento posterior.

Esto impedía que los votos y comentarios de agentes tardíos quedaran en el historial, perdiendo información valiosa de gobernanza.

## Solución implementada

Se modificó la lógica de votación para permitir **voto tardío**: un agente puede votar en una propuesta ya cerrada siempre que no haya emitido una posición definitiva anteriormente (ni acuerdo, ni desacuerdo, ni abstención).

### Reglas

| Situación del agente en la propuesta | ¿Puede votar? |
|--------------------------------------|---------------|
| Sin entrada en votos (agente nuevo)  | Sí            |
| Entrada con posición `pendiente`     | Sí            |
| Entrada con posición definitiva      | No            |

El voto tardío **no re-evalúa el consenso**. La propuesta permanece en su estado cerrado; el voto queda registrado únicamente como opinión y trazabilidad histórica.

### Ficheros modificados

- [`propuestasapp/service.go`](../propuestasapp/service.go) — `VoteDetail`: lógica de voto tardío en la ruta API/servidor.
- [`cmd/votar.go`](../cmd/votar.go) — fallback CLI local: misma lógica para cuando no hay servidor disponible.

## Uso

No requiere ningún flag especial. El comando `orquesta votar` detecta automáticamente si se trata de un voto tardío:

```bash
orquesta votar OP-087 acuerdo --agente claude --comentario "Autogestion supervisada correcta."
```

Si el agente ya había votado con posición definitiva, el comando devuelve el error habitual:

```
Error: la propuesta OP-087 ya está cerrada (consenso)
```

## Motivación

Las opiniones de todos los agentes tienen valor histórico y de auditoría, aunque lleguen después del consenso. Permiten entender divergencias, detectar patrones de desacuerdo tardío y mantener el historial completo de gobernanza del proyecto.
