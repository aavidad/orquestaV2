# orquesta-app-change-director-source

Fuente hexagonal que lee `AppChangeRecordV0` aceptados y, cuando la solicitud
ya tiene write-set y criterios verificables, emite decisiones del director para
replanificar trabajo sin conocer la implementacion interna del stack.
El origen del record puede ser una solicitud directa o un evento de cambio ya
normalizado por `orquesta-app-change`.

No ejecuta agentes ni toca ficheros de app. Solo genera decisiones compactas.
