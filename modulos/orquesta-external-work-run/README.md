# orquesta-external-work-run

Caso de uso para aceptar trabajo externo de dominio ya especificado, crear un
run operativo, abrir `programacion`, registrar el cambio y encolar la run.

No decide contenido, no conoce OPES y no crea directores. Las apps externas
entregan un `AppChangeRequestV0` con `external_work`; Orquesta lo convierte en
microtarea mediante sus fuentes de decision y agentes configurados.
