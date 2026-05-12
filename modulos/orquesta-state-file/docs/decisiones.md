# Decisiones

## STF-001: conector file-based

El estado operativo durable entra como conector file-based para evitar acoplar el
servidor o el nucleo a una base de datos concreta. Backends futuros deben
implementar los mismos puertos.

## STF-002: JSON por agregado

Cada agregado se guarda en un documento JSON separado. Las escrituras usan
temporal, `sync`, `rename` y sincronizacion del directorio.
