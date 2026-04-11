package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var microprogramacionCmd = &cobra.Command{
	Use:   "microprogramacion",
	Short: "Microprogramacion dirigida para agentes",
}

var microprogramacionEspecificacionCmd = &cobra.Command{
	Use:   "especificacion",
	Short: "Gestiona especificaciones de funcion",
}

var microprogramacionEspecificacionListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista especificaciones de funcion",
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		proyecto, _ := cmd.Flags().GetString("proyecto")
		estado, _ := cmd.Flags().GetString("estado")
		limit, _ := cmd.Flags().GetInt("limit")
		items, ok, err := listarEspecificacionesFuncionDesdeAPI(tareaID, proyecto, estado, limit)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion listar")
		}
		if len(items) == 0 {
			fmt.Println("No hay especificaciones registradas.")
			return nil
		}
		fmt.Printf("%-5s %-10s %-30s %-28s %s\n", "ID", "ESTADO", "ARCHIVO", "SIMBOLO", "TESTS")
		fmt.Printf("%-5s %-10s %-30s %-28s %s\n", "─────", "──────────", "──────────────────────────────", "────────────────────────────", "─────")
		for _, item := range items {
			fmt.Printf("%-5d %-10s %-30s %-28s %d\n",
				item.ID,
				item.Estado,
				truncar(item.ArchivoObjetivo, 30),
				truncar(item.SimboloObjetivo, 28),
				len(item.TestsObligatorios),
			)
		}
		return nil
	},
}

var microprogramacionEspecificacionVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Muestra una especificacion de funcion",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		item, ok, err := cargarEspecificacionFuncionDesdeAPI(id)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion ver")
		}
		if item == nil {
			return fmt.Errorf("especificacion no encontrada")
		}
		fmt.Printf("Especificación #%d — %s\n", item.ID, item.Titulo)
		fmt.Printf("  Archivo:     %s\n", item.ArchivoObjetivo)
		fmt.Printf("  Símbolo:     %s\n", item.SimboloObjetivo)
		fmt.Printf("  Estado:      %s\n", item.Estado)
		fmt.Printf("  Formato:     %s\n", item.FormatoSalida)
		fmt.Printf("  Creado por:  %s\n", item.CreadoPor)
		if item.TareaID != nil {
			fmt.Printf("  Tarea:       %d\n", *item.TareaID)
		}
		if item.ProyectoID != nil {
			fmt.Printf("  Proyecto ID: %d\n", *item.ProyectoID)
		}
		fmt.Printf("  Descripción: %s\n", item.Descripcion)
		if len(item.TestsObligatorios) > 0 {
			fmt.Printf("  Tests:       %s\n", strings.Join(item.TestsObligatorios, " | "))
		}
		if len(item.WriteSet) > 0 {
			fmt.Printf("  Write-set:   %s\n", strings.Join(item.WriteSet, ", "))
		}
		return nil
	},
}

var microprogramacionEspecificacionCrearCmd = &cobra.Command{
	Use:   "crear",
	Short: "Crea una especificacion de funcion",
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		proyecto, _ := cmd.Flags().GetString("proyecto")
		titulo, _ := cmd.Flags().GetString("titulo")
		archivo, _ := cmd.Flags().GetString("archivo")
		simbolo, _ := cmd.Flags().GetString("simbolo")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		precondiciones, _ := cmd.Flags().GetStringSlice("pre")
		postcondiciones, _ := cmd.Flags().GetStringSlice("post")
		depsPermitidas, _ := cmd.Flags().GetStringSlice("dep-permitida")
		depsProhibidas, _ := cmd.Flags().GetStringSlice("dep-prohibida")
		tests, _ := cmd.Flags().GetStringSlice("test")
		writeSet, _ := cmd.Flags().GetStringSlice("write-set")
		formatoSalida, _ := cmd.Flags().GetString("formato-salida")
		creadoPor, _ := resolverValorFlag(cmd, "por")

		resp, ok, err := registrarEspecificacionFuncionPorAPI(apiEspecificacionFuncionCreateRequest{
			TareaID:                tareaID,
			Proyecto:               strings.TrimSpace(proyecto),
			Titulo:                 titulo,
			ArchivoObjetivo:        archivo,
			SimboloObjetivo:        simbolo,
			Descripcion:            descripcion,
			Precondiciones:         precondiciones,
			Postcondiciones:        postcondiciones,
			DependenciasPermitidas: depsPermitidas,
			DependenciasProhibidas: depsProhibidas,
			TestsObligatorios:      tests,
			WriteSet:               writeSet,
			FormatoSalida:          formatoSalida,
			CreadoPor:              creadoPor,
		})
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion crear")
		}
		fmt.Printf("✓ Especificacion creada (id: %d)\n", resp.ID)
		return nil
	},
}

var microprogramacionEspecificacionEmitirCmd = &cobra.Command{
	Use:   "emitir <id>",
	Short: "Emite una microtarea cerrada desde una especificacion",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		contexto, _ := cmd.Flags().GetString("contexto")
		resp, ok, err := emitirMicrotareaPorAPI(id, contexto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion emitir")
		}
		if resp == nil || resp.Microtarea == nil {
			return fmt.Errorf("microtarea no emitida")
		}
		fmt.Printf("Microtarea emitida desde especificacion #%d\n", resp.Microtarea.EspecificacionID)
		fmt.Printf("  Titulo:    %s\n", resp.Microtarea.Titulo)
		fmt.Printf("  Archivo:   %s\n", resp.Microtarea.ArchivoObjetivo)
		fmt.Printf("  Simbolo:   %s\n", resp.Microtarea.SimboloObjetivo)
		fmt.Printf("  Formato:   %s\n", resp.Microtarea.FormatoSalida)
		fmt.Printf("  Write-set: %s\n", strings.Join(resp.Microtarea.WriteSet, ", "))
		fmt.Println("")
		fmt.Println(resp.Microtarea.Mensaje)
		return nil
	},
}

var microprogramacionEspecificacionDespacharCmd = &cobra.Command{
	Use:   "despachar <id>",
	Short: "Despacha una microtarea cerrada a un agente concreto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		contexto, _ := cmd.Flags().GetString("contexto")
		if strings.TrimSpace(agente) == "" {
			return fmt.Errorf("agente obligatorio")
		}
		resp, ok, err := despacharMicrotareaPorAPI(id, apiDespacharMicrotareaRequest{
			Agente:   strings.TrimSpace(agente),
			Proyecto: strings.TrimSpace(proyecto),
			Contexto: strings.TrimSpace(contexto),
		})
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion despachar")
		}
		if resp == nil || resp.Despacho == nil {
			return fmt.Errorf("microtarea no despachada")
		}
		fmt.Printf("Microtarea despachada a %s desde especificacion #%d\n", resp.Despacho.AgenteDestino, resp.Despacho.EspecificacionID)
		fmt.Printf("  Runtime order: %d\n", resp.Despacho.RuntimeOrderID)
		if resp.Despacho.ProyectoID != nil {
			fmt.Printf("  Proyecto ID:   %d\n", *resp.Despacho.ProyectoID)
		}
		fmt.Printf("  Archivo:       %s\n", resp.Despacho.Microtarea.ArchivoObjetivo)
		fmt.Printf("  Simbolo:       %s\n", resp.Despacho.Microtarea.SimboloObjetivo)
		return nil
	},
}

var microprogramacionEspecificacionValidarEntregaCmd = &cobra.Command{
	Use:   "validar-entrega <id>",
	Short: "Valida una entrega contra la especificacion de funcion",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		simbolo, _ := cmd.Flags().GetString("simbolo")
		writeSet, _ := cmd.Flags().GetStringSlice("write-set-entregado")
		deps, _ := cmd.Flags().GetStringSlice("dep-usada")
		testsEjecutados, _ := cmd.Flags().GetStringSlice("test-ejecutado")
		testsFallidos, _ := cmd.Flags().GetStringSlice("test-fallido")
		evidencia, _ := cmd.Flags().GetString("evidencia")
		resp, ok, err := validarEntregaMicrotareaPorAPI(id, apiValidarEntregaRequest{
			SimboloEntregado:   simbolo,
			WriteSetEntregado:  writeSet,
			DependenciasUsadas: deps,
			TestsEjecutados:    testsEjecutados,
			TestsFallidos:      testsFallidos,
			Evidencia:          evidencia,
		})
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion validar-entrega")
		}
		if resp == nil || resp.Resultado == nil {
			return fmt.Errorf("validacion no disponible")
		}
		fmt.Printf("Validación de entrega sobre especificación #%d\n", resp.Resultado.EspecificacionID)
		fmt.Printf("  Archivo:  %s\n", resp.Resultado.ArchivoObjetivo)
		fmt.Printf("  Símbolo:  %s\n", resp.Resultado.SimboloObjetivo)
		if resp.Resultado.Valida {
			fmt.Println("  Resultado: válida")
			return nil
		}
		fmt.Println("  Resultado: inválida")
		for _, hallazgo := range resp.Resultado.Hallazgos {
			fmt.Printf("  - [%s/%s] %s\n", hallazgo.Campo, hallazgo.Codigo, hallazgo.Mensaje)
		}
		return nil
	},
}

var microprogramacionEspecificacionRevisarEntregaCmd = &cobra.Command{
	Use:   "revisar-entrega <id>",
	Short: "Envía el código entregado a un agente revisor para que evalúe calidad y emita correcciones",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("id inválido")
		}
		agente, _ := cmd.Flags().GetString("agente")
		if strings.TrimSpace(agente) == "" {
			return fmt.Errorf("--agente obligatorio: nombre del agente revisor")
		}
		proyecto, _ := cmd.Flags().GetString("proyecto")
		archivo, _ := cmd.Flags().GetString("archivo")

		spec, ok, err := cargarEspecificacionFuncionDesdeAPI(id)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion revisar-entrega")
		}
		if spec == nil {
			return fmt.Errorf("especificacion #%d no encontrada", id)
		}

		archivoObjetivo := strings.TrimSpace(archivo)
		if archivoObjetivo == "" && len(spec.WriteSet) > 0 {
			archivoObjetivo = spec.WriteSet[0]
		}
		if archivoObjetivo == "" {
			archivoObjetivo = spec.ArchivoObjetivo
		}

		codigoActual := ""
		if archivoObjetivo != "" {
			raw, readErr := os.ReadFile(archivoObjetivo)
			if readErr == nil {
				codigoActual = string(raw)
			}
		}

		instruccion := buildRevisionInstruction(id, spec.SimboloObjetivo, archivoObjetivo, spec.Descripcion, spec.TestsObligatorios, codigoActual)
		payload, err := json.Marshal(map[string]string{"texto": instruccion})
		if err != nil {
			return fmt.Errorf("error serializando payload: %w", err)
		}

		orderID, ok, err := crearRuntimeOrderDesdeAPI(strings.TrimSpace(agente), "send_instruction", strings.TrimSpace(proyecto), string(payload))
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("microprogramacion especificacion revisar-entrega")
		}
		fmt.Printf("✓ Revisión de spec #%d enviada a %s (orden #%d)\n", id, agente, orderID)
		fmt.Printf("  Archivo: %s\n", archivoObjetivo)
		fmt.Printf("  Símbolo: %s\n", spec.SimboloObjetivo)
		fmt.Printf("  El agente revisor debe responder APROBADO o CORRECCIONES:<lista>\n")
		return nil
	},
}

// buildRevisionInstruction construye una instrucción de revisión compacta (≤720 chars) para agentes TMUX.
func buildRevisionInstruction(specID int64, simbolo, archivo, descripcion string, tests []string, codigo string) string {
	const maxTotal = 720
	test := ""
	if len(tests) > 0 {
		test = tests[0]
	}
	desc := descripcion
	if len(desc) > 200 {
		desc = desc[:200]
	}
	codigoLineas := ""
	if codigo != "" {
		lineas := strings.Split(codigo, "\n")
		var buf strings.Builder
		for i, l := range lineas {
			if i >= 30 {
				break
			}
			buf.WriteString(l)
			buf.WriteByte('\n')
		}
		codigoLineas = buf.String()
	}
	cabecera := fmt.Sprintf("REVISION SPEC=%d SIM=%s ARC=%s\nOBJ:Di APROBADO o CORRECCIONES:<lista>\nDESC:%s\nTEST:%s\nCODIGO:\n", specID, simbolo, archivo, desc, test)
	disponible := maxTotal - len(cabecera) - 6 // 6 para ``` y ```
	if disponible < 0 {
		disponible = 0
	}
	if len(codigoLineas) > disponible {
		codigoLineas = codigoLineas[:disponible]
	}
	return cabecera + "```\n" + codigoLineas + "```"
}

func init() {
	microprogramacionEspecificacionListarCmd.Flags().Int64("tarea", 0, "Filtra por tarea")
	microprogramacionEspecificacionListarCmd.Flags().String("proyecto", "", "Filtra por proyecto (slug o id)")
	microprogramacionEspecificacionListarCmd.Flags().String("estado", "", "Filtra por estado")
	microprogramacionEspecificacionListarCmd.Flags().Int("limit", 0, "Limita resultados")

	microprogramacionEspecificacionCrearCmd.Flags().Int64("tarea", 0, "Tarea a la que queda ligada")
	microprogramacionEspecificacionCrearCmd.Flags().String("proyecto", "", "Proyecto al que queda ligada (slug o id)")
	microprogramacionEspecificacionCrearCmd.Flags().String("titulo", "", "Titulo de la especificacion")
	microprogramacionEspecificacionCrearCmd.Flags().String("archivo", "", "Archivo objetivo relativo al repo")
	microprogramacionEspecificacionCrearCmd.Flags().String("simbolo", "", "Funcion o simbolo objetivo")
	microprogramacionEspecificacionCrearCmd.Flags().String("descripcion", "", "Descripcion obligatoria")
	microprogramacionEspecificacionCrearCmd.Flags().StringSlice("pre", nil, "Precondiciones")
	microprogramacionEspecificacionCrearCmd.Flags().StringSlice("post", nil, "Postcondiciones")
	microprogramacionEspecificacionCrearCmd.Flags().StringSlice("dep-permitida", nil, "Dependencias permitidas")
	microprogramacionEspecificacionCrearCmd.Flags().StringSlice("dep-prohibida", nil, "Dependencias prohibidas")
	microprogramacionEspecificacionCrearCmd.Flags().StringSlice("test", nil, "Tests obligatorios")
	microprogramacionEspecificacionCrearCmd.Flags().StringSlice("write-set", nil, "Ficheros editables permitidos")
	microprogramacionEspecificacionCrearCmd.Flags().String("formato-salida", "", "Formato de salida esperado")
	microprogramacionEspecificacionCrearCmd.Flags().String("por", "alberto", "Actor que crea la especificacion")
	microprogramacionEspecificacionEmitirCmd.Flags().String("contexto", "", "Contexto adicional acotado para la microtarea")
	microprogramacionEspecificacionDespacharCmd.Flags().String("agente", "", "Agente destino de la microtarea")
	microprogramacionEspecificacionDespacharCmd.Flags().String("proyecto", "", "Proyecto destino si se quiere forzar")
	microprogramacionEspecificacionDespacharCmd.Flags().String("contexto", "", "Contexto adicional acotado para la microtarea")
	microprogramacionEspecificacionValidarEntregaCmd.Flags().String("simbolo", "", "Simbolo declarado por la entrega")
	microprogramacionEspecificacionValidarEntregaCmd.Flags().StringSlice("write-set-entregado", nil, "Archivos tocados por la entrega")
	microprogramacionEspecificacionValidarEntregaCmd.Flags().StringSlice("dep-usada", nil, "Dependencias usadas por la entrega")
	microprogramacionEspecificacionValidarEntregaCmd.Flags().StringSlice("test-ejecutado", nil, "Tests ejecutados por la entrega")
	microprogramacionEspecificacionValidarEntregaCmd.Flags().StringSlice("test-fallido", nil, "Tests fallidos reportados por la entrega")
	microprogramacionEspecificacionValidarEntregaCmd.Flags().String("evidencia", "", "Evidencia verificable de la entrega")
	microprogramacionEspecificacionRevisarEntregaCmd.Flags().String("agente", "", "Agente revisor destino (obligatorio)")
	microprogramacionEspecificacionRevisarEntregaCmd.Flags().String("proyecto", "", "Proyecto asociado")
	microprogramacionEspecificacionRevisarEntregaCmd.Flags().String("archivo", "", "Archivo a revisar (por defecto el primero del write-set)")

	microprogramacionEspecificacionCmd.AddCommand(
		microprogramacionEspecificacionListarCmd,
		microprogramacionEspecificacionVerCmd,
		microprogramacionEspecificacionCrearCmd,
		microprogramacionEspecificacionEmitirCmd,
		microprogramacionEspecificacionDespacharCmd,
		microprogramacionEspecificacionValidarEntregaCmd,
		microprogramacionEspecificacionRevisarEntregaCmd,
	)
	microprogramacionCmd.AddCommand(microprogramacionEspecificacionCmd)
}
