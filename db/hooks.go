package db

import (
	"strings"
	"sync"
	"sync/atomic"
)

type EventoHook string

type LifecycleHookEvent struct {
	Actor      string
	Evento     EventoHook
	ProyectoID int64
	Entidad    string
	EntidadID  int64
	Agente     string
	Detalle    string
}

const (
	HookProjectBlocked   EventoHook = "project_blocked"
	HookProjectUnblocked EventoHook = "project_unblocked"
	HookSessionPark      EventoHook = "session_park"
	HookSessionResume    EventoHook = "session_resume"
	HookTaskStart        EventoHook = "task_start"
	HookTaskFinish       EventoHook = "task_finish"
)

var lifecycleHookSubscribersMu sync.RWMutex
var lifecycleHookSubscribers = map[uint64]func(LifecycleHookEvent){}
var lifecycleHookSubscriberSeq atomic.Uint64

func RegisterLifecycleHookSubscriber(fn func(LifecycleHookEvent)) func() {
	if fn == nil {
		return func() {}
	}
	id := lifecycleHookSubscriberSeq.Add(1)
	lifecycleHookSubscribersMu.Lock()
	lifecycleHookSubscribers[id] = fn
	lifecycleHookSubscribersMu.Unlock()
	return func() {
		lifecycleHookSubscribersMu.Lock()
		delete(lifecycleHookSubscribers, id)
		lifecycleHookSubscribersMu.Unlock()
	}
}

func emitLifecycleHookSubscribers(ev LifecycleHookEvent) {
	lifecycleHookSubscribersMu.RLock()
	subs := make([]func(LifecycleHookEvent), 0, len(lifecycleHookSubscribers))
	for _, fn := range lifecycleHookSubscribers {
		if fn != nil {
			subs = append(subs, fn)
		}
	}
	lifecycleHookSubscribersMu.RUnlock()
	for _, fn := range subs {
		func(run func(LifecycleHookEvent)) {
			defer func() {
				_ = recover()
			}()
			run(ev)
		}(fn)
	}
}

func EmitirHookCicloVida(actor string, evento EventoHook, proyectoID int64, entidad string, entidadID int64, agente string, detalle string) {
	nombreEvento := strings.TrimSpace(string(evento))
	if nombreEvento == "" {
		return
	}
	ev := LifecycleHookEvent{
		Actor:      strings.TrimSpace(actor),
		Evento:     evento,
		ProyectoID: proyectoID,
		Entidad:    strings.TrimSpace(entidad),
		EntidadID:  entidadID,
		Agente:     strings.TrimSpace(agente),
		Detalle:    strings.TrimSpace(detalle),
	}
	Audit(strings.TrimSpace(actor), "hook_"+nombreEvento, strings.TrimSpace(entidad), entidadID, strings.TrimSpace(detalle))
	EmitirNotificacion(EventoNotificacion{
		Tipo:       "hook:" + nombreEvento,
		ID:         entidadID,
		Agente:     strings.TrimSpace(agente),
		Texto:      strings.TrimSpace(detalle),
		ProyectoID: proyectoID,
	})
	emitLifecycleHookSubscribers(ev)
}
