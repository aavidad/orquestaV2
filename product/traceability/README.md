# Trazabilidad V03

Este directorio no importa ni ejecuta el legado. Congela un censo mecánico y
reglas explícitas de disposición para poder reimplementar conducta útil sin
copiar el núcleo anterior.

`legacy_go.json` clasifica los 106 directorios de `modulos/` por rutas exactas.
Cada paquete, fichero y símbolo productivo hereda únicamente la regla de su
módulo; los nombres y palabras del código no deciden su destino. El test AST
falla si aparece un módulo sin regla o cambia el hash/conteo del censo.

Un `supersede` significa que las capacidades aceptadas expresan la conducta que
debe caracterizarse y rehacerse. No acredita equivalencia, no autoriza copiar
código y todavía no permite retirar el módulo. Un `retire` también exige una
caracterización que demuestre que no se pierde conducta útil.

`pending_sources.json` conserva fuentes exactas de tareas, bugs, skills y
rulepacks mediante paths y rangos estructurales documentados. Su estado es
deliberadamente pendiente: inventariar una fuente no dispone sus entradas ni
cierra `AC-V03-CANONICAL-LEDGERS`.

Gate focal:

```bash
go test -mod=vendor -count=1 . -run '^TestTraceabilityRebuild'
```
