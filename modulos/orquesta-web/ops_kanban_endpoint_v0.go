package orquestaweb

import "net/http"

const WebOpsKanbanPageEndpointV0 = "/ops/kanban"

type OpsKanbanWebEndpointV0 struct{}

func NewOpsKanbanWebEndpointV0() OpsKanbanWebEndpointV0 {
	return OpsKanbanWebEndpointV0{}
}

func (endpoint OpsKanbanWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != WebOpsKanbanPageEndpointV0 {
		http.NotFound(w, r)
		return
	}
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setWebPublicHTTPAllowV0(w, http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	writeWebHTMLStringResponseV0(w, http.StatusOK, opsKanbanHTMLV0(), "es")
}

func opsKanbanHTMLV0() string {
	return `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Orquesta Kanban</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #101316;
      --panel: #181d22;
      --panel-2: #202832;
      --line: #34404a;
      --text: #eef4f8;
      --muted: #aab7c3;
      --good: #32d583;
      --warn: #fdb022;
      --bad: #f97066;
      --info: #7cd4fd;
      --ink: #0b0d10;
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; min-height: 100%; background: var(--bg); color: var(--text); font: 14px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; letter-spacing: 0; }
    body { padding: 18px; }
    .shell { max-width: 1780px; margin: 0 auto; display: grid; gap: 12px; }
    header { display: flex; justify-content: space-between; align-items: flex-start; gap: 14px; }
    h1 { margin: 0; font-size: 22px; line-height: 1.1; }
    h2, h3 { margin: 0; }
    h2 { font-size: 14px; }
    h3 { font-size: 13px; }
    p { margin: 0; }
    button, .navlink { border: 1px solid var(--line); background: var(--panel-2); color: var(--text); border-radius: 6px; padding: 6px 10px; cursor: pointer; text-decoration: none; }
    button:hover, .navlink:hover { border-color: var(--info); }
    input, select { width: 100%; min-height: 32px; border: 1px solid var(--line); border-radius: 6px; background: #0f1215; color: var(--text); padding: 6px 8px; font: inherit; }
    .sub { color: var(--muted); font-size: 13px; }
    .toolbar { display: flex; align-items: center; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
    .pill { display: inline-flex; align-items: center; gap: 6px; min-height: 28px; padding: 4px 10px; border: 1px solid var(--line); background: var(--panel); border-radius: 6px; color: var(--muted); }
    .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--muted); display: inline-block; }
    .dot.ok { background: var(--good); }
    .dot.warn { background: var(--warn); }
    .dot.bad { background: var(--bad); }
    .notice { border: 1px solid rgba(124,212,253,.36); background: rgba(124,212,253,.07); border-radius: 8px; padding: 10px 12px; color: var(--muted); }
    .kpis { display: grid; grid-template-columns: repeat(5, minmax(140px, 1fr)); gap: 10px; }
    .kpi { min-height: 82px; border: 1px solid var(--line); background: var(--panel); border-radius: 8px; padding: 10px; }
    .kpi .label { color: var(--muted); font-size: 12px; text-transform: uppercase; }
    .kpi .value { margin-top: 6px; font-size: 25px; font-weight: 750; }
    .kpi .hint { color: var(--muted); font-size: 12px; margin-top: 3px; overflow-wrap: anywhere; }
    .filters { display: grid; grid-template-columns: minmax(240px, 1fr) 180px 180px 170px; gap: 8px; align-items: end; }
    .board { display: grid; grid-template-columns: repeat(6, minmax(220px, 1fr)); gap: 10px; align-items: start; overflow-x: auto; padding-bottom: 4px; }
    .lane { min-width: 220px; min-height: 380px; border: 1px solid var(--line); background: rgba(24,29,34,.86); border-radius: 8px; overflow: hidden; }
    .lane-head { min-height: 58px; padding: 10px; border-bottom: 1px solid var(--line); display: grid; gap: 3px; background: rgba(255,255,255,.02); }
    .lane-title { display: flex; justify-content: space-between; gap: 8px; font-weight: 750; }
    .lane-title span:last-child { color: var(--muted); }
    .lane-body { display: grid; gap: 8px; padding: 9px; }
    .card { border: 1px solid var(--line); background: #11161b; border-radius: 8px; padding: 9px; display: grid; gap: 7px; }
    .card.attention { border-color: rgba(249,112,102,.7); }
    .card.reclaim { border-color: rgba(253,176,34,.75); }
    .card.done { border-color: rgba(50,213,131,.6); }
    .card-title { font-weight: 750; overflow-wrap: anywhere; }
    .card-meta { display: grid; gap: 4px; color: var(--muted); font-size: 12px; }
    .row { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; min-width: 0; }
    .tag { display: inline-flex; align-items: center; min-height: 20px; max-width: 100%; border-radius: 999px; padding: 2px 7px; background: var(--panel-2); color: var(--muted); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .tag.ok { background: var(--good); color: var(--ink); }
    .tag.warn { background: var(--warn); color: var(--ink); }
    .tag.bad { background: var(--bad); color: var(--ink); }
    .tag.info { background: var(--info); color: var(--ink); }
    .bar { position: relative; height: 7px; border-radius: 999px; overflow: hidden; background: #2a3138; }
    .bar span { display: block; height: 100%; width: 0%; background: var(--info); }
    .bar span.ok { background: var(--good); }
    .bar span.warn { background: var(--warn); }
    .bar span.bad { background: var(--bad); }
    .details { display: grid; grid-template-columns: minmax(0, 1fr) minmax(320px, .45fr); gap: 10px; }
    .panel { border: 1px solid var(--line); background: var(--panel); border-radius: 8px; overflow: hidden; }
    .panel h2 { padding: 10px 12px; border-bottom: 1px solid var(--line); background: rgba(255,255,255,.02); }
    .pad { padding: 10px 12px; }
    .list { display: grid; gap: 8px; }
    .empty { color: var(--muted); padding: 14px 10px; }
    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    pre { margin: 0; max-height: 320px; overflow: auto; padding: 10px; white-space: pre-wrap; word-break: break-word; background: #0b0d10; border-top: 1px solid rgba(255,255,255,.08); font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
    @media (max-width: 1180px) {
      .kpis { grid-template-columns: repeat(3, minmax(140px, 1fr)); }
      .details { grid-template-columns: 1fr; }
    }
    @media (max-width: 760px) {
      body { padding: 12px; }
      header { display: grid; }
      .toolbar { justify-content: flex-start; }
      .kpis, .filters { grid-template-columns: 1fr; }
      .board { grid-template-columns: repeat(6, minmax(240px, 82vw)); }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header>
      <div>
        <h1>Kanban de observacion</h1>
        <p class="sub">Vista derivada de la cola, estado vivo, stats y acciones publicadas. No persiste columnas ni cambia tareas.</p>
      </div>
      <nav class="toolbar" aria-label="navegacion">
        <span class="pill"><span id="live-dot" class="dot warn"></span><span id="live-text">conectando</span></span>
        <span class="pill">ultima lectura <strong id="last-refresh">-</strong></span>
        <a class="navlink" href="/">Inicio</a>
        <a class="navlink" href="/ops">Ops</a>
        <a class="navlink" href="/autoprogramming">Autoprogramar</a>
        <button id="refresh-now" type="button">Actualizar</button>
      </nav>
    </header>

    <section class="notice">
      Esta pantalla es solo una proyeccion read-only. La fuente de verdad sigue en los contratos existentes:
      <span class="mono">/api/v0/autoprogramming/status</span> y <span class="mono">/api/v0/queue/global-status</span>.
    </section>

    <section class="kpis" aria-label="resumen">
      <div class="kpi"><div class="label">Cola</div><div id="kpi-ready" class="value">-</div><div id="kpi-ready-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Ejecucion</div><div id="kpi-running" class="value">-</div><div id="kpi-running-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Atencion</div><div id="kpi-attention" class="value">-</div><div id="kpi-attention-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Cierre</div><div id="kpi-closure" class="value">-</div><div id="kpi-closure-hint" class="hint">-</div></div>
      <div class="kpi"><div class="label">Agentes</div><div id="kpi-agents" class="value">-</div><div id="kpi-agents-hint" class="hint">-</div></div>
    </section>

    <section class="filters" aria-label="filtros">
      <label><span class="sub">Buscar</span><input id="filter-text" type="search" placeholder="run, app, agente, fase o evidencia"></label>
      <label><span class="sub">App</span><input id="filter-app" type="search" placeholder="app_ref"></label>
      <label><span class="sub">Estado</span><select id="filter-state"><option value="">Todos</option><option value="ready">Cola</option><option value="running">Ejecucion</option><option value="waiting">Espera</option><option value="attention">Atencion</option><option value="closure">Cierre</option><option value="done">Hecho</option></select></label>
      <label><span class="sub">Modo</span><select id="filter-kind"><option value="">Todo</option><option value="goal">Goal-first</option><option value="legacy">Legacy</option><option value="agent">Agentes</option><option value="issue">Incidencias</option></select></label>
    </section>

    <section class="board" aria-label="kanban de trabajo">
      <section class="lane" data-lane="ready"><div class="lane-head"><div class="lane-title"><span>Cola</span><span id="count-ready">0</span></div><p class="sub">Backlog, ready o queued.</p></div><div id="lane-ready" class="lane-body"></div></section>
      <section class="lane" data-lane="running"><div class="lane-head"><div class="lane-title"><span>Ejecucion</span><span id="count-running">0</span></div><p class="sub">Runs o agentes con actividad.</p></div><div id="lane-running" class="lane-body"></div></section>
      <section class="lane" data-lane="waiting"><div class="lane-head"><div class="lane-title"><span>Espera</span><span id="count-waiting">0</span></div><p class="sub">Waits, review o dependencias.</p></div><div id="lane-waiting" class="lane-body"></div></section>
      <section class="lane" data-lane="attention"><div class="lane-head"><div class="lane-title"><span>Atencion</span><span id="count-attention">0</span></div><p class="sub">Bloqueo, stale, error o accion requerida.</p></div><div id="lane-attention" class="lane-body"></div></section>
      <section class="lane" data-lane="closure"><div class="lane-head"><div class="lane-title"><span>Cierre</span><span id="count-closure">0</span></div><p class="sub">Validacion, accepted review o cierre.</p></div><div id="lane-closure" class="lane-body"></div></section>
      <section class="lane" data-lane="done"><div class="lane-head"><div class="lane-title"><span>Hecho</span><span id="count-done">0</span></div><p class="sub">Terminales aceptados o completados.</p></div><div id="lane-done" class="lane-body"></div></section>
    </section>

    <section class="details">
      <div class="panel">
        <h2>Diagnosticos y acciones publicadas</h2>
        <div id="diagnostics" class="pad list"><div class="empty">Sin datos todavia</div></div>
      </div>
      <div class="panel">
        <h2>Fuentes</h2>
        <div class="pad sub">Resumen de la ultima lectura. No se guarda estado local.</div>
        <pre id="source-summary">{}</pre>
      </div>
    </section>
  </main>
  <script>
    const lanes = ['ready', 'running', 'waiting', 'attention', 'closure', 'done'];
    const state = {auto: {}, globalStatus: {}, cards: [], diagnostics: []};
    const refreshMs = 2500;
    let refreshTimer = null;

    document.getElementById('refresh-now').addEventListener('click', refreshAll);
    ['filter-text', 'filter-app', 'filter-state', 'filter-kind'].forEach(function(id) {
      document.getElementById(id).addEventListener('input', render);
    });

    async function postJSON(url, payload) {
      const response = await fetch(url, {
        method: 'POST',
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: JSON.stringify(payload || {})
      });
      const data = await response.json().catch(function() {
        return {estado: 'error', errores_publicos: [{code: 'respuesta_no_json'}]};
      });
      if (!response.ok) {
        const error = new Error((data.errores_publicos && data.errores_publicos[0] && data.errores_publicos[0].code) || ('http_' + response.status));
        error.payload = data;
        throw error;
      }
      return data;
    }

    async function refreshAll() {
      setLive('warn', 'leyendo');
      const id = 'web-ops-kanban-' + Date.now().toString(36);
      const query = {request_id: id, correlation_id: id, queue_ref: 'global', queue_limit: 80, include_agent_progress: true, include_process_refs: true, include_agent_usage: true};
      const result = await Promise.allSettled([
        postJSON('/api/v0/autoprogramming/status', query),
        postJSON('/api/v0/queue/global-status', query)
      ]);
      state.diagnostics = [];
      if (result[0].status === 'fulfilled') {
        state.auto = result[0].value || {};
      } else {
        state.auto = {};
        state.diagnostics.push(diagnosticFromError('autoprogramming_status_unavailable', result[0].reason));
      }
      if (result[1].status === 'fulfilled') {
        state.globalStatus = result[1].value || {};
      } else {
        state.globalStatus = {};
        state.diagnostics.push(diagnosticFromError('queue_global_status_unavailable', result[1].reason));
      }
      state.cards = buildCards(state.auto, state.globalStatus);
      render();
      setLive(state.diagnostics.length ? 'warn' : 'ok', state.diagnostics.length ? 'parcial' : 'live');
      document.getElementById('last-refresh').textContent = new Date().toLocaleTimeString('es-ES');
      clearTimeout(refreshTimer);
      refreshTimer = setTimeout(refreshAll, refreshMs);
    }

    function buildCards(auto, globalStatus) {
      const cards = [];
      const seen = new Set();
      arrayAt(auto, ['queue', 'ranked']).forEach(function(item) {
        addCard(cards, seen, runCard(item, 'run_queue'));
      });
      arrayAt(auto, ['runs']).forEach(function(item) {
        addCard(cards, seen, runCard(item, 'autoprogramming_status'));
      });
      if (auto.run) addCard(cards, seen, runCard(auto.run, 'director_stats'));
      queueGlobalItems(globalStatus).forEach(function(item) {
        addCard(cards, seen, runCard(item, 'queue_global_status'));
      });
      arrayAt(auto, ['agents']).forEach(function(agent) {
        addCard(cards, seen, agentCard(agent));
      });
      arrayAt(auto, ['stale_running']).forEach(function(issue) {
        addCard(cards, seen, issueCard(issue));
      });
      safeActions(auto).forEach(function(action) {
        if ((action.reason || action.action || '').toLowerCase().includes('no_action')) return;
        addCard(cards, seen, actionCard(action));
      });
      return cards.sort(function(a, b) {
        return laneWeight(a.lane) - laneWeight(b.lane) || priorityWeight(b) - priorityWeight(a) || a.title.localeCompare(b.title);
      });
    }

    function runCard(item, source) {
      const runRef = text(item.run_ref || item.runRef || item.ref || item.id || item.task_ref || item.taskRef);
      const appRef = text(item.app_ref || item.appRef || item.project_ref || item.projectRef || item.queue_ref || item.queueRef);
      const status = normalizeToken(item.status || item.estado || item.run_status || item.runStatus || item.queue_status || item.queueStatus);
      const phase = text(item.current_phase || item.currentPhase || item.phase || item.phase_id || item.phaseID || item.step || item.workflow_phase);
      const closure = normalizeToken(item.closure_status || item.closureStatus || item.closure || item.validation_status || item.validationStatus);
      const pct = number(item.percent_complete ?? item.percentComplete ?? item.progress_percent ?? item.progress ?? item.priority_score ?? 0);
      const agents = number(item.agents_in_flight ?? item.agentsInFlight ?? item.progressing_agents ?? item.progressingAgents ?? item.agents_live ?? item.agentsLive ?? 0);
      const failedAgents = number(item.agents_failed ?? item.agentsFailed ?? item.stalled_agents ?? item.stalledAgents ?? 0);
      const tasksClosed = number(item.tasks_closed ?? item.tasksClosed ?? item.closed_tasks ?? item.closedTasks ?? 0);
      const tasksTotal = number(item.tasks_total ?? item.tasksTotal ?? item.total_tasks ?? item.totalTasks ?? 0);
      const mode = normalizeToken(item.director_execution_mode || item.directorExecutionMode || item.execution_mode || item.executionMode);
      const attention = Boolean(item.needs_attention || item.needsAttention) || failedAgents > 0 || hasAttentionText(status) || hasAttentionText(closure);
      const lane = deriveLane({status, phase, closure, agents, tasksClosed, tasksTotal, attention, source});
      return {
        key: 'run:' + (runRef || appRef || source + ':' + JSON.stringify(item).slice(0, 80)),
        kind: mode.includes('goal') ? 'goal' : 'legacy',
        lane,
        title: runRef || appRef || 'run sin ref',
        subtitle: appRef || source,
        runRef,
        appRef,
        status,
        phase,
        closure,
        percent: clamp(pct, 0, 100),
        agents,
        tasksClosed,
        tasksTotal,
        source,
        attention,
        raw: compactRaw(item)
      };
    }

    function agentCard(agent) {
      const agentRef = text(agent.agent_ref || agent.agentRef || agent.ref || agent.id);
      const status = normalizeToken(agent.status || agent.progress_status || agent.progressStatus || (agent.in_flight ? 'running' : 'idle'));
      const attention = Boolean(agent.needs_attention || agent.needsAttention) || hasAttentionText(status) || number(agent.no_progress_ticks ?? agent.noProgressTicks ?? 0) > 0;
      return {
        key: 'agent:' + agentRef,
        kind: 'agent',
        lane: attention ? 'attention' : (agent.in_flight || status.includes('running') ? 'running' : 'waiting'),
        title: agentRef || 'agente sin ref',
        subtitle: text(agent.task_ref || agent.taskRef || agent.run_ref || agent.runRef || 'agente'),
        runRef: text(agent.run_ref || agent.runRef),
        appRef: '',
        status,
        phase: text(agent.progress_status || agent.progressStatus || agent.runtime_kind || agent.runtimeKind),
        closure: '',
        percent: 0,
        agents: 1,
        tasksClosed: 0,
        tasksTotal: 0,
        source: 'agent_progress',
        attention,
        raw: compactRaw(agent)
      };
    }

    function issueCard(issue) {
      const runRef = text(issue.run_ref || issue.runRef);
      const code = normalizeToken(issue.code || issue.reason || 'incidencia');
      const action = normalizeToken(issue.recommended_action || issue.recommendedAction || '');
      const reclaim = code.includes('stale') || code.includes('reclaim') || code.includes('no_process') || action.includes('reclaim');
      return {
        key: 'issue:' + (runRef || code) + ':' + action,
        kind: 'issue',
        lane: 'attention',
        title: runRef || code,
        subtitle: code,
        runRef,
        appRef: text(issue.app_ref || issue.appRef),
        status: normalizeToken(issue.status || issue.severity || 'attention'),
        phase: action,
        closure: '',
        percent: 0,
        agents: 0,
        tasksClosed: 0,
        tasksTotal: 0,
        source: reclaim ? 'reclaimable' : 'stale_running',
        attention: true,
        reclaim,
        raw: compactRaw(issue)
      };
    }

    function actionCard(action) {
      const runRef = text(action.run_ref || action.runRef);
      return {
        key: 'action:' + (runRef || action.action || action.reason || JSON.stringify(action).slice(0, 80)),
        kind: 'issue',
        lane: 'attention',
        title: runRef || normalizeToken(action.action || 'accion segura publicada'),
        subtitle: normalizeToken(action.action || action.reason || 'safe_action'),
        runRef,
        appRef: '',
        status: 'safe_action',
        phase: normalizeToken(action.endpoint || action.reason || ''),
        closure: '',
        percent: 0,
        agents: 0,
        tasksClosed: 0,
        tasksTotal: 0,
        source: 'safe_actions_read_only',
        attention: true,
        raw: compactRaw(action)
      };
    }

    function deriveLane(card) {
      const all = [card.status, card.phase, card.closure].join(' ');
      if (card.attention || hasAttentionText(all)) return 'attention';
      if (isDoneText(all)) return 'done';
      if (card.closure || all.includes('close') || all.includes('cierre') || all.includes('validation') || all.includes('validacion')) return 'closure';
      if (all.includes('wait') || all.includes('esper') || all.includes('review') || all.includes('pending_delivery')) return 'waiting';
      if (card.agents > 0 || all.includes('running') || all.includes('progress') || all.includes('started') || all.includes('active')) return 'running';
      if (all.includes('queued') || all.includes('ready') || all.includes('pending') || all.includes('planned') || card.source.includes('queue')) return 'ready';
      return 'waiting';
    }

    function render() {
      const filtered = state.cards.filter(matchesFilters);
      lanes.forEach(function(lane) {
        const items = filtered.filter(function(card) { return card.lane === lane; });
        document.getElementById('count-' + lane).textContent = String(items.length);
        document.getElementById('lane-' + lane).innerHTML = items.length ? items.map(cardHTML).join('') : '<div class="empty">Sin tarjetas</div>';
      });
      renderKPIs(filtered);
      renderDiagnostics();
      document.getElementById('source-summary').textContent = JSON.stringify({
        autoprogramming_status: sourceShape(state.auto),
        queue_global_status: sourceShape(state.globalStatus),
        cards: state.cards.length,
        filtered_cards: filtered.length,
        read_only_projection: true,
        no_new_source_of_truth: true
      }, null, 2);
    }

    function renderKPIs(cards) {
      const byLane = Object.fromEntries(lanes.map(function(lane) { return [lane, cards.filter(function(card) { return card.lane === lane; }).length]; }));
      setKPI('ready', byLane.ready, 'ready/queued');
      setKPI('running', byLane.running, cards.filter(function(card) { return card.kind === 'agent' && card.lane === 'running'; }).length + ' agentes');
      setKPI('attention', byLane.attention, cards.filter(function(card) { return card.reclaim; }).length + ' reclaimables');
      setKPI('closure', byLane.closure, 'validacion/cierre');
      const agents = cards.filter(function(card) { return card.kind === 'agent'; }).length;
      setKPI('agents', agents, 'publicados por estado');
    }

    function renderDiagnostics() {
      const diagnostics = []
        .concat(state.diagnostics || [])
        .concat(arrayAt(state.auto, ['diagnostics']).map(function(item) {
          return {code: text(item.code || 'diagnostico'), message: text(item.message || item.scope || ''), source: 'autoprogramming_status'};
        }))
        .concat(safeActions(state.auto).map(function(item) {
          return {code: text(item.action || 'safe_action'), message: text(item.reason || item.endpoint || ''), source: 'safe_actions'};
        }));
      const root = document.getElementById('diagnostics');
      if (!diagnostics.length) {
        root.innerHTML = '<div class="empty">Sin diagnosticos ni acciones publicadas</div>';
        return;
      }
      root.innerHTML = diagnostics.slice(0, 30).map(function(item) {
        return '<div class="card"><div class="row"><span class="tag info">' + esc(item.source || 'fuente') + '</span><span class="tag">' + esc(item.code || '-') + '</span></div><div class="card-meta">' + esc(item.message || '-') + '</div></div>';
      }).join('');
    }

    function cardHTML(card) {
      const classes = ['card'];
      if (card.attention) classes.push('attention');
      if (card.reclaim) classes.push('reclaim');
      if (card.lane === 'done') classes.push('done');
      const tasks = card.tasksTotal ? '<span class="tag">' + esc(card.tasksClosed + '/' + card.tasksTotal + ' tareas') + '</span>' : '';
      const agents = card.agents ? '<span class="tag">' + esc(card.agents + ' agentes') + '</span>' : '';
      const closure = card.closure ? '<span class="tag">' + esc(card.closure) + '</span>' : '';
      return '<article class="' + classes.join(' ') + '" data-run-ref="' + esc(card.runRef) + '">' +
        '<div class="card-title">' + esc(card.title) + '</div>' +
        '<div class="card-meta"><span>' + esc(card.subtitle || card.source) + '</span><span class="mono">' + esc(card.phase || card.source) + '</span></div>' +
        '<div class="row"><span class="tag ' + statusClass(card.status, card.attention) + '">' + esc(card.status || card.lane) + '</span><span class="tag">' + esc(card.kind) + '</span>' + closure + agents + tasks + '</div>' +
        '<div class="bar" aria-label="progreso"><span class="' + statusClass(card.status, card.attention) + '" style="width:' + esc(String(card.percent)) + '%"></span></div>' +
        '<details><summary>datos</summary><pre>' + esc(JSON.stringify(card.raw, null, 2)) + '</pre></details>' +
        '</article>';
    }

    function addCard(cards, seen, card) {
      if (!card || !card.key) return;
      const key = card.key + ':' + card.lane;
      if (seen.has(key)) return;
      seen.add(key);
      cards.push(card);
    }

    function matchesFilters(card) {
      const q = document.getElementById('filter-text').value.trim().toLowerCase();
      const app = document.getElementById('filter-app').value.trim().toLowerCase();
      const lane = document.getElementById('filter-state').value;
      const kind = document.getElementById('filter-kind').value;
      const haystack = JSON.stringify([card.title, card.subtitle, card.runRef, card.appRef, card.status, card.phase, card.closure, card.source, card.raw]).toLowerCase();
      if (q && !haystack.includes(q)) return false;
      if (app && !(card.appRef || '').toLowerCase().includes(app)) return false;
      if (lane && card.lane !== lane) return false;
      if (kind && card.kind !== kind) return false;
      return true;
    }

    function queueGlobalItems(value) {
      return []
        .concat(arrayAt(value, ['items']))
        .concat(arrayAt(value, ['runs']))
        .concat(arrayAt(value, ['ranked']))
        .concat(arrayAt(value, ['queue', 'ranked']))
        .concat(arrayAt(value, ['global_status', 'items']));
    }

    function safeActions(auto) {
      if (auto.operator && Array.isArray(auto.operator.safe_actions)) return auto.operator.safe_actions;
      if (Array.isArray(auto.safe_actions)) return auto.safe_actions;
      return [];
    }

    function arrayAt(root, path) {
      let value = root;
      for (const key of path) {
        if (!value || typeof value !== 'object') return [];
        value = value[key];
      }
      return Array.isArray(value) ? value : [];
    }

    function setKPI(id, value, hint) {
      document.getElementById('kpi-' + id).textContent = String(value);
      document.getElementById('kpi-' + id + '-hint').textContent = hint || '-';
    }

    function setLive(kind, textValue) {
      const dot = document.getElementById('live-dot');
      dot.className = 'dot ' + kind;
      document.getElementById('live-text').textContent = textValue;
    }

    function sourceShape(value) {
      return {
        estado: value && value.estado || '',
        has_queue: Boolean(value && value.queue),
        runs: arrayAt(value || {}, ['runs']).length,
        agents: arrayAt(value || {}, ['agents']).length,
        stale_running: arrayAt(value || {}, ['stale_running']).length,
        diagnostics: arrayAt(value || {}, ['diagnostics']).length
      };
    }

    function diagnosticFromError(code, error) {
      const payload = error && error.payload || {};
      const issue = payload.errores_publicos && payload.errores_publicos[0] || {};
      return {code: issue.code || code, message: issue.message || (error && error.message) || code, source: 'fetch'};
    }

    function statusClass(status, attention) {
      const value = normalizeToken(status);
      if (attention || hasAttentionText(value)) return 'bad';
      if (value.includes('wait') || value.includes('pending') || value.includes('queued')) return 'warn';
      if (isDoneText(value)) return 'ok';
      if (value.includes('run') || value.includes('active') || value.includes('goal')) return 'info';
      return '';
    }

    function laneWeight(lane) {
      return {ready: 1, running: 2, waiting: 3, attention: 4, closure: 5, done: 6}[lane] || 9;
    }

    function priorityWeight(card) {
      return (card.attention ? 1000 : 0) + (card.reclaim ? 500 : 0) + number(card.raw.priority_score || card.raw.priorityScore || 0);
    }

    function hasAttentionText(value) {
      const textValue = normalizeToken(value);
      return ['blocked', 'bloque', 'failed', 'error', 'lost', 'stale', 'conflict', 'timeout', 'attention', 'quota', 'invalid'].some(function(token) {
        return textValue.includes(token);
      });
    }

    function isDoneText(value) {
      const textValue = normalizeToken(value);
      return ['done', 'complete', 'completed', 'cerrada', 'closed', 'accepted', 'terminada'].some(function(token) {
        return textValue.includes(token);
      });
    }

    function compactRaw(value) {
      const raw = {};
      Object.keys(value || {}).slice(0, 28).forEach(function(key) {
        const current = value[key];
        if (current == null) return;
        if (typeof current === 'object') {
          if (Array.isArray(current)) raw[key] = current.slice(0, 8);
          else raw[key] = '[object]';
        } else {
          raw[key] = current;
        }
      });
      return raw;
    }

    function number(value) {
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : 0;
    }

    function clamp(value, min, max) {
      return Math.max(min, Math.min(max, value));
    }

    function text(value) {
      return String(value == null ? '' : value).trim();
    }

    function normalizeToken(value) {
      return text(value).toLowerCase().replaceAll('_', '-');
    }

    function esc(value) {
      return text(value).replace(/[&<>"']/g, function(ch) {
        return {'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[ch];
      });
    }

    refreshAll();
  </script>
</body>
</html>
`
}
