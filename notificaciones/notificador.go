package notificaciones

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"orquesta/db"
	"strconv"
	"strings"
	"time"
)

type Notificador interface {
	EnviarMensaje(texto string) error
	EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error
	EnviarPropuestaVotacion(codigo, titulo string) error
	EnviarAvisoFinProyecto(proyectoID int64, nombre string) error
}

type TelegramNotificador struct {
	Token  string
	ChatID string
}

func (n *TelegramNotificador) EnviarMensaje(texto string) error {
	if n.Token == "" || n.ChatID == "" {
		return nil
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.Token)
	payload := map[string]string{
		"chat_id":    n.ChatID,
		"text":       texto,
		"parse_mode": "Markdown",
	}
	return n.postJSON(url, payload)
}

func (n *TelegramNotificador) EnviarAlertaBloqueo(tareaID int64, agente, motivo string) error {
	texto := fmt.Sprintf("🚨 *Bloqueo Detectado*\n\nTarea #%d\nAgente: `%s` \nMotivo: _%s_", tareaID, agente, motivo)
	return n.EnviarMensaje(texto)
}

func (n *TelegramNotificador) EnviarPropuestaVotacion(codigo, titulo string) error {
	texto := fmt.Sprintf("📩 *Nueva Propuesta (%s)*\n\n%s", codigo, titulo)

	payload := map[string]any{
		"chat_id":    n.ChatID,
		"text":       texto,
		"parse_mode": "Markdown",
		"reply_markup": map[string]any{
			"inline_keyboard": [][]map[string]any{
				{
					{"text": "✅ Acuerdo", "callback_data": fmt.Sprintf("voto:%s:acuerdo", codigo)},
					{"text": "❌ Desacuerdo", "callback_data": fmt.Sprintf("voto:%s:desacuerdo", codigo)},
					{"text": "⚖️ Abstención", "callback_data": fmt.Sprintf("voto:%s:abstencion", codigo)},
				},
			},
		},
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.Token)
	return n.postJSON(url, payload)
}

func (n *TelegramNotificador) EnviarAvisoFinProyecto(proyectoID int64, nombre string) error {
	texto := fmt.Sprintf("🎉 *¡PROYECTO FINALIZADO!*\n\nLa aplicación *%s* (#%d) ha sido completada al 100%% sin intervención humana adicional. ¡Buen trabajo equipo!", nombre, proyectoID)
	return n.EnviarMensaje(texto)
}

func (n *TelegramNotificador) postJSON(url string, data any) error {
	body, _ := json.Marshal(data)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error telegram (status %d): %s", resp.StatusCode, string(rb))
	}
	return nil
}

// GlobalNotificador se inicializa si hay configuración disponible.
// Puede ser Telegram, OpenClaw Gateway o un fanout de ambos.
var GlobalNotificador Notificador

func InicializarDesdeConfig() {
	GlobalNotificador = nil
	notifier, telegram, labels := buildConfiguredNotifiers()
	GlobalNotificador = notifier
	for _, label := range labels {
		fmt.Printf("✓ %s\n", label)
	}
	if telegram != nil {
		go StartPolling(telegram)
	}
}

// StartPolling escucha órdenes del administrador
func StartPolling(n *TelegramNotificador) {
	offset := 0
	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", n.Token, offset)
		resp, err := http.Get(url)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		var updateResp struct {
			Ok     bool `json:"ok"`
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  *struct {
					Text string `json:"text"`
					From struct {
						ID       int64  `json:"id"`
						UserName string `json:"username"`
					} `json:"from"`
					Chat struct {
						ID int64 `json:"id"`
					} `json:"chat"`
				} `json:"message"`
				CallbackQuery *struct {
					ID      string `json:"id"`
					Data    string `json:"data"`
					Message *struct {
						Chat struct {
							ID int64 `json:"id"`
						} `json:"chat"`
					} `json:"message"`
				} `json:"callback_query"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		for _, upd := range updateResp.Result {
			offset = upd.UpdateID + 1

			// 1. Manejo de Botones (Callback)
			if upd.CallbackQuery != nil {
				if strconv.FormatInt(upd.CallbackQuery.Message.Chat.ID, 10) != n.ChatID {
					fmt.Printf("⚠️ Callback ignorado de ID desconocido: %d\n", upd.CallbackQuery.Message.Chat.ID)
					continue
				}
				procesarCallback(n, upd.CallbackQuery.Data)
				continue
			}

			// 2. Manejo de Comandos (Texto)
			if upd.Message != nil {
				if strconv.FormatInt(upd.Message.Chat.ID, 10) != n.ChatID {
					fmt.Printf("⚠️ Mensaje ignorado de ID desconocido: %d (%s)\n", upd.Message.Chat.ID, upd.Message.From.UserName)
					continue
				}
				procesarComando(n, upd.Message.Text)
			}
		}
	}
}

func procesarCallback(n *TelegramNotificador, data string) {
	parts := strings.Split(data, ":")
	if len(parts) == 3 && parts[0] == "voto" {
		codigo := parts[1]
		voto := parts[2]
		p, err := db.GetPropuesta(codigo)
		if err != nil {
			n.EnviarMensaje("❌ Error: Propuesta no encontrada.")
			return
		}
		_, err = db.Votar(p.ID, "alberto", db.PosicionVoto(voto), "Votado vía Telegram")
		if err != nil {
			n.EnviarMensaje("❌ Error registrando voto: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("✅ Voto *%s* registrado para la propuesta *%s*", voto, codigo))
		}
	}
}

func procesarComando(n *TelegramNotificador, text string) {
	args := strings.Fields(text)
	if len(args) == 0 {
		return
	}

	cmdName := strings.Split(args[0], "@")[0]
	switch cmdName {
	case "/proyectos":
		proyectos, _ := db.ListarProyectosActivos()
		if len(proyectos) == 0 {
			n.EnviarMensaje("🤷‍♂️ No hay proyectos activos en el orquestador.")
			return
		}
		resBody := "📁 *Proyectos en el Orquestador:*\n\n"
		for _, p := range proyectos {
			status := "🟢 Activo"
			if !p.Activo {
				status = "⚪️ Inactivo"
			}
			resBody += fmt.Sprintf("• *%s*\n  👉 Nombre para la terminal: `%s`\n  _Status: %s_\n\n", p.Nombre, p.Slug, status)
		}
		resBody += "💡 Para lanzar un agente en este proyecto, usa:\n`./agente [nombre] -p [nombre_terminal]`"
		n.EnviarMensaje(resBody)

	case "/status", "/dashboard":
		m, _ := db.ContarTareasPorEstado()
		total := 0
		for _, v := range m {
			total += v
		}
		res := fmt.Sprintf("📊 *Estado de Orquesta*\n\n- Tareas Totales: %d\n- En Progreso: %d\n- Bloqueadas: %d\n- Completadas: %d",
			total, m["en_progreso"], m["bloqueada"], m["completada"])
		n.EnviarMensaje(res)

	case "/progreso":
		// Si no hay argumentos, sacamos el progreso general. Si hay, del proyecto.
		proyectos, _ := db.ListarProyectosActivos()
		if len(proyectos) == 0 {
			n.EnviarMensaje("🤷‍♂️ No hay proyectos activos ahora mismo.")
			return
		}

		resBody := "📈 *Progreso de Proyectos*\n\n"
		for _, p := range proyectos {
			stats, _ := db.GetEstadisticasProyecto(p.ID)
			pct := 0
			if stats.Total > 0 {
				pct = (stats.Completadas * 100) / stats.Total
			}

			// Barra visual
			blocks := pct / 10
			bar := strings.Repeat("█", blocks) + strings.Repeat("░", 10-blocks)

			gitAlert := ""
			if gitMsg, _ := db.GetEstadoGit(p.RutaAbs); gitMsg != "" {
				gitAlert = " 📦⚠️ _Cambios pendientes en Git_"
			}

			resBody += fmt.Sprintf("*%s*\n`[%s]` %d%%%s\n_%d de %d tareas_\n\n", p.Nombre, bar, pct, gitAlert, stats.Completadas, stats.Total)
		}
		n.EnviarMensaje(resBody)

	case "/git":
		proyectos, _ := db.ListarProyectosActivos()
		resBody := "📦 *Estado de Git en Proyectos*\n\n"
		found := false
		for _, p := range proyectos {
			if gitMsg, _ := db.GetEstadoGit(p.RutaAbs); gitMsg != "" {
				resBody += fmt.Sprintf("⚠️ *%s*:\n`%s`\n", p.Nombre, gitMsg)
				found = true
			}
		}
		if !found {
			resBody = "✅ Todos los proyectos están limpios y sincronizados con Git."
		}
		n.EnviarMensaje(resBody)

	case "/desbloquear":
		if len(args) == 1 {
			// Listar bloqueos actuales
			bloqueos, _ := db.ListarResumenBloqueos()
			if len(bloqueos) == 0 {
				n.EnviarMensaje("✅ No hay tareas bloqueadas en este momento.")
				return
			}
			res := "🚨 *Tareas Bloqueadas actualmente:*\n\n"
			for _, b := range bloqueos {
				res += fmt.Sprintf("• *ID #%d*: %s\n  _Bq por: %s_\n  _Motivo: %s_\n\n", b.ID, b.Titulo, b.Agente, b.Motivo)
			}
			res += "👉 Usa `/desbloquear [ID] [resolución]` para liberar una."
			n.EnviarMensaje(res)
			return
		}
		if len(args) < 3 {
			n.EnviarMensaje("⚠️ Uso: `/desbloquear [id_tarea] [resolución]`")
			return
		}
		id, _ := strconv.ParseInt(args[1], 10, 64)
		resolucion := strings.Join(args[2:], " ")
		err := db.DesbloquearTarea(id, "alberto", resolucion)
		if err != nil {
			n.EnviarMensaje("❌ Error: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("🔓 Tarea #%d desbloqueada con éxito.", id))
		}

	case "/pausar":
		if len(args) < 3 {
			n.EnviarMensaje("⚠️ Uso: `/pausar [agente] [minutos]`")
			return
		}
		min, _ := strconv.Atoi(args[2])
		err := db.PausarAgente(args[1], min, "Pausa remota vía Telegram")
		if err != nil {
			n.EnviarMensaje("❌ Error: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("💤 Agente %s pausado por %d min.", args[1], min))
		}

	case "/reanimar", "/despertar":
		if len(args) < 2 {
			n.EnviarMensaje("⚠️ Uso: `/[reanimar|despertar] [agente]`")
			return
		}
		err := db.ResetReanimacion(args[1])
		if err != nil {
			n.EnviarMensaje("❌ Error: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("🌞 Agente %s reanimado con éxito.", args[1]))
		}

	case "/config":
		if len(args) == 1 {
			configs, _ := db.ConfigAll()
			res := "⚙️ *Configuración Actual de Orquesta:*\n\n"
			for k, v := range configs {
				res += fmt.Sprintf("• `%s` = `%s`\n", k, v)
			}
			res += "\n👉 Usa `/config [clave] [valor]` para cambiar algo."
			n.EnviarMensaje(res)
			return
		}
		if len(args) < 3 {
			n.EnviarMensaje("⚠️ Uso: `/config [clave] [valor]`")
			return
		}
		err := db.ConfigSet(args[1], args[2])
		if err != nil {
			n.EnviarMensaje("❌ Error: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("⚙️ Configuración actualizada: `%s = %s`", args[1], args[2]))
		}

	case "/nuevo_proyecto":
		if len(args) < 4 {
			n.EnviarMensaje("⚠️ Uso: `/nuevo_proyecto [slug] [nombre] [ruta_absoluta]`")
			return
		}
		p := &db.Proyecto{
			Slug:    args[1],
			Nombre:  args[2],
			RutaAbs: args[3],
			Tipo:    db.ProyectoRepo,
			Activo:  true,
		}
		id, err := db.UpsertProyecto(p)
		if err != nil {
			n.EnviarMensaje("❌ Error: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("🚀 Proyecto registrado con éxito (ID: %d)", id))
		}

	case "/nueva_tarea":
		if len(args) < 3 {
			n.EnviarMensaje("⚠️ Uso: `/nueva_tarea [proyecto] [titulo] [descripcion...]`")
			return
		}
		p, err := db.GetProyecto(args[1])
		if err != nil {
			n.EnviarMensaje("❌ Proyecto no encontrado.")
			return
		}
		desc := ""
		if len(args) > 3 {
			desc = strings.Join(args[3:], " ")
		}
		t := &db.Tarea{
			ProyectoID:  &p.ID,
			Titulo:      args[2],
			Descripcion: desc,
			Estado:      db.TareaBacklog,
			Prioridad:   db.PrioridadMedia,
			CreadoPor:   "telegram_admin",
		}
		id, err := db.CrearTarea(t)
		if err != nil {
			n.EnviarMensaje("❌ Error: " + err.Error())
		} else {
			n.EnviarMensaje(fmt.Sprintf("📝 Tarea #%d creada en el backlog de %s.", id, p.Nombre))
		}

	case "/reglas":
		tipo := "programador"
		if len(args) > 1 {
			tipo = args[1]
		}
		reglas, _ := db.GetReglasAgente(tipo)
		if len(reglas) == 0 {
			n.EnviarMensaje(fmt.Sprintf("📜 No hay reglas activas para *%s*.", tipo))
			return
		}
		res := fmt.Sprintf("📜 *Reglas de %s:*\n\n", tipo)
		for _, r := range reglas {
			res += fmt.Sprintf("• *%s*: %s\n", r.Titulo, r.Descripcion)
		}
		n.EnviarMensaje(res)

	case "/nueva_regla":
		// /nueva_regla [agente] [cat] [titulo] | [desc]
		txt := strings.Join(args[1:], " ")
		parts := strings.Split(txt, "|")
		if len(parts) < 2 {
			n.EnviarMensaje("⚠️ Uso: `/nueva_regla [agente] [cat] [titulo] | [descripcion]`")
			return
		}
		head := strings.Fields(parts[0])
		if len(head) < 3 {
			n.EnviarMensaje("⚠️ Formato cabecera inválido.")
			return
		}
		r := &db.Regla{
			TipoAgente:  head[0],
			Categoria:   head[1],
			Titulo:      strings.Join(head[2:], " "),
			Descripcion: strings.TrimSpace(parts[1]),
			Activa:      true,
		}
		db.UpsertRegla(r)
		n.EnviarMensaje(fmt.Sprintf("✅ Regla '%s' inyectada para %s.", r.Titulo, r.TipoAgente))

	case "/skills":
		tipo := "programador"
		if len(args) > 1 {
			tipo = args[1]
		}
		skills, _ := db.GetSkillsAgente(tipo)
		if len(skills) == 0 {
			n.EnviarMensaje(fmt.Sprintf("🧠 No hay skills activas para *%s*.", tipo))
			return
		}
		res := fmt.Sprintf("🧠 *Skills de %s:*\n\n", tipo)
		for _, s := range skills {
			res += fmt.Sprintf("• *%s*: %s\n", s.Nombre, s.Descripcion)
		}
		n.EnviarMensaje(res)

	case "/nueva_skill":
		// /nueva_skill [agente] [nombre] [desc] | [cuando_usar]
		txt := strings.Join(args[1:], " ")
		parts := strings.Split(txt, "|")
		if len(parts) < 2 {
			n.EnviarMensaje("⚠️ Uso: `/nueva_skill [persona] [nombre] [desc] | [contexto]`")
			return
		}
		head := strings.Fields(parts[0])
		if len(head) < 2 {
			return
		}
		s := &db.Skill{
			TipoAgente:  head[0],
			Nombre:      strings.Join(head[1:], " "),
			Descripcion: strings.TrimSpace(parts[1]),
			CuandoUsar:  "",
			Activa:      true,
		}
		if len(parts) > 2 {
			s.CuandoUsar = strings.TrimSpace(parts[2])
		}
		db.UpsertSkill(s)
		n.EnviarMensaje(fmt.Sprintf("✅ Skill '%s' entrenada para %s.", s.Nombre, s.TipoAgente))

	case "/workflows":
		tipo := "programador"
		if len(args) > 1 {
			tipo = args[1]
		}
		wfs, _ := db.GetWorkflowsAgente(tipo)
		res := fmt.Sprintf("🔄 *Workflows de %s:*\n\n", tipo)
		for _, w := range wfs {
			res += fmt.Sprintf("• `%s`: %s\n", w.Nombre, w.Descripcion)
		}
		res += "\n👉 Usa `/ver_workflow [nombre]` para ver los pasos."
		n.EnviarMensaje(res)

	case "/ver_workflow":
		if len(args) < 2 {
			return
		}
		// Buscamos en cualquier agente
		var wf *db.Workflow
		for _, t := range []string{"programador", "documentador", "admin"} {
			if w, err := db.GetWorkflow(t, args[1]); err == nil {
				wf = w
				break
			}
		}
		if wf == nil {
			n.EnviarMensaje("❌ Workflow no encontrado.")
			return
		}
		res := fmt.Sprintf("🔄 *Workflow: %s*\n_%s_\n\n", wf.Nombre, wf.Descripcion)
		var pasos []string
		json.Unmarshal([]byte(wf.Pasos), &pasos)
		for i, p := range pasos {
			res += fmt.Sprintf("%d. %s\n", i+1, p)
		}
		n.EnviarMensaje(res)

	case "/agentes":
		agentes, _ := db.ListarAgenteInfo()
		if len(agentes) == 0 {
			n.EnviarMensaje("🤷 No hay agentes registrados.")
			return
		}
		res := "🤖 *Inventario de Agentes Orquesta:*\n\n"
		for _, a := range agentes {
			status := "💤"
			if a.Activo {
				status = "⚡️ *Activo*"
			}
			if a.EstadoCuota == "enfriamiento" {
				status = "☕️ _Enfriando_"
			}

			res += fmt.Sprintf("• *%s* [%s] | %s\n", a.Nombre, a.Rol, status)
			res += fmt.Sprintf("  🔋 Hoy: %d/%d s | %s\n", a.ConsumoDia, a.LimiteDia, a.EstadoCuota)
			if a.MotivoPausa != "" {
				res += fmt.Sprintf("  🚫 _%s_\n", a.MotivoPausa)
			}
			res += "\n"
		}
		n.EnviarMensaje(res)

	case "/limpiar_agentes":
		db.ResetEstadoAgentes()
		n.EnviarMensaje("🧹 *Estados fantasma limpiados.* Todos los agentes están ahora marcados como 'Inactivos'. Solo los procesos reales volverán a marcarse como activos.")

	case "/retirar":
		if len(args) < 2 {
			n.EnviarMensaje("⚠️ Uso: `/retirar [nombre_agente]`")
			return
		}
		db.AgenteSetHabilitado(args[1], false)
		n.EnviarMensaje(fmt.Sprintf("🚫 Agente *%s* retirado del servicio.", args[1]))

	case "/rehabilitar":
		if len(args) < 2 {
			n.EnviarMensaje("⚠️ Uso: `/rehabilitar [nombre_agente]`")
			return
		}
		db.AgenteSetHabilitado(args[1], true)
		n.EnviarMensaje(fmt.Sprintf("✅ Agente *%s* rehabilitado y listo pars el servicio.", args[1]))

	case "/ayuda", "/start":
		res := "🎮 *Consola de Mando Orquesta*\n\n"
		res += "📊 `/status` - Resumen general\n"
		res += "📈 `/progreso` - Avance de proyectos y Git\n"
		res += "📁 `/proyectos` - Listado de proyectos activos\n"
		res += "📦 `/git` - Cambios pendientes en repositorios\n"
		res += "🔓 `/desbloquear [id] [res]` - Resolver bloqueo\n"
		res += "📩 `/votar [op] [voto]` - Votar propuesta\n"
		res += "💤 `/pausar [agente] [min]` - Pausar agente\n"
		res += "🌞 `/reanimar [agente]` - Despertar agente\n"
		res += "🚀 `/nuevo_proyecto [slug] [nom] [ruta]`\n"
		res += "📝 `/nueva_tarea [proj] [tit] [desc]`\n"
		res += "⚙️ `/config [k] [v]` - Ajustes globales\n\n"
		res += "🧠 *Inteligencia Colectiva*:\n"
		res += "📜 `/reglas [agente]` | `/nueva_regla`\n"
		res += "🧠 `/skills [agente]` | `/nueva_skill`\n"
		res += "🔄 `/workflows [agente]` | `/ver_workflow`"
		res += "\n🤖 `/agentes` | `/limpiar_agentes`"
		res += "\n🛡️ `/retirar [nom]` | `/rehabilitar [nom]`"
		n.EnviarMensaje(res)
	}
}
