# Decisiones

## Memoria local solamente

El store usa mapas protegidos por `sync.RWMutex`. No intenta conservar estado entre procesos ni exponer transporte.

## Copias defensivas

Las entradas y salidas copian slices mutables para que callers no puedan modificar estado interno.

## Sin reloj global

El store no asigna `updated_at` automaticamente. Los consumidores pueden sembrar candidatos con tiempo explicito y delegar ranking a `RankRunCandidatesV0`.

## Prioridad v0

Como `RunQueuePriorityWriterPortV0` ya existe, este modulo lo implementa. Si un comando de prioridad llega para una run no sembrada, crea un candidato minimo con estado `ready`.
