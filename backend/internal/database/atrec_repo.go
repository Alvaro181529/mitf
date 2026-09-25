package database

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"mitfv2/models"
)

// ATRECRepository define las operaciones de persistencia para la bitácora ATREC.
type ATRECRepository interface {
	GetAll(statusFilter string) ([]models.TicketATREC, error)
	GetByID(ticketID string) (*models.TicketATREC, error)
	Create(ticket models.TicketATREC) (*models.TicketATREC, error)
	Update(ticket models.TicketATREC) (*models.TicketATREC, error)
	GenerateTicketID() string
}

// MemoryATRECRepository implementa ATRECRepository de forma thread-safe con sync.RWMutex.
type MemoryATRECRepository struct {
	mu      sync.RWMutex
	tickets map[string]models.TicketATREC
	counter int
}

// NewMemoryATRECRepository crea una nueva instancia en memoria con datos iniciales de ejemplo si se desea.
func NewMemoryATRECRepository() *MemoryATRECRepository {
	repo := &MemoryATRECRepository{
		tickets: make(map[string]models.TicketATREC),
		counter: 100,
	}

	// Sembrar algunos tickets de ejemplo para pruebas de entorno
	now := time.Now().UTC()
	t1 := models.TicketATREC{
		TicketID:           "ATREC-2026-101",
		CreatedAt:          now.Add(-2 * time.Hour),
		UpdatedAt:          now.Add(-2 * time.Hour),
		TSMSubsystemID:     1,
		Severity:           models.SeverityCritical,
		SourceErrorCode:    "260199001",
		Description:        "Arco de filamento en tubo de rayos X detectado en rotación sostenida",
		AssignedTechnician: "Ing. Carlos Mendoza",
		Status:             models.StatusOpen,
		PostRepairQA: models.PostRepairQA{
			ZAlignmentPassed:            false,
			PhantomIQPassed:             false,
			DetectorFlatfieldCalibrated: false,
			QASignOffBy:                 "",
			ResolutionType:              "",
		},
	}
	t2 := models.TicketATREC{
		TicketID:           "ATREC-2026-102",
		CreatedAt:          now.Add(-5 * time.Hour),
		UpdatedAt:          now.Add(-1 * time.Hour),
		TSMSubsystemID:     2,
		Severity:           models.SeverityMajor,
		SourceErrorCode:    "260144012",
		Description:        "Desviación de jitter en anillos deslizantes del rotor",
		AssignedTechnician: "Tec. Laura Gómez",
		Status:             models.StatusPendingCalibration,
		PostRepairQA: models.PostRepairQA{
			ZAlignmentPassed:            true,
			PhantomIQPassed:             false,
			DetectorFlatfieldCalibrated: false,
			QASignOffBy:                 "Tec. Laura Gómez",
			ResolutionType:              models.ResolutionTroubleshootingL1,
		},
	}
	repo.tickets[t1.TicketID] = t1
	repo.tickets[t2.TicketID] = t2
	repo.counter = 103

	return repo
}

// GenerateTicketID genera un ID correlativo con el año actual (ej: "ATREC-2026-103").
func (r *MemoryATRECRepository) GenerateTicketID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	year := time.Now().Year()
	id := fmt.Sprintf("ATREC-%d-%03d", year, r.counter)
	r.counter++
	return id
}

// GetAll obtiene todos los tickets, ordenados descendentemente por fecha de creación, opcionalmente filtrados por status.
func (r *MemoryATRECRepository) GetAll(statusFilter string) ([]models.TicketATREC, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.TicketATREC, 0, len(r.tickets))
	statusFilterUpper := strings.ToUpper(strings.TrimSpace(statusFilter))

	for _, t := range r.tickets {
		if statusFilterUpper != "" && strings.ToUpper(t.Status) != statusFilterUpper {
			continue
		}
		result = append(result, t)
	}

	// Ordenar del más reciente al más antiguo
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// GetByID busca un ticket por su identificador único.
func (r *MemoryATRECRepository) GetByID(ticketID string) (*models.TicketATREC, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cleanID := strings.TrimSpace(ticketID)
	ticket, exists := r.tickets[cleanID]
	if !exists {
		// Buscar también ignorando mayúsculas/minúsculas
		for _, t := range r.tickets {
			if strings.EqualFold(t.TicketID, cleanID) {
				return &t, nil
			}
		}
		return nil, fmt.Errorf("ticket con ID '%s' no encontrado", ticketID)
	}

	return &ticket, nil
}

// Create inserta un nuevo ticket en el mapa de memoria.
func (r *MemoryATRECRepository) Create(ticket models.TicketATREC) (*models.TicketATREC, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tickets[ticket.TicketID]; exists {
		return nil, fmt.Errorf("ticket con ID '%s' ya existe", ticket.TicketID)
	}

	r.tickets[ticket.TicketID] = ticket
	return &ticket, nil
}

// Update actualiza un ticket existente.
func (r *MemoryATRECRepository) Update(ticket models.TicketATREC) (*models.TicketATREC, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cleanID := strings.TrimSpace(ticket.TicketID)
	if _, exists := r.tickets[cleanID]; !exists {
		// Buscar con case insensitive
		found := false
		for k, t := range r.tickets {
			if strings.EqualFold(t.TicketID, cleanID) {
				cleanID = k
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("ticket con ID '%s' no encontrado para actualizar", ticket.TicketID)
		}
	}

	ticket.UpdatedAt = time.Now().UTC()
	r.tickets[cleanID] = ticket
	return &ticket, nil
}
