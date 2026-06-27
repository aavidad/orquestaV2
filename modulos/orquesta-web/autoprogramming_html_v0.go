package orquestaweb

func autoprogrammingHTMLV0() string {
	return `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Orquesta Autoprogramacion</title>
  <style>
    :root { color-scheme: dark; --bg:#101316; --panel:#181d22; --line:#34404a; --text:#eef4f8; --muted:#aab7c3; --accent:#7cd4fd; --good:#32d583; --warn:#fdb022; --bad:#f97066; }
    * { box-sizing: border-box; }
    body { margin:0; min-height:100%; padding:22px; background: radial-gradient(circle at top left, rgba(124,212,253,.17), transparent 32rem), var(--bg); color:var(--text); font:14px/1.5 ui-sans-serif, system-ui, sans-serif; }
    .shell { max-width:1260px; margin:0 auto; display:grid; gap:16px; }
    header { display:flex; justify-content:space-between; gap:14px; align-items:flex-start; }
    h1 { margin:0; font-size:clamp(28px,4vw,52px); letter-spacing:-.04em; }
    h2 { margin:0 0 12px; font-size:18px; }
    a { color:var(--accent); }
    .nav { display:flex; gap:8px; flex-wrap:wrap; }
    .nav a, button { border:1px solid var(--line); background:#202832; color:var(--text); border-radius:8px; padding:8px 10px; text-decoration:none; cursor:pointer; }
    button.primary { background:var(--accent); color:#071015; border-color:var(--accent); font-weight:800; }
    .grid { display:grid; grid-template-columns:minmax(0,1fr) minmax(340px,.75fr); gap:16px; align-items:start; }
    .panel { border:1px solid var(--line); background:rgba(24,29,34,.92); border-radius:14px; padding:16px; }
    form { display:grid; gap:12px; }
    label { display:grid; gap:5px; color:var(--muted); }
    input, textarea, select { width:100%; border:1px solid var(--line); background:#0e1216; color:var(--text); border-radius:8px; padding:9px; font:inherit; }
    textarea { min-height:78px; resize:vertical; }
    .row { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; }
    .hint { color:var(--muted); }
    .result { display:grid; gap:10px; }
    .toggle { display:flex; align-items:center; gap:8px; width:max-content; max-width:100%; color:var(--text); }
    .toggle input { width:auto; }
    .pill { display:inline-flex; width:max-content; border-radius:999px; padding:3px 9px; background:#2a3540; color:var(--muted); }
    .pill.ok { background:rgba(50,213,131,.14); color:var(--good); }
    .pill.warn { background:rgba(253,176,34,.14); color:var(--warn); }
    .pill.bad { background:rgba(249,112,102,.14); color:var(--bad); }
    .status-grid { display:grid; grid-template-columns:max-content minmax(0,1fr); gap:6px 10px; margin:0; }
    .status-grid dt { color:var(--muted); }
    .status-grid dd { margin:0; min-width:0; word-break:break-word; }
    .health-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:8px; }
    .health-item { border:1px solid #3d4b56; border-radius:8px; padding:8px; background:#111820; min-width:0; }
    .health-item strong { display:block; font-size:18px; }
    .health-item span { display:block; color:var(--muted); font-size:12px; }
    .action-list { display:grid; gap:8px; }
    .action-item { display:grid; gap:4px; padding:9px; border:1px solid #3d4b56; border-left:4px solid var(--warn); border-radius:8px; background:#111820; }
    .action-item.blocked { border-left-color:var(--bad); }
    .action-item.info { border-left-color:var(--accent); }
    .action-head { display:flex; justify-content:space-between; gap:10px; align-items:center; }
    .action-meta { display:flex; gap:6px; flex-wrap:wrap; color:var(--muted); font-size:12px; }
    .action-payload { display:block; overflow:auto; padding:6px 8px; border:1px solid #2b3742; border-radius:6px; background:#090d11; color:var(--muted); font-size:12px; }
    .goal-list { display:grid; gap:6px; }
    .goal-item { display:flex; justify-content:space-between; align-items:center; gap:8px; width:100%; text-align:left; }
    .goal-item.active { border-color:var(--accent); background:#10212b; }
    .goal-item code { color:var(--muted); }
    pre { margin:0; max-height:420px; overflow:auto; padding:12px; border-radius:10px; background:#090d11; border:1px solid #24303a; white-space:pre-wrap; word-break:break-word; }
    .actions { display:flex; gap:8px; flex-wrap:wrap; }
    button:disabled { opacity:.55; cursor:not-allowed; }
    @media (max-width: 860px) { body{padding:14px} header,.grid,.row,.health-grid{grid-template-columns:1fr; display:grid} .nav{justify-content:flex-start} }
  </style>
</head>
<body>
  <main class="shell">
    <header>
      <div>
        <h1>Autoprogramacion</h1>
        <p class="hint">Prepara trabajo programable, consulta estado y avanza una run sin salir de la web.</p>
      </div>
      <nav class="nav" aria-label="navegacion">
        <a href="/">Inicio</a><a href="/ops">Ops</a><a href="/run-queue">Cola</a><a href="/run-control">Control</a>
      </nav>
    </header>
    <section class="grid">
      <div class="panel">
        <h2>Preparar run</h2>
        <form id="prepare-form">
          <div class="row">
            <label>Proyecto <input name="project_ref" value="project-ref-orquesta-web" required></label>
            <label>Worktree ref <input name="worktree_ref" value="worktree-ref-orquesta-web" required></label>
          </div>
          <div class="row">
            <label>Branch ref <input name="branch_ref" value="branch-ref-autoprogramming-web" required></label>
            <label>Prioridad <input name="priority_score" type="number" value="80" min="0" max="100"></label>
          </div>
          <div class="row">
            <label>Task ref <input name="task_ref" value="task-ref-web-autoprogramming-001" required></label>
            <label>Area <input name="area" value="web" required></label>
          </div>
          <label>Titulo <input name="title" value="Mejorar superficie web de operador"></label>
          <label>Objetivo <textarea name="objective">Implementar un corte pequeno, probado y usable desde la web de Orquesta.</textarea></label>
          <label>Write-set <textarea name="write_set">modulos/orquesta-web</textarea></label>
          <label>Tests requeridos <textarea name="required_tests">go test -count=1 ./modulos/orquesta-web</textarea></label>
          <label>Criterios de aceptacion <textarea name="acceptance">La web conserva refs opacas
No accede a stores ni runtime
El cambio queda cubierto por tests</textarea></label>
          <label class="toggle" title="Usa Codex Goal como loop interno y deja Orquesta como preparador, observador y validador.">
            <input name="goal_first" type="checkbox" checked>
            Codex Goal automatico
          </label>
          <div class="actions">
            <button class="primary" type="submit">Preparar run</button>
            <button type="button" id="refresh-status">Consultar estado</button>
            <button type="button" id="supervise-run">Supervisar legacy</button>
          </div>
        </form>
      </div>
      <aside class="panel result" aria-live="polite">
        <h2>Resultado</h2>
        <span id="state-pill" class="pill warn">sin accion</span>
        <div class="hint">Run actual: <code id="run-ref">-</code></div>
        <div class="actions">
          <a id="stats-link" href="/director-stats">Abrir stats</a>
          <a href="/ops">Ver en cockpit</a>
        </div>
        <section id="goal-panel" class="action-list" hidden>
          <h2>Goal</h2>
          <div id="goal-list" class="goal-list" hidden></div>
          <dl class="status-grid">
            <dt>Goal ref</dt><dd><code id="goal-ref">-</code></dd>
            <dt>Externo</dt><dd><code id="external-goal-ref">-</code></dd>
            <dt>Goal</dt><dd><span id="goal-status" class="pill warn">pendiente</span></dd>
            <dt>Cierre</dt><dd><span id="closure-status" class="pill warn">sin validar</span></dd>
          </dl>
          <div class="actions">
            <button type="button" id="observe-goal">Observar goal</button>
            <button type="button" id="stop-goal-poll">Parar seguimiento</button>
          </div>
          <div id="goal-note" class="hint">Sin goal lanzado.</div>
        </section>
        <section id="queue-health-panel" class="action-list" hidden>
          <h2>Salud de cola</h2>
          <div id="queue-health-grid" class="health-grid"></div>
        </section>
        <section id="stale-running-panel" class="action-list" hidden>
          <h2>Acciones requeridas</h2>
          <div id="stale-running-list" class="action-list"></div>
        </section>
        <section id="safe-actions-panel" class="action-list" hidden>
          <h2>Acciones seguras</h2>
          <div id="safe-actions-list" class="action-list"></div>
        </section>
        <pre id="output">{}</pre>
      </aside>
    </section>
  </main>
  <script>
    const output = document.getElementById('output');
    const runRefNode = document.getElementById('run-ref');
    const pill = document.getElementById('state-pill');
    const statsLink = document.getElementById('stats-link');
    const stalePanel = document.getElementById('stale-running-panel');
    const staleList = document.getElementById('stale-running-list');
    const safeActionsPanel = document.getElementById('safe-actions-panel');
    const safeActionsList = document.getElementById('safe-actions-list');
    const queueHealthPanel = document.getElementById('queue-health-panel');
    const queueHealthGrid = document.getElementById('queue-health-grid');
    const superviseRunButton = document.getElementById('supervise-run');
    const goalPanel = document.getElementById('goal-panel');
    const goalRefNode = document.getElementById('goal-ref');
    const externalGoalRefNode = document.getElementById('external-goal-ref');
    const goalStatusNode = document.getElementById('goal-status');
    const closureStatusNode = document.getElementById('closure-status');
    const goalNote = document.getElementById('goal-note');
    const goalListNode = document.getElementById('goal-list');
    let currentRunRef = '';
    let currentGoalRef = '';
    let currentExternalGoalRef = '';
    let currentGoals = [];
    let currentSafeActions = [];
    let currentGoalIndex = 0;
    let goalPollTimer = 0;
    let goalPollCount = 0;

    function lines(value) {
      return String(value || '').split(/\n|,/).map(v => v.trim()).filter(Boolean);
    }
    function requestId(prefix) {
      return prefix + '-' + Date.now().toString(36);
    }
    function setResult(kind, value) {
      output.textContent = JSON.stringify(value || {}, null, 2);
      const estado = String((value || {}).estado || kind || 'unknown');
      pill.textContent = estado;
      pill.className = 'pill ' + (estado === 'ok' ? 'ok' : (estado === 'error' ? 'bad' : 'warn'));
      const runRef = resultRunRef(value);
      if (runRef) {
        currentRunRef = runRef;
        runRefNode.textContent = currentRunRef;
        statsLink.href = '/director-stats?include_agent_progress=true&run_ref=' + encodeURIComponent(currentRunRef);
      }
      renderGoal(value || {});
      renderQueueHealth((value || {}).queue_health || null);
      renderStaleRunning((value || {}).stale_running || []);
      renderSafeActions((value || {}).safe_actions || (((value || {}).operator || {}).safe_actions) || []);
      updateSuperviseState();
    }
    function resultRunRef(value) {
      const goal = selectedGoal(value);
      const spec = (((value || {}).goal_specs || [])[0]) || {};
      return String(goal.run_ref || (value || {}).run_ref || spec.run_ref || '').trim();
    }
    function goalInfo(value) {
      const goal = selectedGoal(value);
      const spec = (((value || {}).goal_specs || [])[0]) || {};
      return {
        run_ref: resultRunRef(value),
        goal_ref: String(goal.goal_ref || (value || {}).goal_ref || spec.goal_ref || '').trim(),
        external_goal_ref: String(goal.external_goal_ref || (value || {}).external_goal_ref || '').trim(),
        goal_status: String(goal.goal_status || (value || {}).goal_status || spec.status || '').trim(),
        closure_status: String((value || {}).closure_status || '').trim(),
        closure_accepted: Boolean((value || {}).closure_accepted),
        closure_needs_rework: Boolean((value || {}).closure_needs_rework)
      };
    }
    function normalizedGoals(value) {
      const sourceGoals = Array.isArray((value || {}).goals) ? (value || {}).goals : [];
      if (sourceGoals.length) {
        return sourceGoals.map((goal, index) => Object.assign({_index:index}, goal || {}));
      }
      const single = (value || {}).goal || {};
      if (single.run_ref || single.goal_ref || single.external_goal_ref || single.goal_status) {
        return [Object.assign({_index:0}, single)];
      }
      const specs = Array.isArray((value || {}).goal_specs) ? (value || {}).goal_specs : [];
      return specs.map((spec, index) => ({
        _index: index,
        run_ref: spec.run_ref || '',
        goal_ref: spec.goal_ref || '',
        goal_status: 'preparado'
      }));
    }
    function selectedGoal(value) {
      const goals = normalizedGoals(value);
      if (!goals.length) return {};
      if (currentGoalIndex < 0 || currentGoalIndex >= goals.length) currentGoalIndex = 0;
      return goals[currentGoalIndex] || goals[0] || {};
    }
    function renderGoal(value) {
      currentGoals = normalizedGoals(value);
      const info = goalInfo(value);
      const hasGoal = Boolean(info.goal_ref || currentGoals.length);
      goalPanel.hidden = !hasGoal;
      if (!hasGoal) return;
      renderGoalList(currentGoals);
      currentRunRef = info.run_ref || currentRunRef;
      currentGoalRef = info.goal_ref || currentGoalRef;
      currentExternalGoalRef = info.external_goal_ref || currentExternalGoalRef;
      goalRefNode.textContent = currentGoalRef || '-';
      externalGoalRefNode.textContent = currentExternalGoalRef || '-';
      setStatusPill(goalStatusNode, info.goal_status || 'preparado');
      setStatusPill(closureStatusNode, info.closure_status || (info.closure_accepted ? 'aceptado' : 'pendiente'));
      if (info.closure_accepted) {
        goalNote.textContent = 'Cierre aceptado por evidencias.';
      } else if (info.closure_needs_rework) {
        goalNote.textContent = 'Cierre bloqueado: requiere rework.';
      } else if (!info.run_ref) {
        goalNote.textContent = 'Goal spec preparado; falta backend Goal opt-in para lanzarlo.';
      } else {
        goalNote.textContent = 'Seguimiento goal-first activo por run_ref.';
      }
    }
    function renderGoalList(goals) {
      goalListNode.textContent = '';
      goalListNode.hidden = !goals || goals.length <= 1;
      (goals || []).forEach((goal, index) => {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'goal-item' + (index === currentGoalIndex ? ' active' : '');
        button.dataset.goalIndex = String(index);
        const label = document.createElement('span');
        label.textContent = 'Goal ' + String(index + 1);
        const ref = document.createElement('code');
        ref.textContent = goal.run_ref || goal.goal_ref || '-';
        const status = document.createElement('span');
        status.className = 'pill warn';
        status.textContent = goal.goal_status || 'preparado';
        button.append(label, ref, status);
        button.addEventListener('click', () => {
          currentGoalIndex = index;
          currentRunRef = String(goal.run_ref || '').trim();
          currentGoalRef = String(goal.goal_ref || '').trim();
          currentExternalGoalRef = String(goal.external_goal_ref || '').trim();
          renderGoal({goals: currentGoals});
          runRefNode.textContent = currentRunRef || '-';
          statsLink.href = currentRunRef
            ? '/director-stats?include_agent_progress=true&run_ref=' + encodeURIComponent(currentRunRef)
            : '/director-stats';
          stopGoalPolling();
        });
        goalListNode.appendChild(button);
      });
    }
    function setStatusPill(node, status) {
      const normalized = String(status || 'pendiente').trim();
      const low = normalized.toLowerCase();
      node.textContent = normalized;
      node.className = 'pill ' + (
        low === 'complete' || low === 'completed' || low === 'accepted' || low === 'aceptado' || low === 'cerrada' || low === 'closed'
          ? 'ok'
          : (low === 'blocked' || low === 'failed' || low === 'invalid' || low === 'rejected' ? 'bad' : 'warn')
      );
    }
    function updateSuperviseState() {
      const goalActive = Boolean(currentGoalRef) || hasGoalFirstSafeActionForCurrentRun();
      const legacyAvailable = Boolean(legacySuperviseSafeActionForCurrentRun());
      superviseRunButton.disabled = goalActive || !legacyAvailable;
      superviseRunButton.title = goalActive
        ? 'Esta run usa Goal; usa Observar goal.'
        : (legacyAvailable
          ? 'Solo para runs legacy de autoprogramacion con accion segura publicada.'
          : 'La API no publica supervision legacy como accion segura.');
    }
    function hasGoalFirstSafeActionForCurrentRun() {
      return goalFirstSafeActions().some(item => {
        const runRef = String(item.run_ref || '').trim();
        return !currentRunRef || !runRef || runRef === currentRunRef;
      });
    }
    function goalFirstSafeActions() {
      return (currentSafeActions || []).filter(item =>
        item &&
        item.action === 'observe_goal' &&
        item.endpoint === '/api/v0/autoprogramming/goal/observe'
      );
    }
    function legacySuperviseSafeActionForCurrentRun() {
      return (currentSafeActions || []).find(item => {
        if (!item || item.endpoint !== '/api/v0/autoprogramming/supervise') return false;
        if (item.action !== 'supervise') return false;
        const runRef = String(item.run_ref || '').trim();
        const scope = String(item.scope || '').trim();
        if (scope === 'queue') return !currentRunRef;
        return !currentRunRef || !runRef || runRef === currentRunRef;
      }) || null;
    }
    function renderStaleRunning(items) {
      staleList.textContent = '';
      stalePanel.hidden = !items || !items.length;
      (items || []).slice(0, 8).forEach(item => {
        const row = document.createElement('article');
        const severity = String(item.severity || 'warning');
        row.className = 'action-item ' + severity;
        const head = document.createElement('div');
        head.className = 'action-head';
        const code = document.createElement('strong');
        code.textContent = item.code || 'estado_accionable';
        const status = document.createElement('span');
        status.className = 'pill ' + (severity === 'blocked' ? 'bad' : (severity === 'info' ? 'ok' : 'warn'));
        status.textContent = item.status || severity;
        head.append(code, status);
        const meta = document.createElement('div');
        meta.className = 'action-meta';
        [item.run_ref, item.app_ref, item.reason].filter(Boolean).forEach(value => {
          const span = document.createElement('span');
          span.textContent = value;
          meta.appendChild(span);
        });
        const action = document.createElement('div');
        action.className = 'hint';
        action.textContent = item.recommended_action || 'inspect_run_ref';
        row.append(head, meta, action);
        staleList.appendChild(row);
      });
    }
    function renderSafeActions(items) {
      currentSafeActions = Array.isArray(items) ? items : [];
      safeActionsList.textContent = '';
      safeActionsPanel.hidden = !items || !items.length;
      (items || []).slice(0, 8).forEach(item => {
        const row = document.createElement('article');
        row.className = 'action-item info';
        const head = document.createElement('div');
        head.className = 'action-head';
        const code = document.createElement('strong');
        code.textContent = item.action || 'accion';
        const status = document.createElement('span');
        status.className = 'pill warn';
        status.textContent = [item.method || 'POST', item.scope || 'run'].filter(Boolean).join(' ');
        head.append(code, status);
        const meta = document.createElement('div');
        meta.className = 'action-meta';
        [item.endpoint, item.run_ref, item.reason].filter(Boolean).forEach(value => {
          const span = document.createElement('span');
          span.textContent = value;
          meta.appendChild(span);
        });
        const payload = document.createElement('code');
        payload.className = 'action-payload';
        payload.textContent = compactActionPayload(item);
        const controls = document.createElement('div');
        controls.className = 'actions';
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = 'Ejecutar';
        button.disabled = !safeActionExecutable(item);
        button.title = button.disabled
          ? 'Accion informativa o sin payload suficiente.'
          : 'Ejecuta la accion segura indicada por la API.';
        button.addEventListener('click', async () => {
          const data = await executeSafeAction(item);
          setResult('safe-action', data);
        });
        controls.appendChild(button);
        row.append(head, meta, payload, controls);
        safeActionsList.appendChild(row);
      });
    }
    function compactActionPayload(item) {
      const payload = Object.assign({}, item && item.payload ? item.payload : {});
      if (item && item.action === 'observe_goal' && item.run_ref && !payload.run_ref) {
        payload.run_ref = item.run_ref;
      }
      if (item && item.endpoint === '/api/v0/autoprogramming/goal/observe') {
        payload.requested_by = payload.requested_by || 'orquesta-web';
      }
      return Object.keys(payload).length ? JSON.stringify(payload) : '{}';
    }
    function safeActionExecutable(item) {
      if (!item || item.method !== 'POST' || !item.endpoint) return false;
      if (item.action === 'observe_goal') return Boolean(item.run_ref || currentRunRef);
      return Boolean(item.payload && Object.keys(item.payload).length);
    }
    async function executeSafeAction(item) {
      const id = requestId('web-safe-action');
      const payload = Object.assign({}, item.payload || {});
      if (item.action === 'observe_goal') {
        payload.run_ref = payload.run_ref || item.run_ref || currentRunRef;
        payload.requested_by = payload.requested_by || 'orquesta-web';
      }
      payload.request_id = payload.request_id || id;
      payload.correlation_id = payload.correlation_id || id;
      return postJSON(item.endpoint, payload);
    }
    function renderQueueHealth(health) {
      queueHealthGrid.textContent = '';
      queueHealthPanel.hidden = !health;
      if (!health) return;
      [
        ['Agentes vivos', health.agents_live || 0],
        ['Runs vivas', health.running_live || 0],
        ['Sin stats recientes', health.running_without_recent_stats || 0],
        ['Stale sin proceso', health.running_stale_no_process || health.running_stale || 0],
        ['En cola', health.queued || 0],
        ['Bloqueadas', health.blocked || 0],
        ['Completadas', health.completed || 0],
        ['Fallidas', health.failed || 0]
      ].forEach(([label, value]) => {
        const item = document.createElement('div');
        item.className = 'health-item';
        const number = document.createElement('strong');
        number.textContent = String(value);
        const text = document.createElement('span');
        text.textContent = label;
        item.append(number, text);
        queueHealthGrid.appendChild(item);
      });
    }
    async function postJSON(url, payload) {
      const response = await fetch(url, {
        method: 'POST',
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({estado:'error', errores_publicos:[{code:'respuesta_no_json'}]}));
      if (!response.ok && !data.estado) data.estado = 'error';
      return data;
    }
    function goalContextRefs(enabled) {
      if (!enabled) return [];
      return [
        'goal_migration:goal-first',
        'goal_capability:starter',
        'goal_capability:observer',
        'goal_capability:closure-validator'
      ];
    }
    function preparePayload(form) {
      const id = requestId('web-autoprogramming');
      const taskContextRefs = goalContextRefs(form.goal_first && form.goal_first.checked);
      return {
        request_id: id,
        correlation_id: id,
        requested_by: 'orquesta-web',
        priority_score: Number(form.priority_score.value || 0),
        max_bursts: 4,
        max_steps_per_burst: 8,
        max_dispatches_per_wait: 8,
        autoprogramming_request: {
          request_ref: id,
          project_ref: form.project_ref.value.trim(),
          worktree_ref: form.worktree_ref.value.trim(),
          worktree_isolated: true,
          branch_ref: form.branch_ref.value.trim(),
          write_set: lines(form.write_set.value),
          required_tests: lines(form.required_tests.value),
          tasks: [{
            task_ref: form.task_ref.value.trim(),
            area: form.area.value.trim(),
            title: form.title.value.trim(),
            objective: form.objective.value.trim(),
            acceptance_criteria: lines(form.acceptance.value),
            required_tests: lines(form.required_tests.value),
            context_refs: taskContextRefs
          }]
        }
      };
    }
    async function refreshStatus() {
      const id = requestId('web-autoprogramming-status');
      const data = await postJSON('/api/v0/autoprogramming/status', {
        request_id: id,
        correlation_id: id,
        run_ref: currentRunRef,
        queue_ref: 'global',
        queue_limit: 20,
        include_agent_progress: true,
        include_agent_usage: true
      });
      setResult('status', data);
    }
    async function observeGoal(startPolling) {
      if (!currentRunRef) {
        goalPanel.hidden = false;
        goalNote.textContent = 'No hay run_ref de Goal para observar.';
        return;
      }
      const id = requestId('web-autoprogramming-goal');
      const data = await postJSON('/api/v0/autoprogramming/goal/observe', {
        request_id: id,
        correlation_id: id,
        run_ref: currentRunRef,
        requested_by: 'orquesta-web'
      });
      setResult('goal', data);
      if (goalTerminal(data)) {
        stopGoalPolling();
        return;
      }
      if (startPolling) startGoalPolling();
    }
    function startGoalPolling() {
      stopGoalPolling();
      goalPollCount = 0;
      goalPollTimer = window.setInterval(async () => {
        goalPollCount += 1;
        if (goalPollCount > 60 || !currentRunRef || !currentGoalRef) {
          stopGoalPolling();
          return;
        }
        await observeGoal(false);
      }, 5000);
    }
    function stopGoalPolling() {
      if (goalPollTimer) window.clearInterval(goalPollTimer);
      goalPollTimer = 0;
    }
    function goalTerminal(value) {
      const info = goalInfo(value || {});
      const goal = String(info.goal_status || '').toLowerCase();
      const closure = String(info.closure_status || '').toLowerCase();
      const run = String((value || {}).run_status || '').toLowerCase();
      return info.closure_accepted ||
        ['complete','completed','blocked','failed','invalid','canceled','cancelled'].includes(goal) ||
        ['accepted','blocked','failed','rejected'].includes(closure) ||
        ['closed','cerrada','blocked','failed'].includes(run);
    }
    document.getElementById('prepare-form').addEventListener('submit', async function(event) {
      event.preventDefault();
      const data = await postJSON('/api/v0/autoprogramming/prepare-run', preparePayload(event.target));
      setResult('prepare', data);
      if (currentRunRef && currentGoalRef) await observeGoal(true);
    });
    document.getElementById('refresh-status').addEventListener('click', refreshStatus);
    document.getElementById('observe-goal').addEventListener('click', function() {
      observeGoal(true);
    });
    document.getElementById('stop-goal-poll').addEventListener('click', stopGoalPolling);
    document.getElementById('supervise-run').addEventListener('click', async function() {
      if (currentGoalRef || hasGoalFirstSafeActionForCurrentRun()) {
        goalPanel.hidden = false;
        goalNote.textContent = 'Esta run publica observe_goal; no se usa supervision legacy.';
        updateSuperviseState();
        return;
      }
      const action = legacySuperviseSafeActionForCurrentRun();
      if (!action) {
        goalPanel.hidden = false;
        goalNote.textContent = 'La API no publica supervision legacy como accion segura.';
        updateSuperviseState();
        return;
      }
      const data = await executeSafeAction(action);
      setResult('supervise', data);
    });
  </script>
</body>
</html>
`
}
