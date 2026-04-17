package reviewapp

import (
	"fmt"
	"strings"
	"time"
)

const (
	GateStatePending        = "pendiente"
	GateStateInReview       = "en_revision"
	GateStateChangesAsked   = "cambios_pedidos"
	GateStateApproved       = "aprobado"
	GateStateBlocked        = "bloqueado"
	defaultRequestedByActor = "orquesta"
)

var (
	validGateStates = map[string]struct{}{
		GateStatePending:      {},
		GateStateInReview:     {},
		GateStateChangesAsked: {},
		GateStateApproved:     {},
		GateStateBlocked:      {},
	}
	validSeverity = map[string]struct{}{
		"":         {},
		"low":      {},
		"medium":   {},
		"high":     {},
		"critical": {},
	}
)

type Gate struct {
	ID             int64      `json:"id"`
	ProyectoID     int64      `json:"proyecto_id"`
	ProyectoSlug   string     `json:"proyecto_slug,omitempty"`
	TareaID        *int64     `json:"tarea_id,omitempty"`
	WorktreeID     *int64     `json:"worktree_id,omitempty"`
	RequestedBy    string     `json:"requested_by"`
	ReviewerAgente string     `json:"reviewer_agente,omitempty"`
	Estado         string     `json:"estado"`
	SeverityMax    string     `json:"severity_max,omitempty"`
	FindingsJSON   string     `json:"findings_json,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

type GateFilter struct {
	ProyectoID     *int64
	Estado         *string
	ReviewerAgente *string
	Limit          int
}

type ProjectRef struct {
	ID   int64  `json:"id"`
	Slug string `json:"slug"`
}

type Store interface {
	GetProject(ref string) (*ProjectRef, error)
	GetReviewGate(id int64) (*Gate, error)
	ListReviewGates(filter GateFilter) ([]*Gate, error)
	CreateReviewGate(gate *Gate) (int64, error)
	UpdateReviewGate(gate *Gate) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type CreateGateInput struct {
	ProyectoRef     string
	TareaID         *int64
	WorktreeID      *int64
	RequestedBy     string
	ReviewerAgente  string
	SeverityMax     string
	InitialFindings string
}

type ListInput struct {
	ProyectoRef    string
	Estado         string
	ReviewerAgente string
	Limit          int
}

type ResolveGateInput struct {
	ID             int64
	Estado         string
	ReviewerAgente string
	FindingsJSON   string
}

type UpdateGateInput struct {
	ID             int64
	Estado         string
	ReviewerAgente *string
	TareaID        *int64
	WorktreeID     *int64
	SeverityMax    *string
	FindingsJSON   *string
}

func NormalizeGateState(v string) (string, error) {
	state := strings.ToLower(strings.TrimSpace(v))
	if _, ok := validGateStates[state]; !ok {
		return "", fmt.Errorf("estado de gate no soportado: %s", strings.TrimSpace(v))
	}
	return state, nil
}

func CanTransitionGateState(from, to string) bool {
	from = strings.ToLower(strings.TrimSpace(from))
	to = strings.ToLower(strings.TrimSpace(to))
	switch from {
	case GateStatePending:
		return to == GateStateInReview || to == GateStateChangesAsked || to == GateStateApproved || to == GateStateBlocked
	case GateStateInReview:
		return to == GateStateChangesAsked || to == GateStateApproved || to == GateStateBlocked
	case GateStateChangesAsked:
		return to == GateStateInReview || to == GateStateApproved || to == GateStateBlocked
	case GateStateBlocked:
		return to == GateStatePending || to == GateStateInReview || to == GateStateChangesAsked || to == GateStateApproved
	case GateStateApproved:
		return false
	default:
		return false
	}
}

func IsResolvedGateState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), GateStateApproved)
}

func IsBlockingGateState(state string) bool {
	state = strings.ToLower(strings.TrimSpace(state))
	return state == GateStateBlocked || state == GateStateChangesAsked
}

func GateNeedsReviewAction(gate *Gate) bool {
	if gate == nil {
		return false
	}
	return IsBlockingGateState(gate.Estado) || strings.EqualFold(strings.TrimSpace(gate.Estado), GateStatePending)
}

func NormalizeSeverity(v string) (string, error) {
	level := strings.ToLower(strings.TrimSpace(v))
	if _, ok := validSeverity[level]; !ok {
		return "", fmt.Errorf("severity_max no soportada: %s", strings.TrimSpace(v))
	}
	return level, nil
}

func (s *Service) List(input ListInput) ([]*Gate, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("store no configurado")
	}
	filter := GateFilter{Limit: input.Limit}
	if ref := strings.TrimSpace(input.ProyectoRef); ref != "" {
		proyecto, err := s.store.GetProject(ref)
		if err != nil {
			return nil, err
		}
		if proyecto == nil {
			return []*Gate{}, nil
		}
		filter.ProyectoID = &proyecto.ID
	}
	if estado := strings.TrimSpace(input.Estado); estado != "" {
		norm, err := NormalizeGateState(estado)
		if err != nil {
			return nil, err
		}
		filter.Estado = &norm
	}
	if reviewer := strings.TrimSpace(input.ReviewerAgente); reviewer != "" {
		filter.ReviewerAgente = &reviewer
	}
	return s.store.ListReviewGates(filter)
}

func (s *Service) Create(input CreateGateInput) (*Gate, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("store no configurado")
	}
	proyecto, err := s.store.GetProject(strings.TrimSpace(input.ProyectoRef))
	if err != nil {
		return nil, err
	}
	if proyecto == nil {
		return nil, fmt.Errorf("proyecto no encontrado: %s", strings.TrimSpace(input.ProyectoRef))
	}
	severity, err := NormalizeSeverity(input.SeverityMax)
	if err != nil {
		return nil, err
	}
	requestedBy := strings.TrimSpace(input.RequestedBy)
	if requestedBy == "" {
		requestedBy = defaultRequestedByActor
	}
	now := time.Now().UTC()
	gate := &Gate{
		ProyectoID:     proyecto.ID,
		ProyectoSlug:   strings.TrimSpace(proyecto.Slug),
		TareaID:        input.TareaID,
		WorktreeID:     input.WorktreeID,
		RequestedBy:    requestedBy,
		ReviewerAgente: strings.TrimSpace(input.ReviewerAgente),
		Estado:         GateStatePending,
		SeverityMax:    severity,
		FindingsJSON:   strings.TrimSpace(input.InitialFindings),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	id, err := s.store.CreateReviewGate(gate)
	if err != nil {
		return nil, err
	}
	gate.ID = id
	return gate, nil
}

func (s *Service) Resolve(input ResolveGateInput) (*Gate, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("store no configurado")
	}
	if input.ID <= 0 {
		return nil, fmt.Errorf("id de gate obligatorio")
	}
	next, err := NormalizeGateState(input.Estado)
	if err != nil {
		return nil, err
	}
	gate, err := s.store.GetReviewGate(input.ID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, fmt.Errorf("review gate no encontrado: %d", input.ID)
	}
	current, err := NormalizeGateState(gate.Estado)
	if err != nil {
		return nil, err
	}
	if current != next && !CanTransitionGateState(current, next) {
		return nil, fmt.Errorf("transición de gate no permitida: %s -> %s", current, next)
	}
	gate.Estado = next
	if reviewer := strings.TrimSpace(input.ReviewerAgente); reviewer != "" {
		gate.ReviewerAgente = reviewer
	}
	if strings.TrimSpace(input.FindingsJSON) != "" {
		gate.FindingsJSON = strings.TrimSpace(input.FindingsJSON)
	}
	now := time.Now().UTC()
	gate.UpdatedAt = now
	if next == GateStateApproved {
		gate.ResolvedAt = &now
	}
	if next != GateStateApproved {
		gate.ResolvedAt = nil
	}
	if err := s.store.UpdateReviewGate(gate); err != nil {
		return nil, err
	}
	return gate, nil
}

func (s *Service) Update(input UpdateGateInput) (*Gate, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("store no configurado")
	}
	if input.ID <= 0 {
		return nil, fmt.Errorf("id de gate obligatorio")
	}
	gate, err := s.store.GetReviewGate(input.ID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, fmt.Errorf("review gate no encontrado: %d", input.ID)
	}
	if input.ReviewerAgente != nil {
		gate.ReviewerAgente = strings.TrimSpace(*input.ReviewerAgente)
	}
	if input.TareaID != nil {
		gate.TareaID = input.TareaID
	}
	if input.WorktreeID != nil {
		gate.WorktreeID = input.WorktreeID
	}
	if input.SeverityMax != nil {
		severity, err := NormalizeSeverity(*input.SeverityMax)
		if err != nil {
			return nil, err
		}
		gate.SeverityMax = severity
	}
	if input.FindingsJSON != nil {
		gate.FindingsJSON = strings.TrimSpace(*input.FindingsJSON)
	}
	if strings.TrimSpace(input.Estado) != "" {
		return s.Resolve(ResolveGateInput{
			ID:             input.ID,
			Estado:         input.Estado,
			ReviewerAgente: gate.ReviewerAgente,
			FindingsJSON:   gate.FindingsJSON,
		})
	}
	gate.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateReviewGate(gate); err != nil {
		return nil, err
	}
	return gate, nil
}
