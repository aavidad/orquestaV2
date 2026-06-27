package orquestaweb

const opsDashboardHTMLChunk4V0 = `          '</div></div>';
      }).join('') : '<div class="sub">Sin agentes observados para este run.</div>';
      byId('selected-detail').innerHTML =
        '<div class="task-title">' + esc(runTitle(run)) + '</div>' +
        '<div class="task-subtitle mono">' + esc(run.run_ref || '-') + '</div>' +
        runFlowHTML(run, runAgents, tasks) +
        '<div class="kv"><div class="k">Estado del Director</div><div>' +
          '<div class="sub">fase <span class="mono">' + esc(run.current_phase || '-') + '</span> · cierre <span class="mono">' + esc(run.closure_status || '-') + '</span> · bloqueo <span class="mono">' + esc(run.blocked ? 'sí' : 'no') + '</span></div>' +
          '<div class="sub">tareas ' + esc(String(run.tasks_closed || 0)) + '/' + esc(String(run.tasks_total || tasks.length || 0)) + ' · agentes vuelo ' + esc(String(run.agents_in_flight || 0)) + ' · avanzan ' + esc(String(run.progressing_agents || 0)) + ' · parados ' + esc(String(run.stalled_agents || 0)) + '</div>' +
          '<div class="sub">frescura <span class="mono">' + esc(run.progress_source || '-') + '</span> · stats <span class="mono">' + esc(run.stats_fetch_status || '-') + '</span> · motivo <span class="mono">' + esc(run.stats_reason_code || '-') + '</span></div>' +
        '</div></div>' +
        '<div class="kv"><div class="k">ID tarea</div><div><span class="task-id">' + esc(run.task_id || taskIDFromRef(run.run_ref)) + '</span></div></div>' +
        '<div class="kv"><div class="k">Proyecto</div><div class="mono">' + esc(run.app_ref || '-') + '</div></div>' +
        '<div class="kv"><div class="k">Estado</div><div>' + statusPill(run.status || 'unknown') + '</div></div>' +
        '<div class="kv"><div class="k">Validación</div><div>' + statusPill(run.validation || validationState(run)) + '</div></div>' +
        '<div class="kv"><div class="k">Progreso</div><div>' + bar(run.percent_complete || 0, !!run.blocked) + '<span class="sub">' + esc(String(run.percent_complete || 0)) + '% · fase ' + esc(run.current_phase || '-') + '</span></div></div>' +
        (isCompletedSnapshot ? '<div class="kv"><div class="k">Observada</div><div class="mono">' + esc(run.observed_at || '-') + '</div></div>' : '') +
        '<div class="kv"><div class="k">Agentes</div><div>' + esc(String(run.agents_in_flight || 0)) + ' activos · ' + esc(String(run.agents_started || runAgents.length || 0)) + ' observados</div></div>' +
        '<div class="kv"><div class="k">Uso observado</div><div>' + esc(String(usageAgents.length)) + ' agentes · ' + esc(formatTokens(usageTokens)) + '</div></div>' +
        '<div class="controls"><input id="detail-priority" type="number" min="0" max="1000" step="1" value="' + esc(priority) + '" aria-label="Prioridad"><button type="button" onclick="setSelectedPriority()">Prioridad</button></div>' +
        '<div class="button-row">' +
          '<button type="button" onclick="controlSelectedRun(\'pause\')">Pausar</button>' +
          '<button type="button" onclick="controlSelectedRun(\'resume\')">Reanudar</button>' +
          '<button type="button" onclick="reactivateSelectedRun()">Reactivar</button>' +
          '<button type="button" onclick="advanceSelectedRun()">' + esc(selectedRunAdvanceActionLabel(run)) + '</button>' +
          '<button type="button" onclick="controlSelectedRun(\'stop\')">Parar</button>' +
          '<button type="button" onclick="controlSelectedRun(\'cancel\')">Cancelar</button>' +
          '<button type="button" onclick="deactivateSelectedRun()">Quitar de cola</button>' +
        '</div>' +
        (controlMessage ? '<div class="sub">' + esc(controlMessage) + '</div>' : '') +
        '<div class="kv"><div class="k">Subtareas</div><div>' + taskRows + '</div></div>' +
        '<div class="kv"><div class="k">Agentes</div><div>' + agentRows + '</div></div>' +
        runtimeDetailHTML(run);
    }
	    async function setSelectedPriority() {
	      const run = runByRef(selectedRunRef);
	      if (!run) return;
	      const value = Number((byId('detail-priority') || {}).value || 0);
	      controlMessage = 'Actualizando prioridad...';
	      renderSelectedDetail();
	      try {
	        await mutateQueue(run, value, '', 'cambio de prioridad desde panel ops');
	        controlMessage = 'Prioridad actualizada.';
	      } catch (err) {
	        controlMessage = err.message || String(err);
	      }
	      await refreshAll();
	    }
	    async function mutateQueue(run, priority, status, reason) {
	      const payload = {
	        request_id: 'ops-queue-' + Date.now(),
	        action: 'set_priority',
	        queue_ref: 'global',
	        run_ref: run.run_ref,
	        app_ref: run.app_ref,
	        priority_score: Number(priority || 0),
	        requested_by: 'orquesta-ops-panel',
	        reason: reason || 'mutacion de cola desde panel ops',
	        idempotency_key: 'ops-queue-' + (status || 'priority') + '-' + run.run_ref + '-' + Date.now()
	      };
	      if (status) payload.status = status;
	      const result = await fetchJSON('/api/v0/runs/queue/priority', {
	        method: 'POST',
	        headers: {'content-type': 'application/json'},
	        body: JSON.stringify(payload)
	      });
	      if (result.estado !== 'ok') throw new Error('No se pudo actualizar la cola.');
	      return result;
	    }
	    async function setQueueRowPriority(runRef, delta) {
	      const run = runByRef(runRef);
	      if (!run) return;
	      selectedRunRef = run.run_ref || runRef || '';
	      selectedAgentRef = '';
	      const nextPriority = Math.max(0, Number(run.priority_score || 0) + Number(delta || 0));
	      controlMessage = 'Actualizando prioridad de fila...';
	      renderSelectedDetail();
	      try {
	        await mutateQueue(run, nextPriority, '', 'ajuste rapido de prioridad desde fila de cola');
	        controlMessage = 'Prioridad de fila actualizada.';
	      } catch (err) {
	        controlMessage = err.message || String(err);
	      }
	      await refreshAll();
	    }
	    async function controlQueueRow(runRef, action) {
	      const run = runByRef(runRef);
	      if (!run) return;
	      selectedRunRef = run.run_ref || runRef || '';
	      selectedAgentRef = '';
	      controlMessage = 'Enviando ' + action + ' desde fila...';
	      renderSelectedDetail();
	      try {
	        await mutateRunControl(run, action);
	        if (action === 'resume') await mutateQueue(run, run.priority_score || 50, 'ready', 'reanudar desde fila de cola');
	        controlMessage = 'Acción de fila aplicada: ' + action + '.';
	      } catch (err) {
	        controlMessage = err.message || String(err);
	      }
	      await refreshAll();
	    }
	    function allAutoprogrammingSafeActions() {
	      return Array.isArray(lastSnapshot.safeActions) ? lastSnapshot.safeActions : [];
	    }
	    function autoprogrammingSafeActionsForRun(runRef) {
	      const ref = String(runRef || '');
	      if (!ref) return [];
	      return allAutoprogrammingSafeActions().filter(function(action) {
	        return String((action || {}).run_ref || '') === ref;
	      });
	    }
	    function autoprogrammingObserveGoalSafeAction(runRef) {
	      return autoprogrammingSafeActionsForRun(runRef).find(function(action) {
	        return String((action || {}).action || '') === 'observe_goal' &&
	          String((action || {}).endpoint || '') === '/api/v0/autoprogramming/goal/observe';
	      });
	    }
	    function firstAutoprogrammingSafeAction(actionName, scope) {
	      return allAutoprogrammingSafeActions().find(function(action) {
	        return String((action || {}).action || '') === actionName &&
	          (!scope || String((action || {}).scope || '') === scope);
	      });
	    }
	    function safeActionPayload(action, fallbackRunRef) {
	      const payload = Object.assign({}, (action || {}).payload || {});
	      if (!payload.run_ref && ((action || {}).run_ref || fallbackRunRef)) {
	        payload.run_ref = (action || {}).run_ref || fallbackRunRef;
	      }
	      return payload;
	    }
	    function runIsGoalFirst(run) {
	      const goal = (run || {}).goal || {};
	      return Boolean(autoprogrammingObserveGoalSafeAction((run || {}).run_ref)) ||
	        String((run || {}).director_execution_mode || goal.director_execution_mode || '').toLowerCase() === 'goal_first' ||
	        Boolean(goal.available || goal.goal_ref || goal.can_observe);
	    }
	    function selectedRunAdvanceActionLabel(run) {
	      return runIsGoalFirst(run) ? 'Observar goal' : 'Avanzar run';
	    }
	    function queueAdvanceActionLabel(run) {
	      return runIsGoalFirst(run) ? 'G' : '↪';
	    }
	    function queueAdvanceActionTitle(run) {
	      return runIsGoalFirst(run) ? 'Observar goal desde fila' : 'Avanzar run desde fila';
	    }
	    async function advanceQueueRow(runRef) {
	      const run = runByRef(runRef);
	      if (!run) return;
	      if (runIsGoalFirst(run)) return observeGoalOps(run, 'goal desde fila');
	      return superviseQueueRow(runRef);
	    }
	    async function superviseQueueRow(runRef) {
	      const run = runByRef(runRef);
	      if (!run) return;
	      selectedRunRef = run.run_ref || runRef || '';
	      selectedAgentRef = '';
	      controlMessage = 'Avanzando run desde fila...';
	      renderSelectedDetail();
	      try {
	        const result = await superviseOps({run_ref: run.run_ref || runRef}, 'run desde fila');
	        controlMessage = supervisorResultText(result);
	      } catch (err) {
	        controlMessage = err.message || String(err);
	      }
	      await refreshAll();
	    }
	    async function mutateRunControl(run, action) {
	      const result = await fetchJSON('/api/v0/runs/control', {
	        method: 'POST',
	        headers: {'content-type': 'application/json'},
	        body: JSON.stringify({
	          request_id: 'ops-control-' + Date.now(),
	          action: action,
	          run_ref: run.run_ref,
	          app_ref: run.app_ref,
	          requested_by: 'orquesta-ops-panel',
	          reason: 'accion ' + action + ' desde panel ops',
	          idempotency_key: 'ops-control-' + action + '-' + run.run_ref + '-' + Date.now()
	        })
	      });
	      if (result.estado !== 'ok') throw new Error('No se pudo aplicar control ' + action + '.');
	      return result;
	    }
	    function rememberObservedRun(run, status) {
	      completedHistory = completedHistory.filter(function(item) { return item.run_ref !== run.run_ref; });
	      completedHistory.unshift({
	        run_ref: run.run_ref,
	        app_ref: run.app_ref,
	        task_id: run.task_id || taskIDFromRef(run.run_ref),
	        title: runTitle(run),
	        status: status,
	        percent_complete: run.percent_complete || 0,
	        observed_at: new Date().toLocaleString()
	      });
	      completedHistory = completedHistory.slice(0, 200);
	      saveCompletedHistory();
	    }
	    async function deactivateSelectedRun() {
	      const run = runByRef(selectedRunRef);
	      if (!run) return;
	      if (!confirm('Quitar esta tarea de la cola? No borra registros ni artefactos.')) return;
	      controlMessage = 'Quitando de cola...';
	      renderSelectedDetail();
	      try {
	        await mutateQueue(run, 0, 'canceled', 'quitar de cola desde panel ops; no borra registros ni artefactos');
	        await mutateRunControl(run, 'cancel');
	        rememberObservedRun(run, 'canceled');
	        controlMessage = 'Tarea retirada de cola.';
	      } catch (err) {
	        controlMessage = err.message || String(err);
	      }
	      await refreshAll();
	    }
	    async function reactivateSelectedRun() {
	      const run = runByRef(selectedRunRef);
	      if (!run) return;
	      const value = Number((byId('detail-priority') || {}).value || run.priority_score || 50);
	      controlMessage = 'Reactivando...';
	      renderSelectedDetail();
	      try {
	        await mutateQueue(run, value, 'ready', 'reactivacion desde panel ops');
	        await mutateRunControl(run, 'resume');
	        completedHistory = completedHistory.filter(function(item) { return item.run_ref !== run.run_ref; });
	        saveCompletedHistory();
	        controlMessage = 'Tarea reactivada.';
	      } catch (err) {
	        controlMessage = err.message || String(err);
	      }
	      await refreshAll();
	    }
	    async function controlSelectedRun(action) {
	      const run = runByRef(selectedRunRef);
	      if (!run) return;
	      if (action === 'cancel' && !confirm('Quitar/cancelar esta tarea de la cola?')) return;
      controlMessage = 'Enviando ' + action + '...';
      renderSelectedDetail();
      try {
	        const result = await mutateRunControl(run, action);
	        controlMessage = result.estado === 'ok' ? ('Acción aplicada: ' + action + ' -> ' + (result.status || '-')) : 'No se pudo aplicar acción.';
	        if (action === 'cancel') {
	          await mutateQueue(run, 0, 'canceled', 'cancelacion desde panel ops');
	          rememberObservedRun(run, 'canceled');
	        }
	        if (action === 'stop') {
	          await mutateQueue(run, run.priority_score || 0, 'stopped', 'parada desde panel ops');
	          rememberObservedRun(run, 'stopped');
	        }
	        if (action === 'resume') {
	          await mutateQueue(run, run.priority_score || 50, 'ready', 'reanudar desde panel ops');
	          completedHistory = completedHistory.filter(function(item) { return item.run_ref !== run.run_ref; });
	          saveCompletedHistory();
	        }
	      } catch (err) {
	        controlMessage = err.message || String(err);
      }
      await refreshAll();
    }
    async function superviseGlobalWave() {
      const observeAction = firstAutoprogrammingSafeAction('observe_goal', 'run');
      if (observeAction) {
        supervisorMessage = 'Observando goal...';
        text('director-supervisor-message', supervisorMessage);
        try {
          supervisorMessage = await executeAutoprogrammingSafeActionOps(observeAction, 'goal desde cola');
        } catch (err) {
          supervisorMessage = err.message || String(err);
        }
        text('director-supervisor-message', supervisorMessage);
        await refreshAll();
        return;
      }
      const superviseAction = firstAutoprogrammingSafeAction('supervise', 'queue');
      if (allAutoprogrammingSafeActions().length && !superviseAction) {
        supervisorMessage = 'Sin accion segura de cola; revisa la accion por run publicada.';
        text('director-supervisor-message', supervisorMessage);
        await refreshAll();
        return;
      }
      supervisorMessage = 'Lanzando ola...';
      text('director-supervisor-message', supervisorMessage);
      try {
        if (superviseAction) {
          supervisorMessage = await executeAutoprogrammingSafeActionOps(superviseAction, 'ola global');
        } else {
          const result = await superviseOps({queue_ref: 'global'}, 'ola global');
          supervisorMessage = supervisorResultText(result);
        }
      } catch (err) {
        supervisorMessage = err.message || String(err);
      }
      text('director-supervisor-message', supervisorMessage);
      await refreshAll();
    }
    async function advanceSelectedRun() {
      const run = runByRef(selectedRunRef);
      if (!run) return;
      if (runIsGoalFirst(run)) return observeGoalOps(run, 'goal seleccionado');
      return superviseSelectedRun();
    }
    async function superviseSelectedRun() {
      const run = runByRef(selectedRunRef);
      if (!run) return;
      controlMessage = 'Avanzando run...';
      renderSelectedDetail();
      try {
        const result = await superviseOps({run_ref: run.run_ref}, 'run seleccionado');
        controlMessage = supervisorResultText(result);
      } catch (err) {
        controlMessage = err.message || String(err);
      }
      await refreshAll();
    }
    async function observeGoalOps(run, label) {
      if (!run || !run.run_ref) return;
      selectedRunRef = run.run_ref || selectedRunRef || '';
      selectedAgentRef = '';
      controlMessage = 'Observando Goal...';
      renderSelectedDetail();
      const now = Date.now();
      try {
        const safeAction = autoprogrammingObserveGoalSafeAction(run.run_ref);
        const endpoint = safeAction ? safeAction.endpoint : '/api/v0/apps/director/goal/observe';
        const payload = safeActionPayload(safeAction, run.run_ref);
        payload.request_id = payload.request_id || 'ops-observe-goal-' + now;
        payload.correlation_id = payload.correlation_id || 'corr-ops-observe-goal-' + now;
        payload.run_ref = payload.run_ref || run.run_ref;
        payload.requested_by = payload.requested_by || 'orquesta-ops-panel';
        payload.idempotency_key = payload.idempotency_key || 'ops-observe-goal-' + run.run_ref + '-' + now;
        const result = await fetchJSON(endpoint, {
          method: 'POST',
          headers: {'content-type': 'application/json'},
          body: JSON.stringify(payload)
        });
        controlMessage = goalOpsResultText(result, label);
      } catch (err) {
        controlMessage = err.message || String(err);
      }
      await refreshAll();
    }
    async function superviseOps(scope, label) {
      const now = Date.now();
      const payload = {
        request_id: 'ops-supervise-' + now,
        correlation_id: 'corr-ops-supervise-' + now,
        queue_ref: (scope || {}).queue_ref || '',
        run_ref: (scope || {}).run_ref || '',
        continue_message: 'supervision desde panel ops: ' + (label || 'tick'),
        max_ticks: 1,
        max_runs_per_tick: 1,
        max_executions: 1,
        max_bursts: 1,
        max_steps_per_burst: 1,
        max_dispatches_per_wait: 1,
        max_commands: 1,
        max_outbox_per_cycle: 1,
        max_decision_cycles: 1,
        max_external_waits: 1,
        idempotency_key: 'ops-supervise-' + ((scope || {}).run_ref || (scope || {}).queue_ref || 'global') + '-' + now
      };
      if (!payload.queue_ref && !payload.run_ref) payload.queue_ref = 'global';
      return await fetchJSON('/api/v0/runs/supervise', {
        method: 'POST',
        headers: {'content-type': 'application/json'},
        body: JSON.stringify(payload)
      });
    }
    async function executeAutoprogrammingSafeActionOps(action, label) {
      if (!action || !action.endpoint) throw new Error('Accion segura sin endpoint.');
      const now = Date.now();
      const payload = safeActionPayload(action, '');
      payload.request_id = payload.request_id || 'ops-safe-action-' + now;
      payload.correlation_id = payload.correlation_id || 'corr-ops-safe-action-' + now;
      payload.requested_by = payload.requested_by || 'orquesta-ops-panel';
      payload.idempotency_key = payload.idempotency_key || 'ops-safe-action-' + (action.run_ref || action.scope || action.action || 'global') + '-' + now;
      const result = await fetchJSON(action.endpoint, {
        method: action.method || 'POST',
        headers: {'content-type': 'application/json'},
        body: JSON.stringify(payload)
      });
      if (String(action.action || '') === 'observe_goal') {
        return goalOpsResultText(result, label);
      }
      return supervisorResultText(result);
    }
    function goalOpsResultText(result, label) {
      result = result || {};
      return 'goal ' + (label || 'observado') +
        ' · estado ' + (result.estado || '-') +
        ' · run ' + shortRef(result.run_ref || '-') +
        ' · goal_status ' + (result.goal_status || '-') +
        ' · cierre ' + (result.closure_status || '-') +
        (result.closure_accepted ? ' · aceptado' : '');
    }
    function supervisorResultText(result) {
      result = result || {};
      const last = result.last || {};
      const history = Array.isArray(result.history) ? result.history : [];
      const diagnostics = Array.isArray(result.diagnostics) ? result.diagnostics : [];
      const nextActions = Array.isArray(result.next_actions) ? result.next_actions : [];
      let message = 'estado ' + (result.estado || '-') +
        ' · run ' + shortRef(result.run_ref || '-') +
        ' · stop_reason ' + (result.stop_reason || '-') +
        ' · ticks ' + String(result.ticks || history.length || 0) +
        ' · last.status ' + (last.status || '-') +
        ' · history ' + String(history.length);
      if (diagnostics.length) message += ' · diagnosticos ' + diagnostics.length;
      if (nextActions.length) message += ' · siguiente ' + nextActions.slice(0, 2).join(' | ');
      return message;
    }
    function effectiveConfigSettings(server) {
      const config = (server || {}).effective_config || {};
      return Array.isArray(config.settings) ? config.settings : [];
    }
    function effectiveConfigSettingValue(settings, key) {
      const found = (settings || []).find(function(setting) { return setting.key === key; });
      return found ? String(found.value == null ? '' : found.value) : '';
    }
    function updatePendingEnv(key, value) {
      const settings = effectiveConfigSettings(lastServer);
      const current = effectiveConfigSettingValue(settings, key);
      if (String(value || '') === current) {
        delete pendingEnvEdits[key];
      } else {
        pendingEnvEdits[key] = String(value || '');
      }
      savePendingEnvEdits();
      adminMessage = 'Cambios guardados como pendientes. No afectan al proceso actual hasta reinicio.';
      renderAdminConfig(lastServer);
    }
    function resetPendingEnv() {
      pendingEnvEdits = {};
      savePendingEnvEdits();
      adminMessage = 'Cambios pendientes descartados en este navegador.';
      renderAdminConfig(lastServer);
    }
    function pendingEnvLines(settings) {
      return (settings || []).filter(function(setting) {
        return Object.prototype.hasOwnProperty.call(pendingEnvEdits, setting.key) &&
          String(pendingEnvEdits[setting.key]) !== String(setting.value == null ? '' : setting.value);
      }).map(function(setting) {
        return setting.key + '=' + String(pendingEnvEdits[setting.key] || '');
      });
    }
    async function requestServerShutdownFromAdmin(forced) {
      const warning = forced ?
        'Cierre forzoso: se pueden perder datos de agentes/runs que no hayan hecho checkpoint. Continuar?' :
        'Solicitar cierre ordenado: Orquesta intentara drenar/parar runs antes del reinicio. Continuar?';
      if (!confirm(warning)) return;
      adminMessage = forced ? 'Solicitando cierre forzoso...' : 'Solicitando cierre ordenado...';
      renderAdminConfig(lastServer);
      try {
        const result = await fetchJSON('/api/v0/server/shutdown', {
          method: 'POST',
          headers: {'content-type': 'application/json'},
          body: JSON.stringify({
            request_id: 'ops-admin-shutdown-' + Date.now(),
            correlation_id: 'corr-ops-admin-shutdown-' + Date.now(),
            forced: !!forced,
            requested_by: 'orquesta-director-admin-web',
            reason: forced ? 'cierre forzoso solicitado desde admin web para reinicio de entorno' : 'cierre ordenado solicitado desde admin web para reinicio de entorno',
            idempotency_key: 'ops-admin-shutdown-' + (forced ? 'forced' : 'ordered') + '-' + Date.now(),
            evidence_refs: ['evidence-ref-ops-admin-env-restart']
          })
        });
        adminMessage = 'Shutdown ' + (result.status || '-') + ' · ready=' + String(!!result.shutdown_ready) +
          ' · runs ' + String(result.runs_stopped || 0) + '/' + String(result.runs_requested || 0) +
          ' · agentes vuelo ' + String(result.agents_in_flight || 0);
      } catch (err) {
        adminMessage = err.message || String(err);
      }
      await refreshAll();
    }
    function renderAdminConfig(server) {
      const settings = effectiveConfigSettings(server);
      if (!settings.length) {
        byId('admin-config').innerHTML = '<div class="sub">Configuracion efectiva no publicada por el servidor.</div>';
        return;
      }
      const lines = pendingEnvLines(settings);
      const rows = settings.map(function(setting) {
        const current = String(setting.value == null ? '' : setting.value);
        const value = Object.prototype.hasOwnProperty.call(pendingEnvEdits, setting.key) ? String(pendingEnvEdits[setting.key]) : current;
        const changed = value !== current;
        return '<div class="config-row ' + (changed ? 'changed' : '') + '">' +
          '<div><div class="task-title">' + esc(setting.label || setting.key) + '</div>' +
          '<div class="task-subtitle mono">' + esc(setting.key) + ' · actual ' + esc(current || '-') + '</div>' +
          '<div class="sub">' + esc(setting.description || '') + '</div></div>' +
          '<input type="text" value="' + esc(value) + '" aria-label="' + esc(setting.key) + '" onchange="updatePendingEnv(\'' + jsArg(setting.key) + '\', this.value)">' +
        '</div>';
      }).join('');
      byId('admin-config').innerHTML =
        '<div class="warning-box">Las variables editadas aqui quedan pendientes: no se aplican al proceso actual hasta reiniciar Orquesta con ese entorno.</div>' +
        rows +
        '<div class="button-row">' +
          '<button type="button" onclick="requestServerShutdownFromAdmin(false)">Cierre ordenado</button>' +
          '<button type="button" onclick="requestServerShutdownFromAdmin(true)">Forzoso</button>' +
          '<button type="button" onclick="resetPendingEnv()">Descartar pendientes</button>' +
        '</div>' +
        (adminMessage ? '<div class="sub">' + esc(adminMessage) + '</div>' : '') +
        '<details data-detail-key="admin-env-pending" ' + (lines.length ? 'open' : '') + '><summary>Variables pendientes para reinicio (' + lines.length + ')</summary><pre>' + esc(lines.length ? lines.join('\n') : 'Sin cambios pendientes') + '</pre></details>';
    }
    function renderServer(server) {
      byId('server-detail').innerHTML = [
        kv('PID', server.pid),
        kv('Dirección', server.addr),
        kv('Heartbeat', server.last_heartbeat_at),
        kv('Supervisor', (server.last_supervisor_status || '-') + ' · tick ' + (server.supervisor_ticks || 0)),
        kv('Automejora', (server.idle_self_improvement_reason || '-') + ' · ' + idleReasonText(server.idle_self_improvement_reason) + ' · ok ' + (server.idle_self_improvement_ok || 0) + '/' + (server.idle_self_improvement_runs || 0)),
        kv('Proyecto', server.project_work_dir),
        kv('Runtime', server.runtime_work_dir)
      ].join('');
    }
    function renderResources(resources) {
      const process = resources.process || {};
      const diskRows = (resources.disks || []).map(function(disk) {
        return kv('Disco ' + (disk.used_percent || 0) + '%', shortRef(disk.path) + ' · libre ' + fmtBytes(disk.available_bytes));
      }).join('');
      byId('resources-detail').innerHTML = [
        kv('RSS', fmtBytes(process.rss_bytes)),
        kv('Go alloc', fmtBytes(process.go_alloc_bytes)),
        kv('Go heap', fmtBytes(process.go_heap_in_use_bytes)),
        kv('Goroutines', process.num_goroutine),
        diskRows || kv('Disco', 'sin datos')
      ].join('');
    }
    function renderWorkspaceSources(timeline) {
      timeline = timeline || {};
      const freshness = timeline.freshness || {};
      const sources = Array.isArray(timeline.sources) ? timeline.sources : [];
      const sourceRows = sources.map(function(source) {
        return kv(source.source || '-', (source.status || '-') + (source.reason_code ? ' · ' + source.reason_code : ''));
      }).join('');
      byId('sources-detail').innerHTML = [
        kv('Timeline', timeline.timeline_ref || timeline.timeline_id || '-'),
        kv('Generado', timeline.generated_at || '-'),
        kv('Frescura', 'watermark ' + (freshness.watermark_ref || '-') + ' · edad ' + String(freshness.max_age_seconds == null ? '-' : freshness.max_age_seconds) + 's · parcial ' + String(!!freshness.partial) + ' · stale ' + String(!!freshness.stale)),
        sourceRows || kv('Fuentes', 'workspace_timeline no publicado')
      ].join('');
    }
    function renderDiagnostics(items) {
      if (!items.length) {
        byId('diagnostics').innerHTML = '<div class="sub">Sin incidencias publicadas.</div>';
        return;
      }
      byId('diagnostics').innerHTML = items.slice(0, 12).map(function(item) {
        if (typeof item === 'string') return '<div class="error">' + esc(item) + '</div>';
        return '<div class="kv"><div class="k">' + esc(item.code || '-') + '</div><div>' + esc(item.message || item.scope || item.field || '-') + '</div></div>';
      }).join('');
    }
    function kv(k, v) {
      return '<div class="kv"><div class="k">' + esc(k) + '</div><div class="mono" title="' + esc(v || '-') + '">' + esc(v == null || v === '' ? '-' : v) + '</div></div>';`
